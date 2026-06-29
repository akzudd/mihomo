# 移植 provider dialer-proxy 行为

Status: ready-for-agent

## 背景

需要从 `dev` 分支提交 `d5ad2279731ab44e88eee8a94892b7812135791e` 移植 provider `dialer-proxy` 相关行为。只移植行为，不移植 `build.txt` 和 `constant/version.go`。

详见同目录 [PRD](../PRD.md)。

## 范围

- 调整 `adapter/provider/provider.go` 中 provider 导入代理时的 `dialer-proxy` 注入和改名顺序。
- 调整 `component/proxydialer/byname.go` 中按名称解析代理的范围。
- 按需增加测试文件，覆盖 PRD 中的验收标准。

## 实现约束

- 全局代理命中时必须优先使用全局代理。
- provider 内部代理多命中时必须返回稳定错误，不能依赖 map 遍历顺序选择其中一个。
- 不改变 provider 过滤、去重、健康检查、更新和 proxy group 解析行为。
- 不修改版本号，不新增本地构建记录。

## 验收标准

- provider 级 `dialer-proxy` 会随 provider override 的 `additional-prefix` 和 `additional-suffix` 改成最终引用名。
- 节点自带 `dialer-proxy` 会随 provider override 的 `additional-prefix` 和 `additional-suffix` 改成最终引用名。
- `dialer-proxy` 能解析 provider 内部代理。
- 全局代理与 provider 内部代理同名时，全局代理优先。
- 多个 provider 内部代理同名且全局代理不存在时，返回包含目标代理名的歧义错误。
- `git diff` 不包含 `build.txt` 和 `constant/version.go`。

## 建议验证

- `go test ./adapter/provider ./component/proxydialer`
- 如新增测试需要跨包替身，可改跑更小或更大的受影响包测试，并在 closeout 中说明原因。

## Comments
