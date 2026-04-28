# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build, lint, and test

- Requirements: Go 1.20+ (`go.mod`), but CI validates a wider matrix from Go 1.20 through 1.26.
- Download dependencies:
  - `go mod download`
- Build the default binary in the repo root:
  - `go build`
- Build with the gVisor TUN stack tag used by CI and release builds:
  - `go build -tags with_gvisor`
- Build via Makefile with release-style flags and embedded version/build time:
  - `make linux-amd64-v3`
  - `make windows-amd64-v3`
- Lint:
  - `golangci-lint run ./...`
  - `make lint`
- Main test suite:
  - `go test ./... -v -count=1`
- Main test suite with gVisor tag:
  - `go test ./... -v -count=1 -tags with_gvisor`
- Run a single package test:
  - `go test ./listener/inbound -run TestVmess -v -count=1`
- Run a single test file pattern across the repo:
  - `go test ./... -run TestName -v -count=1`
- Separate protocol integration suite under `test/` (requires Docker; README says Linux/macOS support):
  - `make -C test test`
  - `make -C test benchmark`
- Validate a config file without starting the daemon:
  - `go run . -t -f path/to/config.yaml`
- Show version / enabled build tags:
  - `go run . -v`

## CI and release facts

- GitHub Actions runs tests on Linux, Windows, and macOS across Go 1.20-1.26, both with and without `with_gvisor`.
- CI removes `listener/inbound/*_test.go` on macOS before running the main suite, so platform-specific listener tests may not be portable as-is.
- Release builds and most Makefile targets compile with `-tags with_gvisor`, `CGO_ENABLED=0`, `-trimpath`, and injected `constant.Version` / `constant.BuildTime` ldflags.
- `.golangci.yaml` enables only `gofumpt`, `staticcheck`, `govet`, and `gci`.

## High-level architecture

### Runtime startup flow

- `main.go` is the only top-level entrypoint for the daemon.
- Startup path is:
  1. parse CLI flags / env overrides
  2. resolve config source (`-f`, `-config`, stdin, or default home dir config)
  3. parse config through `hub.Parse(...)`
  4. apply runtime state via `hub.ApplyConfig(...)`
  5. register geo updater if enabled
  6. stay resident and reload on `SIGHUP`
- `main.go` also exposes non-daemon subcommands:
  - `convert-ruleset`
  - `generate`

### Config parsing vs runtime application

- `config/config.go` is the central schema and parser for the YAML config.
- The parser is split into two phases:
  - `ParseRawConfig(...)` builds a normalized `config.Config` by parsing proxies, providers, rules, listeners, DNS, TUN, sniffer, TLS, etc.
  - `executor.ApplyConfig(...)` mutates live runtime subsystems from that parsed config.
- Important consequence: when changing config behavior, check both parsing and application paths. Many fields are valid only if both halves are updated.

### Control plane modules

- `hub/` is the high-level control plane.
- `hub.Parse(...)` parses config and then applies it.
- `hub/executor/executor.go` is the main runtime orchestrator. It fans config out into subsystems in a fixed order:
  - certificates / trust store
  - experimental flags
  - auth users
  - proxies and providers
  - rules and rule providers
  - sniffer
  - hosts
  - general runtime settings
  - NTP
  - DNS
  - listeners
  - TUN
  - iptables
  - tunnels
  - profile persistence / updater
- `hub/route/` serves the external REST API, WebSocket streams (`/logs`, `/traffic`, `/memory`), config mutation endpoints, and external UI hosting.

### Data plane: inbound -> metadata -> rule match -> outbound

- `listener/` owns inbound servers and listener lifecycle.
  - Classic built-in ports (HTTP, SOCKS, mixed, redir, tproxy, Shadowsocks, VMess, TUIC, TUN) are recreated from `executor.updateListeners(...)` / `listener.ReCreate*`.
  - Declarative custom listeners from config are parsed by `listener.ParseListener(...)` and hot-swapped by `listener.PatchInboundListeners(...)`.
- Inbound handlers pass connections/packets into `tunnel.Tunnel`.
- `tunnel/tunnel.go` is the core forwarding engine:
  - normalizes metadata
  - restores hostnames from fake-IP / host mapping
  - optionally sniffs protocols / domains
  - optionally resolves process info
  - matches rules against current runtime state
  - dials the selected outbound proxy
  - wraps connections with traffic statistics
- For TCP, `handleTCPConn(...)` handles sniffing, rule resolution, proxy dialing, and bidirectional relay.
- For UDP, the tunnel uses sharded worker queues plus a NAT table for session mapping and reply routing.

### Proxies, groups, and providers

- `adapter/parser.go` maps each `proxies:` item to a concrete outbound adapter in `adapter/outbound/`.
- `adapter/outbound/` contains protocol implementations such as Shadowsocks, VMess, VLESS, Trojan, Hysteria, TUIC, WireGuard, SSH, AnyTLS, Masque, TrustTunnel, Reality, etc.
- `adapter/outboundgroup/` implements higher-level selection behavior such as `select`, `fallback`, `load-balance`, and `url-test`.
- `adapter/provider/` and `constant/provider` abstractions back dynamic proxy providers and health checks.
- Parsing logic auto-injects built-ins like `DIRECT`, `REJECT`, `PASS`, `COMPATIBLE`, and a synthetic `GLOBAL` selector if the config does not define one.

### Rules and rule providers

- `rules/` and `rules/provider/` implement rule parsing and external rule-set loading.
- Top-level rules and `sub-rules` are both parsed during config load.
- `sub-rules` are validated for circular references before runtime.
- At runtime, `tunnel.match(...)` iterates the active rule list and selects the first matching adapter that is present and protocol-compatible.
- DNS policies reuse the rule-provider ecosystem: `dns.nameserver-policy`, fake-IP filters, and some domain/IP matching paths can reference `rule-set:` sources.

### DNS stack

- `dns/` is a first-class subsystem, not a thin wrapper around `net.Resolver`.
- `executor.updateDNS(...)` rebuilds the resolver, host mapper, fake-IP enhancer, and optional DNS server together.
- `config.parseDNS(...)` is dense and validation-heavy; most DNS feature work will need edits there first.
- The tunnel depends on DNS state heavily for:
  - fake-IP reverse mapping
  - host overrides
  - proxy/direct resolver split
  - rule matching that requires domain/IP resolution
- `main.go` intentionally overrides `net.DefaultResolver` so accidental use of the standard resolver crashes fast.

### TUN / transparent proxying

- TUN behavior is configured in `config.parseTun(...)` and applied by `listener.ReCreateTun(...)` via `listener/sing_tun`.
- Transparent proxy support is split across:
  - `listener/redir`
  - `listener/tproxy`
  - `listener/sing_tun`
  - Linux iptables automation in `executor.updateIPTables(...)`
- `iptables.enable` is Linux-only and is intentionally incompatible with `tun.enable`; `executor.updateIPTables(...)` exits with an error if both are enabled.

### External controller / API

- `hub/route/server.go` builds the controller router.
- The controller can listen on:
  - TCP
  - TLS TCP
  - Unix domain socket
  - Windows named pipe
- Authentication uses a bearer token, with WebSocket token fallback via query string.
- The external UI path is served under `/ui`; config parsing also validates that configured UI paths stay within safe local paths.

### Testing structure

- Most package-level tests live alongside implementation files throughout the repo.
- `test/` is a separate Go module for protocol integration tests and Docker-based end-to-end checks.
- If you touch protocol behavior, listeners, or transport compatibility, check whether there is both:
  - a colocated package test
  - a `test/` integration scenario

## Repository-specific notes

- The repo root often contains built artifacts like `mihomo.exe` or temporary debug binaries on Windows. Do not assume every root-level executable is source-controlled or safe to delete.
- `docs/config.yaml` is the canonical config example referenced by README.
- The dashboard itself is not in this repository; README points to `metacubexd` for the web UI.
