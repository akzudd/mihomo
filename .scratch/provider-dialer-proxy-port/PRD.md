# PRD: 移植 provider dialer-proxy 行为

## 背景

`dev` 分支提交 `d5ad2279731ab44e88eee8a94892b7812135791e` 引入了 provider 级 `dialer-proxy` 与 provider 内部代理解析能力。当前分支需要移植其中的行为变更，但不移植版本号和本地构建记录。

当前代码在 provider 导入代理时，会先写入 provider 级 `dialer-proxy`，再执行 provider override。若 override 为代理名添加 `additional-prefix` 或 `additional-suffix`，导入后的代理名会改变，但 `dialer-proxy` 引用仍保持原始名称，导致引用目标不一致。

## 目标

- provider 级 `dialer-proxy` 在注入到代理后，应引用导入后的最终代理名。
- 节点自带的 `dialer-proxy` 在 provider override 改名后，也应跟随同一套 `additional-prefix` / `additional-suffix`。
- `dialer-proxy` 运行时解析按名称查找代理时，应先查全局代理，再查 provider 内部代理。
- 当多个 provider 内部代理同名且全局代理不存在时，应返回明确错误，不依赖 Go map 遍历顺序。

## 非目标

- 不修改 `constant/version.go`。
- 不新增或移植 `build.txt`。
- 不改变 proxy group 的 `dialer-proxy` 禁用规则。
- 不改变 provider 导入代理的过滤、去重、健康检查和 provider 更新语义。

## 术语

- `dialer-proxy`：代理使用的底层拨号代理引用，按代理名称解析。
- provider 级 `dialer-proxy`：定义在 proxy provider 上的默认 `dialer-proxy`。
- provider override：proxy provider 导入代理时应用的字段改写规则。
- imported proxy name：代理导入并完成 override 后的最终名称。

## 行为要求

### provider 导入阶段

provider 导入代理时，`dialer-proxy` 处理顺序应满足：

1. 先识别代理自身是否已有 `dialer-proxy`。
2. 若代理自身没有 `dialer-proxy`，且 provider 配置了 `dialer-proxy`，则使用 provider 级值。
3. provider override 应继续控制代理最终名称。
4. 若最终存在 `dialer-proxy`，并且 override 配置了 `additional-prefix` 或 `additional-suffix`，则该 `dialer-proxy` 引用也应用相同前缀或后缀。

### 运行时解析阶段

按名称解析 `dialer-proxy` 时，查找顺序为：

1. 全局代理 map。
2. provider 内部代理列表。

如果全局代理命中，直接使用全局代理，不检查 provider 内部同名代理。

如果全局代理未命中：

- provider 内部代理命中 1 个时，使用该代理。
- provider 内部代理命中 0 个时，返回未找到错误。
- provider 内部代理命中多个时，返回歧义错误，错误信息应包含目标代理名。

## 验收标准

- provider 级 `dialer-proxy` 会随 `additional-prefix` 和 `additional-suffix` 变成最终引用名。
- 节点自带 `dialer-proxy` 会随 `additional-prefix` 和 `additional-suffix` 变成最终引用名。
- `dialer-proxy` 可以引用 provider 内部代理。
- 全局代理与 provider 内部代理同名时，全局代理优先。
- 多个 provider 内部代理同名且全局代理不存在时，返回稳定错误。
- 移植不包含 `build.txt` 和 `constant/version.go`。

## 测试要求

- 为 provider parser 增加单元测试，覆盖 provider 级 `dialer-proxy`、节点级 `dialer-proxy`、prefix、suffix 和组合情况。
- 为 by-name proxy dialer 增加单元测试或小范围测试替身，覆盖全局优先、provider 内部单一命中、未命中和多命中歧义。
- 运行受影响包的 Go 测试；若测试范围扩大，应说明原因。

## 参考提交

- `d5ad2279731ab44e88eee8a94892b7812135791e`，提交信息：`provider proxy dialer`
