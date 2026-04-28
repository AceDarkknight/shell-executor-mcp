# 计划：统一迁移到 MCP Streamable HTTP 并同步文档

**日期:** 2026-04-27
**状态:** 草稿

## 1. 背景与目标

当前仓库已经在服务端使用了 `mcp.NewStreamableHTTPHandler`，但项目内对外暴露方式、客户端接入示例、测试样例和文档表述仍然混用了以下几种模型：

- 将 MCP 对外端点描述为 **HTTP SSE**；
- 在部分文档和测试中使用 **`/sse`**；
- 在部分 README 和配置示例中使用 **裸 `host:port` 根地址**；
- 在客户端代码中使用 `StreamableClientTransport`，但对“传完整 endpoint URL 还是传 base URL”的约定并不一致；
- `WithHeader` / `WithHeaders` 已暴露给调用方，但当前实现中未见将 `c.headers` 明确注入 MCP HTTP 请求链路。

这导致仓库对“官方推荐用法”的表达不一致，也已经在外部接入时造成了传输层错误与理解偏差。

本次重构的目标是：

1. **统一使用 MCP Streamable HTTP 作为唯一官方推荐传输方式**；
2. **在现有端口上收敛为单一、明确、可文档化的 MCP endpoint**，采用官方常见命名 **`/mcp`**；
3. **彻底移除 `/sse` 和根路径上的旧接入约定**，只保留新的官方推荐 endpoint；
4. **统一客户端配置约定**：客户端必须配置**完整 endpoint URL**（如 `http://host:port/mcp`），而不是裸 `host:port`；
5. **同步更新代码、配置、测试、脚本和文档**，消除仓库内关于 SSE / Streamable HTTP 的冲突描述；
6. **保留现有 `/internal/*` 集群内部 JSON API**，继续与 MCP endpoint 共用同一端口，但职责边界更清晰；
7. **补齐鉴权头与迁移验证链路**，避免 endpoint 重构掩盖已有 header 注入和超时问题。

## 2. 当前问题与现状分析

### 2.1 代码与文档的关键不一致

已确认的现状如下：

- 服务端在 `cmd/server/cmd/run.go` 中通过 `mcp.NewStreamableHTTPHandler(...)` 创建 MCP Handler，并挂载在根路径 **`/`**；
- `docs/api.md` 和 `pkg/mcpclient/client_test.go` 中存在多处 **`/sse`** 示例；
- `README.md`、`cmd/client/README.md`、`pkg/configs/README.md` 等示例中又多为不带路径的裸地址；
- `pkg/mcpclient/client.go` 中 `StreamableClientTransport.Endpoint = serverURL`，意味着调用方传什么 URL，客户端就连接什么 URL；
- `cmd/client/cmd/run.go` 会通过 `WithHeader("X-Cluster-Token", cfg.Token)` 传入 token，但当前 `pkg/mcpclient/client.go` 中未见将 `c.headers` 注入到 transport 或底层请求的明确逻辑；
- `internal/dispatch/dispatcher.go` 中本地执行和 peer HTTP 调用都使用 **5 秒超时**，后续迁移验证需要把它从协议问题中区分出来。

### 2.2 当前风险

如果在没有统一规划的情况下直接改动路由或文档，会引入以下风险：

1. **根路径隐式依赖被破坏**：当前部分调用方可能依赖 `http://host:port` 即可访问 MCP；
2. **一次性切换破坏旧调用方**：如果仍有客户端依赖 `/sse` 或根路径，切换后会直接失败；
3. **鉴权问题被掩盖**：即使 endpoint 改对，如果 header 没真正带出去，外部仍会失败；
4. **超时问题被误判为协议问题**：迁移后若命令执行较慢，5 秒超时可能继续制造“传输失败”假象；
5. **文档与配置继续分裂**：代码升级后如果脚本、README、测试未同步，外部仍会被旧用法误导。

## 3. 目标架构与迁移原则

### 3.1 对外协议与端点约定

按官方推荐，项目对外统一采用 **MCP Streamable HTTP**，并约定：

- **Canonical MCP endpoint**：`/mcp`
- **完整示例**：`http://host:port/mcp`
- **HTTP 语义**：
  - `POST /mcp`：发送 MCP JSON-RPC 请求
  - `GET /mcp`：建立 SSE 流（如果服务器支持服务器主动消息）
- **不再将 `/sse` 作为主入口**；
- **不再将裸 `host:port` 文档化为推荐接入方式**。

### 3.2 迁移策略

本次采用**一次性全量迁移**，不保留兼容层：

- 服务端 MCP endpoint 直接切换为 **`/mcp`**；
- 删除 `/sse` 作为 MCP 外部入口的语义和实现；
- 删除根路径 `/` 作为 MCP 默认入口的语义和实现；
- 仓库内所有客户端、测试、脚本、示例配置、README、API 文档一次性切换到 `.../mcp`；
- 所有旧写法（`/sse`、裸 `host:port`）统一视为错误配置，而不是保留支持。

这意味着本次改造不是“平滑过渡”，而是“仓库内与项目外部集成方式的统一重置”。实施时必须以“全量替换 + 全量验证”为目标，而不是“兼容旧路径”。

### 3.3 端口策略

本次不改变现有单端口部署模型：

- **同一端口同时承载：**
  - `MCP Streamable HTTP`：`/mcp`
  - 内部集群 API：`/internal/*`
- 不新增专用 MCP 端口；
- 不修改内部 `/internal/*` 路由命名；
- 后续若有独立网关或反向代理需求，再通过部署文档说明，而不是本次重构中引入新的端口维度。

### 3.4 客户端配置规范

统一规范如下：

- `server_url` / `servers[].url` 表示 **完整 MCP endpoint URL**；
- 示例必须写成 `http://host:port/mcp`；
- 不允许再在文档、测试或脚本中使用 `http://host:port/sse`；
- 不允许再依赖“裸地址默认映射到 MCP 根路径”的隐式行为；
- 所有 SDK 与 CLI 文档都要明确：这是 **endpoint URL**，不是 **base origin**。

## 4. 实施方案

### 4.1 服务端路由收敛

目标：对外形成清晰、稳定、唯一的 MCP 入口。

计划修改点：

1. 在 `cmd/server/cmd/run.go` 中，将 `mcp.NewStreamableHTTPHandler` 的主要挂载路径调整为 **`/mcp`**；
2. 保持 `/internal/exec`、`/internal/join`、`/internal/sync` 不变；
3. 删除 `/sse` 相关对外入口与语义；
4. 删除根路径 `/` 作为 MCP 入口的旧约定；
5. 服务端日志与启动日志中输出 canonical endpoint，例如：`Listening MCP endpoint at http://host:port/mcp`。

注意事项：

- `/internal/*` 必须继续保留并保持行为不变；
- 删除根路径 MCP 入口后，要确保其他 HTTP 路由的 404/错误提示行为清晰；
- 启动日志和帮助文档必须明确客户端应访问 `.../mcp`，不能再让调用方猜测路径。

### 4.2 客户端 transport 与配置统一

目标：保证所有客户端调用都显式指向 `.../mcp`，并确保 header 真正进入请求链路。

计划修改点：

1. 检查并调整 `pkg/mcpclient/client.go`、`pkg/mcpclient/options.go`：
   - 明确 `serverURL` 表示完整 endpoint URL；
   - `StreamableClientTransport.Endpoint` 必须使用完整 `.../mcp`；
2. 梳理 `pkg/configs/client_config.go` 与各类 config 示例：
   - 统一配置注释与文档描述；
   - 样例全部切到 `.../mcp`；
3. 修复 `WithHeader` / `WithHeaders` 到底层 HTTP 请求的注入链路：
   - 若 SDK transport 支持 header 配置，则在 transport 层注入；
   - 若不支持，则通过包装 `http.Client.Transport` 或等效方式注入；
4. 检查 `cmd/client/cmd/run.go`、测试 client 和外部集成示例，确保 token 传递逻辑与新配置规范一致；
5. 补充文档：
   - “配置的是 endpoint URL”
   - “header 会用于所有 MCP 请求”
   - “不要手动拼 `/sse`”。

### 4.3 全量替换范围控制

由于本次不保留兼容层，计划中必须显式维护“全量替换范围”，确保仓库内所有引用点都同步切换：

| 范围 | 旧写法 | 新写法 |
|---|---|---|
| 服务端 MCP 路由 | `/` 或 `/sse` | `/mcp` |
| 客户端配置 | `http://host:port` 或 `http://host:port/sse` | `http://host:port/mcp` |
| README / API 文档 | HTTP SSE 主入口 | Streamable HTTP `/mcp` |
| SDK / CLI 示例 | 裸地址或 `/sse` | 完整 endpoint URL |
| 测试 / 脚本 | 裸地址或 `/sse` | `.../mcp` |
| 内部节点通信 | `/internal/*` | `/internal/*`（不变） |

全量替换范围必须同时体现在：

- 代码修改清单；
- 测试更新清单；
- 文档修订清单；
- 发布说明 / migration guide；
- 回归验证清单。

### 4.4 文档与示例同步更新

目标：仓库中的所有入口文档只讲一种推荐用法。

需同步更新的主要文档/示例范围：

#### 核心文档
- `README.md`
- `docs/api.md`
- `docs/architecture.md`
- `docs/requirements.md`
- `cmd/server/README.md`
- `cmd/client/README.md`
- `pkg/mcpclient/README.md`
- `pkg/configs/README.md`
- `internal/dispatch/README.md`（如涉及端点描述）

#### 计划/历史文档
- `docs/plan/2026-01-19-implementation-plan.md`
- `docs/plan/2026-01-27-client-refactor.md`
- `docs/plan/2026-03-04-fix-client-connection-timeout.md`

说明：历史计划文档不一定要逐字重写，但至少需要避免继续误导读者。可以采用以下策略之一：

1. 在文档顶部增加“后续已由 Streamable HTTP 计划更新”的说明；
2. 对其中明显错误的 `/sse` / SSE 主叙述做最小修正；
3. 在 README/API 文档中提供统一的迁移说明，减少历史文档误导。

#### 配置与脚本示例
- `test/client_config.json`
- `test/server_config.json`
- `test/test_server_config.json`
- `scripts/test_single_node.sh`
- `scripts/test_cluster.sh`
- `bin/server-template.json`
- 其他引用 endpoint URL 的测试或示例文件

#### 测试代码
- `pkg/mcpclient/client_test.go`
- `test/mcpclient/main.go`
- 任何包含 `/sse` 或裸 endpoint 假设的测试样例

### 4.5 超时与错误语义梳理

本次重构不以“改超时策略”为主目标，但需要把超时从协议迁移中隔离出来，避免回归结论失真。

计划中至少要做以下工作：

1. 记录当前超时基线：
   - `pkg/mcpclient` 默认 HTTP client timeout 为 30s；
   - `internal/dispatch/dispatcher.go` 对 peer HTTP 和本地执行使用 5s；
2. 在验证计划中区分：
   - **transport endpoint 错误**
   - **header/auth 错误**
   - **命令执行超时**
3. 对长命令用例（如 `journalctl`、`top`）增加迁移后验证，确认失败原因不会再被错误归因到 `/sse` / `/mcp`。

### 4.6 迁移文档与发布沟通

除常规 README/API 更新外，建议新增或扩展迁移说明内容，至少覆盖：

1. 从 `/sse` 或裸地址迁移到 `/mcp` 的方式；
2. 对外部集成方的升级提示；
3. header / token 传递方式说明；
4. 常见错误排查（404、transport rejected、timeout、token 未生效）。

## 5. 分阶段修改步骤

### 阶段 1：盘点与定标

1. 确认 canonical endpoint 为 `/mcp`；
2. 明确本次为一次性迁移，不保留 `/sse` 或根路径兼容；
3. 明确配置项语义：`server_url` / `url` 为完整 endpoint URL；
4. 建立受影响文件清单；
5. 确定历史文档最小修正范围。

### 阶段 2：服务端与客户端主链路改造

1. 服务端路由挂载到 `/mcp`；
2. 移除 `/sse` 与根路径上的旧 MCP 接入方式；
3. 客户端与 SDK 统一使用 `.../mcp`；
4. 修复 header 注入链路，确保 token 能进入 MCP 请求；
5. 更新启动日志、错误信息和帮助文本。

### 阶段 3：配置、测试与脚本更新

1. 更新 JSON config 示例；
2. 更新 shell 脚本中的 endpoint；
3. 更新单元测试和集成测试；
4. 补充全量替换后的路由与配置测试；
5. 增加 header 注入验证测试。

### 阶段 4：文档统一

1. README/API/Architecture 全量修正；
2. Client/Server README 同步；
3. `pkg/mcpclient` / `pkg/configs` README 同步；
4. 增加 migration guide / 发布说明；
5. 对历史计划文档做最小范围的“已过时说明”或纠偏。

### 阶段 5：验证与退役准备

1. 执行完整验证清单；
2. 确认内外部调用方均可工作；
3. 统计是否仍有 `/sse` 依赖；
4. 满足条件后进入 `/sse` 退役准备阶段。

## 6. 验证计划

### 6.1 构建与静态检查

- `go build ./...` 成功；
- 修改文件相关测试全部通过；
- 关键模块无新增诊断错误。

### 6.2 路由与协议验证

至少验证以下场景：

1. `POST /mcp` 可以正常完成 `initialize`、`tools/list`、`tools/call`；
2. `GET /mcp` 行为符合 SDK / 服务端预期；
3. `/internal/*` 路由不受 `/mcp` 调整影响；
4. `/sse` 与根路径不再作为 MCP 入口时，错误行为符合预期且可诊断。

### 6.3 客户端与配置验证

1. `cmd/client` 使用 `http://host:port/mcp` 能正常连接；
2. `pkg/mcpclient` 使用完整 endpoint URL 能正常连接；
3. 外部调用示例使用完整 endpoint URL 能正常工作；
4. 传入 token/header 后，服务端能够实际收到并处理该请求头。

### 6.4 回归验证

1. 单节点执行命令通过；
2. 集群模式执行命令通过；
3. 安全拦截逻辑不受影响；
4. 长命令或慢命令的失败原因可区分为 timeout，而不是 transport mismatch；
5. 心跳、重连逻辑在新 endpoint 下仍工作正常。

### 6.5 文档验证

1. 仓库中不再出现把 `/sse` 或根路径写成 Streamable HTTP 主入口的文档；
2. 所有推荐示例统一使用 `.../mcp`；
3. 文档中对 endpoint URL / base URL 的区别描述一致；
4. 迁移说明可独立指导外部接入方完成升级。

## 7. 风险评估与应对

### 风险 1：历史客户端依赖 `/sse` 或根路径

**影响**：切换后旧客户端会立即失效。  
**应对**：在实施前先完成仓库内与受控外部调用方的引用盘点；通过发布说明和升级文档明确“本次无兼容层，必须同步切换到 `/mcp`”。

### 风险 2：header 注入未生效

**影响**：endpoint 已迁移，但鉴权调用仍然失败。  
**应对**：将“header 注入验证”设为阻塞项；无验证不得宣称迁移完成。

### 风险 3：超时问题被误判

**影响**：调用方误以为 `/mcp` 仍然有 transport 问题。  
**应对**：在测试和排查文档中明确区分 transport failure 与 execution timeout。

### 风险 4：文档更新不彻底

**影响**：外部用户继续按旧文档接入。  
**应对**：将 README / API / SDK README / client config 示例纳入强制检查项；历史计划文档至少加过时说明。

### 风险 5：一次性切换范围遗漏

**影响**：仓库内仍残留旧路径引用，导致上线后局部功能失效。  
**应对**：将 grep/脚本/测试/文档扫描纳入实施前后检查项，确保 `/sse` 与根路径 MCP 用法被彻底清理。

## 8. 验收标准

满足以下全部条件，才可认定本次重构完成：

1. 服务端对外 canonical MCP endpoint 为 **`/mcp`**；
2. 仓库中的所有推荐用法、示例配置、测试样例均已统一为 **完整 endpoint URL**；
3. `/sse` 与根路径不再作为 MCP 文档入口或推荐配置；
4. 客户端 header / token 注入链路得到实际验证；
5. `/internal/*` 内部 API 与集群行为不受影响；
6. 文档中不再混用“HTTP SSE 主协议”和“Streamable HTTP 主协议”的表述；
7. 回归测试能够区分协议问题、鉴权问题和执行超时问题；
8. 发布/迁移说明已经明确本次为一次性切换，外部集成必须升级到 `/mcp`。

## 9. 预期效果

完成后，项目将获得以下收益：

- 对外接入方式与 MCP 官方推荐保持一致；
- 客户端与服务端的 endpoint 约定更清晰，减少 transport 层误接；
- 文档、测试、脚本和代码的叙述统一，降低集成成本；
- 仓库内与外部集成方式统一为单一路径，减少后续维护歧义；
- 为进一步完善认证、可观测性和超时治理打下更清晰的协议边界。
