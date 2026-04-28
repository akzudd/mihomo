# Reality `short-id` YAML 解析修复设计

## 背景

`RealityOptions.ShortID` 语义上是十六进制字符串，但当 YAML 中写成未加引号的值，例如：

```yaml
short-id: 7266e6
```

会先被 `gopkg.in/yaml.v3` 识别为科学计数法数值，再在弱类型结构解码阶段被格式化成字符串 `7.266E+09`，最终导致 REALITY 的十六进制解析失败。

约束条件：**不能要求用户修改 YAML 文件本身**。

---

## 根因总结

问题链路如下：

1. `common/yaml/yaml.go` 直接调用 `yaml.Unmarshal(...)`
2. YAML 将未加引号的 `7266e6` 识别为数值标量，而不是字符串标量
3. 代理/监听器解析使用 `structure.NewDecoder(... WeaklyTypedInput: true)`
4. `common/structure/structure.go` 在 `decodeString(...)` 中把浮点值格式化为科学计数法字符串
5. REALITY 解析器收到的不是原始 `7266e6`，而是 `7.266E+09`
6. `hex.Decode(...)` 失败

因此，真正的修复必须发生在**原始 YAML token 仍然可见**的阶段，也就是 `common/yaml` 入口层，而不是结构解码层或 REALITY 协议层。

---

## 目标

1. 不修改用户 YAML 文件
2. 同时修复 outbound 和 inbound 的 REALITY `short-id`
3. 不影响普通数值字段、布尔字段和其他字符串字段
4. 不改动现有 REALITY 十六进制校验逻辑
5. 不改变全局 `WeaklyTypedInput` 策略

---

## 关键设计决策

### 决策 1：在 `common/yaml` 做 AST 级预处理

**结论：采用。**

设计：
- 先把 YAML 解析成 `yaml.Node`
- 遍历文档树
- 遇到 key 为 `short-id` 的节点时，把对应 value 强制保留为字符串语义
- 再从处理后的 AST decode 到现有目标对象

原因：
- 原始词法值 `7266e6` 只有在 YAML AST 阶段仍然存在
- 一旦完成普通 `Unmarshal` 进入 `map[string]any`，原始词法就丢失了，后续无法可靠恢复

trade-off：
- 优点：修复层正确、影响面小、对上层透明
- 缺点：`common/yaml` 不再只是薄封装，需要引入一层 YAML AST 处理逻辑

### 决策 2：按键名全局匹配 `short-id`

**结论：采用。**

设计：
- 不做精确路径匹配
- 只按 key 名 `short-id` 处理
- 作用于整个 YAML 文档中所有同名字段

覆盖范围：
- outbound `reality-opts.short-id`
- inbound `reality.short-id`

trade-off：
- 优点：实现简单、稳定、对 schema 结构变动不敏感
- 缺点：未来如果出现语义不同的 `short-id` 字段，也会被按字符串处理

当前判断：该 trade-off 可接受。

### 决策 3：同时支持 scalar 和 sequence

**结论：采用。**

需要支持：

```yaml
short-id: 7266e6
```

```yaml
short-id: [7266e6]
```

```yaml
short-id:
  - 7266e6
```

设计：
- 如果 `short-id` 的 value 是 scalar，则强制保留为字符串
- 如果 `short-id` 的 value 是 sequence，则对其中每个 scalar item 强制保留为字符串

trade-off：
- 优点：一次覆盖出站和入站
- 缺点：AST 处理需要多一个 sequence 分支，但复杂度很低

### 决策 4：不在 `common/structure` 或 REALITY 层补偿

**结论：不采用。**

不采用的原因：
- 修改 `decodeString(...)` 的 float → string 格式化规则，无法恢复原始词法 `7266e6`
- 在 REALITY 解析器中对 `7.266E+09` 做逆推属于不可靠推断，会污染协议层职责
- 关闭全局 `WeaklyTypedInput` 会带来高风险回归

---

## 文件 / 模块变更清单

### 需要修改

#### 1. `common/yaml/yaml.go`

当前行为：
- 直接调用 `yaml.Unmarshal(in, out)`

设计变更：
- 改为：
  1. 先解析为 `yaml.Node`
  2. 执行 `short-id` 字符串保留预处理
  3. 再将处理后的 AST decode 到目标对象

### 建议新增

#### 2. `common/yaml/preserve_string_keys.go`

职责：
- 实现 YAML AST 遍历
- 对 key=`short-id` 的 value 做字符串语义保留
- 同时处理 scalar 和 sequence 两种形态

说明：
- 也可命名为 `normalize.go`，但建议文件名体现其职责

#### 3. `common/yaml/preserve_string_keys_test.go`

测试职责：
- 验证 `short-id: 7266e6` 最终被解码为字符串 `"7266e6"`
- 验证 `short-id: [7266e6]` 的数组元素最终为字符串 `"7266e6"`
- 验证非 `short-id` 的普通数值字段不受影响
- 验证非 `short-id` 的其他字符串字段不受影响

#### 4. `config/reality_shortid_test.go`（或同等位置的新回归测试）

测试职责：
- 覆盖完整配置解析链路，而不是只测局部 helper
- 验证 outbound 和 inbound 两条链路都修复

建议场景：
- outbound `reality-opts.short-id: 7266e6`
- inbound `reality.short-id: [7266e6]`

### 明确不建议修改

- `common/structure/structure.go`
- `adapter/outbound/reality.go`
- `listener/reality/reality.go`
- `adapter/parser.go`
- `listener/parse.go`

原因：这些位置都不是正确修复层。

---

## 实现顺序

### 阶段 1：先补回归测试

目标：
- 把 bug 固化为 parser-level regression test

内容：
- 新增完整配置解析测试，确保问题路径真实经过：
  - `common/yaml`
  - `config.Parse(...)`
  - `structure.NewDecoder(... WeaklyTypedInput: true)`

风险点：
- 最小可解析配置可能被其他必填项干扰
- 需要避免只测到 struct 构造而没测到 YAML 解析入口

### 阶段 2：实现 `common/yaml` AST 预处理

目标：
- 在原始词法丢失前修复 `short-id`

内容：
- 遍历 `yaml.Node`
- 匹配 key=`short-id`
- 处理 scalar / sequence
- 只对 scalar 节点做字符串语义保留，不新增 schema 校验

风险点：
1. 只修 scalar，漏掉 sequence
2. 误处理非 scalar sequence 元素
3. 作用范围过宽，影响其他字段

### 阶段 3：补 `common/yaml` 单元测试

目标：
- 单独验证 AST 修复逻辑自身

内容：
- 单值 `short-id`
- 数组 `short-id`
- 邻近普通数值字段
- 非 `short-id` 字段

风险点：
- 如果测试粒度过粗，可能难以定位是 AST 逻辑问题还是后续 decode 问题

### 阶段 4：定向回归验证

目标：
- 确认修复没有破坏现有配置解析行为

建议验证范围：
- `common/yaml`
- `config`
- REALITY 相关 inbound/outbound 测试

风险点：
- 现有大量 REALITY 测试可能是直接构造 struct，无法替代 parser-level regression test

---

## 已确认的实施口径

以下口径已确认：

1. 修复范围只限于 REALITY `short-id`
2. 必须同时覆盖入站和出站
3. 匹配策略采用方案 A：按键名 `short-id` 全局匹配
4. 接受在 `common/yaml` 引入 AST 预处理复杂度
5. 接受新增 parser-level regression test
6. `short-id` 数组中如果元素不是 scalar，不在 YAML 层新增报错，交给后续现有解析逻辑处理
7. AST 预处理可按需要同时调整 tag/style，但目标只是确保最终 decode 为 string

---

## 验收标准

1. `short-id: 7266e6` 最终进入 outbound `RealityOptions.ShortID` 时为 `"7266e6"`
2. `short-id: [7266e6]` 最终进入 inbound `RealityConfig.ShortID` 时数组元素为 `"7266e6"`
3. 现有普通数值字段仍保持数值语义
4. 未命中 `short-id` 的字段不受影响
5. REALITY 原有 hex 校验逻辑无需改动即可通过

---

## 实施 checklist

### 阶段 1：先锁定回归

- [ ] 新增 parser-level regression test
- [ ] 覆盖 outbound 场景：`reality-opts.short-id: 7266e6`
- [ ] 覆盖 inbound 场景：`short-id: [7266e6]`
- [ ] 断言当前行为确实失败，且失败原因来自字符串被变成科学计数法

完成标准：
- 测试能稳定复现 bug
- 测试路径真实经过 `common/yaml` 和 `WeaklyTypedInput`

### 阶段 2：实现 YAML 入口预处理

- [ ] 在 `common/yaml` 引入 `yaml.Node` 解析入口
- [ ] 增加 AST 遍历逻辑
- [ ] 匹配所有 key=`short-id`
- [ ] 若 value 是 scalar，强制保留为字符串语义
- [ ] 若 value 是 sequence，对每个 scalar item 强制保留为字符串语义
- [ ] 非 scalar item 不报新错，保持原样交给后续解析

完成标准：
- 不改 YAML 文件内容前提下，`7266e6` 能以原样字符串进入后续 decode

重点风险：
- 漏掉 sequence 分支
- 误伤非 `short-id` 字段

### 阶段 3：补 YAML 层单元测试

- [ ] 测 `short-id: 7266e6`
- [ ] 测 `short-id: [7266e6]`
- [ ] 测普通数值字段不受影响
- [ ] 测普通字符串字段不受影响
- [ ] 测嵌套 map 中的 `short-id` 也生效

完成标准：
- 只修 `short-id`
- 不改变其他字段类型语义

### 阶段 4：验证完整链路

- [ ] 回跑阶段 1 的 regression test
- [ ] 验证 outbound `RealityOptions.ShortID == "7266e6"`
- [ ] 验证 inbound `RealityConfig.ShortID[0] == "7266e6"`
- [ ] 验证 REALITY 原有 hex 校验逻辑无需修改即可通过

完成标准：
- 修复后不需要改 `adapter/outbound/reality.go` 或 `listener/reality/reality.go`

### 阶段 5：做最小回归检查

- [ ] 跑 `common/yaml` 相关测试
- [ ] 跑 `config` 相关测试
- [ ] 跑 REALITY 相关 inbound/outbound 定向测试
- [ ] 如有必要，再跑一次 `go test ./... -run ...` 的定向集合

完成标准：
- 没有新增解析层回归
- 普通配置解析路径保持原样

### 文件级执行顺序

1. `config/reality_shortid_test.go` 或同等回归测试文件
2. `common/yaml/yaml.go`
3. `common/yaml/preserve_string_keys.go`
4. `common/yaml/preserve_string_keys_test.go`
5. 回跑定向测试

### 实施时不要做的事

- [ ] 不改 `common/structure/structure.go`
- [ ] 不改 `adapter/outbound/reality.go`
- [ ] 不改 `listener/reality/reality.go`
- [ ] 不关闭全局 `WeaklyTypedInput`
- [ ] 不要求用户给 YAML 加引号

### 最终验收清单

- [ ] `short-id: 7266e6` → outbound 仍是 `"7266e6"`
- [ ] `short-id: [7266e6]` → inbound 元素仍是 `"7266e6"`
- [ ] 非 `short-id` 数值字段仍是数值
- [ ] 非 `short-id` 字符串字段不受影响
- [ ] REALITY 现有校验逻辑无需改动
