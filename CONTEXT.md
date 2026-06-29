# Mihomo

Mihomo 是一个代理运行时，配置中的代理、代理集合和拨号引用共同决定出站连接路径。

## Language

**dialer-proxy**:
代理使用的底层拨号代理引用。该引用按代理名称解析，并且应指向导入和重命名后的最终代理名。
_Avoid_: relay, upstream proxy

**Provider-level dialer-proxy**:
定义在 proxy provider 上的默认 `dialer-proxy`，用于该 provider 导入的代理没有显式拨号引用时。
_Avoid_: global dialer proxy

**Provider override**:
proxy provider 在导入代理时应用的代理字段改写规则。名称前缀和后缀属于 provider override 的一部分，会改变导入后的最终代理名。
_Avoid_: post-processing, patch

**Imported proxy name**:
代理从 proxy provider 导入并完成 override 后的最终名称。其他配置引用 provider 导入代理时，应引用该名称。
_Avoid_: original proxy name
