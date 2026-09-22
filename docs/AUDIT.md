# EyvesCloud 源码审计报告

> 审计对象：仓库 `FenhaoLost/VMCLOUD`（产品名 EyvesCloud，二进制 `eyvescloud`）
> 审计范围：Go 后端（`backend/`）、React 前端（`frontend/src`）、Mofang 集成模块（`Mofang/`）、部署脚本（`install.sh`）与文档。
> 性质：静态审计（结论均有源码依据，未执行/未编译）。严重程度：高 / 中 / 低。

## 修复状态快照

| 编号 | 问题 | 状态 |
| --- | --- | --- |
| H1 | `/api/v1/api-keys*` 缺管理员门禁（API Key 自我提权） | ✅ 已修（`server.go` 补 `AdminMiddleware`） |
| H2 | `/api/v1/tasks/` 缺管理员门禁 | ✅ 已修（`server.go` 补 `AdminMiddleware`） |
| H3 | 空绑定 API Key 越权操作全部容器 | ⚪ 判定为**设计行为**：容器绑定是“附加限制”，无绑定的 Key 由 scope 管控；改为“空=无权”会破坏魔方/自动化按名称管理容器的场景。缓解手段应为 scope 最小化 + 按需绑定（未改代码） |
| H4 | `install.sh` 默认从上游拉发行版 | ✅ 已修（默认仓库改为 `FenhaoLost/VMCLOUD`） |
| H5 | Mofang `webssh.php` 任意 WebSocket/SSRF | ✅ 已修：目标白名单（当前域名 + `EYVESCLOUD_WS_ALLOW`）+ 拒绝私网/回环/保留地址，且仅 ws/wss |
| H6 | Mofang `clicd.php` 关闭 TLS 校验 | ✅ 已修：默认开启 `CURLOPT_SSL_VERIFYPEER/HOST`，仅 `insecure=1` 显式跳过 |
| M1 | 子用户/容器口令明文落库并回显 | ⭕ 未修：涉及哈希/一次性返回的大改造，风险高，建议单独做 |
| M2 | 空 scope Key 默认升级为 `["*"]` | ✅ 已修（空 scope 保持为空、默认无权限） |
| M3 | `sh -c` / `bash -c` 拼接执行（注入形状） | ✅ 已修：`getContainerVethByNS` 纯数字校验；防火墙清理改为 argv 执行去 shell |
| M4 | `SecurityAutoShutdown/ARPProtection` 无锁写 | ✅ 已修（`HandleSecuritySettings` 读写均加锁） |
| M5 | 子用户审计日志 `Contains` 匹配 | ✅ 已修（`subuser.go` 审计/登录日志均改为精确 `==` 匹配；回归测试 `TestFilterSubUserAuditLogsExactMatch`） |
| M6 | JWT 无 issuer/audience | ✅ 已修（签发与校验均加 `iss`/`aud`） |
| M7 | 请求体无大小上限 | ✅ 已修（入口统一 `MaxBytesReader` 64MiB） |
| N1 | 母-子：子节点默认地址取母地址导致自代理断链 | ✅ 已修（agent 建连主控自检源 IP 地址；安装脚本默认地址置空） |
| N2 | 母-子：无离线检测 | ✅ 已修（节点读取前 60s 心跳超时置 offline） |
| N3 | 母-子：列表与操作数据源不一致 | ⚪ 已核实列表为 config 派生（`lxc.ListContainers`），与操作同源，常见路径一致；未强行改 |
| N4 | 母-子：母仅能开关机/重启子的容器 | ✅ 部分完善：agent 增加 `reset-password`，母通过代理转发请求体即可重置子容器密码（`agent_api.go` / `nodes.go`）；重装/创建等异步重型操作另文扩展 |
| K1 | API Key 鉴权存在性 DoS（O(N) 次 argon2） | ✅ 已修（新增 SHA-256 指纹 `KeyFingerprint`，`matchApiKey` 先指纹预筛再 argon2，不匹配请求降 O(1)；SQLite 迁移 `key_fingerprint` 列） |
| F5 | 前端 API Key 创建/删除失败无反馈 | ✅ 已修（`ApiIntegration.tsx` 增加 `formError` 展示与删除失败 alert） |
| F6 | 前端租户提交无 try/catch | ✅ 已修（`Tenants.tsx` submit 包裹 try/catch，失败 `setError`） |
| F7 | 审计日志分页越界、WebSSH 闭包陈旧 | ✅ 已修（`AuditLogs.tsx` 钳制当前页；`WebSSHViewer.tsx` 改函数式状态更新） |
| F8 | 裸 URL 下载辅助函数（鉴权旁路隐患） | ✅ 已修（删除 `getAuditLogExportUrl`/`getBackupDownloadUrl` 死函数，统一走鉴权下载） |
| F9 | 租户 `HandleTenantItem` 仅 strip `/api/tenants/` 前缀，`/api/v1/tenants/{id}` 的单租户 PUT/DELETE 无法解析 id 而失效 | ✅ 已修（双前缀 strip `/api/v1/tenants/` 与 `/api/tenants/`） |

> 验证：`go build ./...` ✅、`go vet ./...` ✅、`go test ./internal/{api,config,cli,server}` ✅、`php -l eyvescloud.php` ✅。

---

## 二轮复核（运行时 + 浏览器逐页实操，2026-09-22）

> 本轮在干净环境实际构建并部署（Go 后端 + 重建前端 `frontend/dist` + 浏览器登录面板逐页点击、抓控制台与网络失败），对全部功能做端到端核验。

### 复核结论
- 编译/静态：`go build ./...`、`go vet ./...`、全量 `go test ./...`、前端 `tsc --noEmit` 与 `npm run build` 全部通过；`CGO_ENABLED=0` 发布构建正常（SQLite 用纯 Go 的 `modernc.org/sqlite`，无需 CGO）。
- 运行时 API：登录、仪表盘、容器、模板/镜像、快照、路由、存储、IPv6、任务、子用户、审计日志、安全、通知、API Key、策略、节点、租户、设置/两步验证/登录日志全部 200 且 `success=true`。
- 企业化模块实操成功：配置备份「创建/下载/还原」、审计日志导出、API 限流开关、新建租户、新建节点、新建 API Key、新建策略。
- 已知异常均已释疑：`/api/security/check` 405 是需要 POST、`/api/security/logs` 400 是需要 `container` 参数，均为正确行为，前端调用方式一致。

### 本轮新发现并修复

| 编号 | 问题 | 状态 |
| --- | --- | --- |
| F1 | 母-子：母面板被控节点详情仅暴露「开机/关机/重启」，**未暴露「重置 SSH 密码」按钮**。后端 `nodes.go` 已能代理转发 `reset-password`、`agent_api.go` 子端也已实现并回传新密码，但母 UI 没有入口，导致用户无法从母面板重置子容器密码（此前 N4 只补了后端、漏了前端）。 | ✅ 已修（`frontend/src/pages/NodeManagement.tsx`：容器行新增「重置 SSH 密码」按钮，调用 `nodeContainerAction(...,'reset-password')`，回显新密码并提示仅显示一次；前端已重建进 `frontend/dist`） |

### 待处理（承接上文快照的进行项）
- M1：子用户/容器口令明文落库并回显（大改造，另立项）。
- 2FA 端点可被 `*` scope 的 API Key 触发（认证模型调整，另立项）。
- 节点 token 明文 HTTP 传输（加密传输，另立项）。

---

## 三轮复核（源码 + 部署链路验证，2026-09-22）

> 本轮处理低危项、复核前端与后端调用一致性、并对发布/部署链路做了实际构建与运行验证。

### 处理低危项
- M5（子用户审计日志 `Contains` 匹配）：核实现行源码已用 `==` 精确匹配（`filterSubUserAuditLogs` / `filterSubUserLoginLogs`），且子用户本就不具备 `audit:read`/`loginlog:read` scope（`subUserScopeAllowed`），无可泄漏路径。**补充回归测试**锁定精确匹配、防止回退为 Contains（`internal/api/subuser_test.go`），已通过（含 `-race`）。

### 源码复核新发现并修复

| 编号 | 问题 | 状态 |
| --- | --- | --- |
| D1 | 部署硬伤：`install.sh` 与 `DEPLOYMENT.md` 的手动 systemd 单元包含 `NoNewPrivileges`/`ProtectControlGroups`/`ProtectKernelTunables`/`ProtectKernelModules`/`RestrictSUIDSGID`。eyvescloud 通过子进程管理 LXC/KVM，需可写 cgroup、`/proc/sys`、可加载内核模块及 setuid 辅助程序，这些加固会导致**'系统卡部署后容器/虚拟机无法启动'**。 | ✅ 已修（移除冲突项，保留 `PrivateTmp`+`RestrictRealtime`；`install.sh` 与 `DEPLOYMENT.md` 同步更新并加说明） |
| D2 | 数据竞争：`updateExpiry`/`updateResourceLimit`（handlers.go）、`updateSnapshotQuota`（snapshots.go）、LXC 快照调度器（snapshot.go:241）在未持有 `AppConfigMu` 写锁时直接改写 `FindContainer` 返回的共享容器指针。 | ✅ 已修（改用 `config.MutateContainerNoSave`/`MutateContainerByID`，锁内原子更新 + 统一 `SaveConfig`） |
| D3 | 数据竞争：`config.AddSnapshot/FindSnapshot/RemoveSnapshot/ContainerSnapshots` 及 `api.ensureImageEnabled/removeImageEnabled/getEnabledImageSet` 无锁读写共享配置切片。 | ✅ 已修（快照函数内部加 `AppConfigMu`；镜像函数改用 `config.MutateGlobal`/RLock；已核验各调用点不持有 `AppConfigMu`，无死锁） |

### 部署链路验证（实跑通过）
- `bash build.sh` 全流程成功：前端 `npm run build` → Go 发布构建（`CGO_ENABLED=0`）→ 打包 `dist/eyvescloud-linux-amd64(.tar.gz)`（含二进制 + install.sh）。
- 用 `build.sh` 产物直接运行：可正常首启（生成随机管理员 + SQLite 配置持久化）、嵌入式 Web 面板可访问、`/api/health` 正常。
- `go build ./...`、`go vet ./...`、全量 `go test ./...`、受影响包 `go test -race` 全部通过 → **无回归、无数据竞争、无死锁**。

### 待处理（仅剩需另立项的三项）
- M1：子用户/容器口令明文落库并回显（大改造）。
- 2FA 端点可被 `*` scope 的 API Key 触发（认证模型调整）。
- 节点 token 明文 HTTP 传输（加密传输）。

---

## 四轮复核（Mofang / API Key / 策略引擎，2026-09-22）

> 对三个此前未深评的高风险独立面做源码复核，均未发现可确证的新缺陷。

- **Mofang PHP 模块**
  - `webssh.php`：WebSocket 目标仅放行「当前请求主机 + `EYVESCLOUD_WS_ALLOW`」白名单，公网/私网 IP 均被拒（无 SSRF）；协议串经白名单正则 `websocketProtocolValue` 校验且全部 `json_encode` 输出进 JS，无 XSS/注入。
  - `eyvescloud.php`：魔方计费对接客户端，默认启用 `CURLOPT_SSL_VERIFYPEER/HOST`（H6 已修），以管理员配置的 API Key + 服务器地址对接面板，无 shell 执行面。
- **API Key 鉴权（apikey.go）**：密钥 argon2id 加盐哈希、常量时间比较、legacy 迁移；每次请求实时校验 disabled/过期/IP 白名单/scope；**创建时空 scope 落入「只读默认 scope」，无法借此提权**。注：`validateApiKeyDetails` 对存量空 scope 密钥在认证时升级为 `"*"`（legacy 兼容），仅 M2 文档描述与实际不符，**不可利用**，为避免破坏存量密钥未改动。
- **策略/安全引擎（policy_engine.go）**：快照求值，按规则冷却（`policyTriggerMu`）；动作统一走 `config.MutateContainerByID` 持锁改动并持久化，scope 匹配（all/tenant:/container:）正确，无数据竞争、无 nil 解引用。

**结论**：本轮复核的高风险面均稳健，无新增可确证缺陷。剩余仅三项需另立项的大改造（M1 口令明文、2FA 的 `*` scope 触发、节点 token 明文传输）。

---

## 五轮复核（API Key 鉴权模型深审，2026-09-22）

> 对 API Key 的完整鉴权链做深审，确认无越权，发现 3 个低危/加固类观察项。

### 稳健面（无漏洞）
- **每次请求实时鉴权**：`AuthMiddleware` 在 JWT 失败后回落 `validateApiKeyRequest`；`validateApiKeyDetails` 对每个请求都从**实时配置**校验 `Disabled`/过期/IP 白名单/scope/容器绑定。因此 API Key 的**吊销、scope 收缩、绑定变更均即时生效**（优于子用户 JWT 内固化 claims 需到 token_version 才失效）。
- **密钥存储**：argon2id 加盐、常量时间比较、legacy 迁移路径兜底。
- **不可自提权**：`/api/api-keys*` 与 `/api/v1/api-keys*` 均由 `AdminMiddleware` 门禁（要求 `admin:access`）；子用户 token 与无 `admin:access` 的 Key 均被拒。`updateApiKey`（可改 scope/绑定）仅管理员可达，无法被他钥自改自升。
- **空 scope 创建安全**：创建时空 scope 落入只读 `defaultApiKeyScopes`，不会成为 `"*"`。
- **响应不泄敏感**：`createApiKey`/`listApiKeys` 仅返回 ID/名称/Prefix/scope/绑定等，**不含 KeyHash（salt/argon 摘要）与完整 Key**；完整 Key 仅在创建响应出现一次。
- **SSH/VNC 票据交互**：API Key 需 `terminal:ssh` scope，并经 `isContainerAllowedForRequest` 校验容器绑定；一次性、60s 过期票据与容器名绑定。未绑定容器但持 `terminal:ssh` 的 Key 可访问全部容器——属 H3（scope 式）设计，已记录。
- **小瑕疵**：`Prefix = rawKey[:13]+"..."` 即固定串 `eyvescloud_sk_…`（无熵），对所有 Key 相同、无辨识/预筛价值，仅影响展示与 K1 可行性，无安全风险。

### 观察项（低危 / 加固建议，未改代码）
| 编号 | 问题 | 建议 |
| --- | --- | --- |
| K1 | `matchApiKey` 对每个带 Key 形态头部的请求做 **O(N) 次 argon2**（每次约 64MB、t=3）。多密钥 + 大量伪造 `Bearer eyvescloud_sk_…` 请求会消耗大量 CPU/内存（存在性 DoS）。存量 `Prefix` 字段因所有 Key 共享前缀、无法廉价预筛。 | **已修复**：为每个 Key 新增 SHA-256 指纹 `KeyFingerprint`（`apiKeyFingerprint`），`createApiKey` 落库；`matchApiKey` 先常量时间比对指纹、命中后才跑 argon2，不匹配请求降为 O(1)。SQLite 增加 `key_fingerprint` 列并迁移。新增回归测试 `TestApiKeyFingerprintPreScreensInvalidKeys`。 |
| K2 | `updateApiKeyLastUsedForKey` 在每个已鉴权 API 请求上都 `MutateGlobal` → **全量配置序列化并写 SQLite**（更新 LastUsed/IP），高频 API 调用下写放大明显。 | 节流：仅当上次写入过久（如 >60s）才落库更新。 |
| K3 | ~~M2 文档与实际不符~~（已定性为有利行为，非缺陷）：`validateApiKeyDetails` 对**存量空 scope** 的 legacy 密钥在认证时升级 `"*"`，这是正确的向后兼容（存量全权 Key 保留原有权限）；**新建**空 scope Key 走只读默认 scope，已在 `createApiKey` 固定。 | 无需改动；新增测试 `TestApiKeyContainerBindingEnforced`、`TestApiKeyCreateNormalizesEmptyScopeToReadOnlyFallback` 锁定绑定语义与创建默认。 |

---

## 六轮复核（前端源码深审，2026-09-22）

> 对前端深层逻辑与安全模式做静态深审。结论：**未发现可用 XSS/存储漏洞**（用户数据均以 React 文本节点渲染；无 `dangerouslySetInnerHTML`/`window.open`）；WebSSH/VNC 票据通过 WebSocket 子协议头发送而非 URL（正确）；审计导出/备份下载已统一走 `Settings.tsx` 的 `downloadWithAuth`（`Bearer` 鉴权）。发现以下可修复项，均已处理：

### 已修复
| 编号 | 问题 | 修复 |
| --- | --- | --- |
| F5 | `ApiIntegration` 创建/删除 API Key 失败被静默吞掉（`catch { // ignore }`），用户无任何反馈，表单保持打开。 | 新增 `formError` 状态在表单内红条展示保存失败；删除失败用 `alert` 反馈；新增 `errorMessage` 助手从 axios 错误提取 `response.data.message`。 |
| F6 | `Tenants` 的 `submit` 无 try/catch，重名/非法 ID 提交直接产生未处理 rejection 且无反馈。 | `submit` 包裹 try/catch，失败时 `setError(...)`（含后端 message），成功才 `cancel()+fetchData()`。 |
| F7 | `AuditLogs` 分页：刷新后日志变少使 `page > totalPages`，`slice` 返回空表且页码显示失配；`WebSSHViewer` `onclose` 捕获首渲染的陈旧 `status`（stale closure）。 | `AuditLogs` 用 `safePage=min(page,totalPages)` 渲染并新增 clamp effect 收紧 `page` 状态；`WebSSHViewer` 的 `onclose` 改为纯函数式 `setStatus(current=>…)`，去掉对闭包 `status` 的引用。 |
| F8 | `api.ts` 的 `getAuditLogExportUrl`/`getBackupDownloadUrl` 返回**裸 URL**（不携带 `Authorization`），一旦被 `<a href>`/`window.location` 引用会绕过 Bearer 鉴权（浏览器 `<a>` 不会附 localStorage token）。 | 两函数为死代码，已删除；下载统一走 `Settings.tsx` `downloadWithAuth`。 |

### 已核实为正确 / 无需改动
- **Security 开关单字段提交**（初判可能互相覆盖）：核实后端 `HandleSecuritySettings` 按指针存在性逐字段 merge（`if req.AutoShutdown != nil`），单字段 PUT 不会重置另一项 → **非缺陷**。
- **子用户访问码入分享 URL**（`SubUserManagement`/`ContainerDetail` 拼 `?code=<access_code>`）：`access_code` 是换取登录态的真实凭据，放在 URL query 会经浏览器历史/Referer/代理日志留存。属**产品设计**（“一键分享链接让子用户免密登录”），改为一次性码需登录流程大改造（另立项），当前建议前端在 UI 提示“该链接含凭据，勿经不可信渠道中转”。
- API Key 新密钥仅存于创建时 React state，表内仅展示 `prefix`，未入 localStorage/URL/console。

---

## 目录

- [一、总体结论](#一总体结论)
- [二、高危问题（High）](#二高危问题high)
- [三、中危问题（Medium）](#三中危问题medium)
- [四、低危问题（Low）](#四低危问题low-javascript部分未单列)
- [五、与文档 / 声明不一致的问题](#五与文档--声明不一致的问题)
- [六、建议修复优先级](#六建议修复优先级)

---

## 一、总体结论

工程整体基础较好：SQLite 全部使用参数化查询；镜像下载有 `safehttp` 的 SSRF 防护（回环 / 链路本地 / 保留地址 / 重定向均被拦）；面板访问策略正确区分信任代理与 `X-Forwarded-For`；CORS/安全头完整；防火墙命令大多采用 argv 参数化执行；WebSSH/VNC 采用一次性票据。前端未发现可复现的存储型 XSS 或 `dangerouslySetInnerHTML` 滥用。

**最需要优先修复的三件事：**

1. `/api/v1/*` 版本化路由整体鉴权深度低于 `/api/*`，存在 **API Key 自我提权**路径（H1）。
2. **未绑定容器的 API Key 绑定判断失效 → 可越权操作全部容器**（H3）。
3. **`install.sh` 默认从上游 EYVESCLOUD 下载发行版**，导致仓库文档里的一键安装实际装的是别家二进制（H4）。

Mofang 模块是独立高风险面（webssh.php 无鉴权任意 WebSocket 代理、eyvescloud.php 关闭 TLS 校验）。

---

## 二、高危问题（High）

### H1. `/api/v1/api-keys` 缺少管理员门禁 → API Key 可自我提权为 root

- **位置**：`backend/internal/server/server.go:107-108` vs `167-168`
- **现象**：老版路由 `/api/api-keys` 用 `AdminMiddleware`，而版本化路由 `/api/v1/api-keys` 只用 `AuthMiddleware`。`api/apikey.go:55-88,159` 内部仅检查 scope。
- **风险**：任何持有 `apikey:update` scope 的非管理员 Key，可 PATCH 自己把 `Scopes` 改成 `["*"]`；持有 `apikey:create` 的 Key 可直接创建一个 `["*"]` 的 Key，从而获得完整管理员权限。`hasScope` 对 `*` 一律放行，`AdminMiddleware` 也认可 `admin:access`。
- **建议**：给 `/api/v1/api-keys*` 补回 `AdminMiddleware`（与 `/api/api-keys*` 对齐），并对 `*` scope 的授予做管理员门禁。

### H2. `/api/v1/tasks/` 删除任务缺少管理员门禁

- **位置**：`server.go:84` vs `146`
- **现象**：`/api/tasks/` 用 `AdminMiddleware`，`/api/v1/tasks/` 只用 `AuthMiddleware`。
- **风险**：与 `/api` 鉴权标准不一致，子用户或窄 scope Key 可能调用 `HandleTaskDelete`。
- **建议**：对齐 `/api` 的鉴权链；抽查 `HandleTaskDelete` 内部是否对操作人/越权做 scope 校验。

### H3. 未绑定 `container_uuids` 的 API Key 绑定判断失效 → 越权管控全量容器

- **位置**：`internal/api/subuser.go:359-379`、`internal/api/handlers.go:700-711,591-601`
- **现象**：
  ```go
  if ctx.Type == authTypeAPIKey && len(ctx.ContainerUUIDs) == 0 {
      return subUserAccess{}, false   // restricted=false
  }
  ```
- **风险**：为「无绑定 Key」的语义是"无权"，但此处 `false` 被当作"未限制"，于是 `filterContainersForRequest` / `isContainerAllowedForRequest` 对**所有容器放行**。持有 `container:power` 这类 scope 的空绑定 Key 可以 start/stop/restart/reset-password 全部容器，连列表接口也会泄露所有容器的 IP/租户/公网地址。
- **建议**：把"空绑定"显式解释为"无权（允许 0 个容器）"，或为 API Key 增加"绑定全部"的显式标记，避免空态歧义。

### H4. `install.sh` 默认拉取**上游 EYVESCLOUD** 发行版 → 一键安装装错产品

- **位置**：`install.sh:4`
  ```sh
  REPO="${EYVESCLOUD_REPO:-FenhaoLost/VMCLOUD}"
  ```
- **现象**：原代码 `REPO` 默认值指向上游仓库（`MengMengCode/CLICD`），导致一键安装实际下载别家二进制。**已修复**：默认值改为本仓库 `FenhaoLost/VMCLOUD`（`EYVESCLOUD_REPO` 仍可覆盖）。由于本仓库代码/产物已全量改名为 `eyvescloud`，默认发行版来源即为本仓库。
- **影响**：修复前发布 / 回滚 / 功能皆错位；现默认即本仓库。

### H5. Mofang `webssh.php` 无鉴权 + 任意目标 → 浏览器端 SSRF / Open WebSocket 代理

- **位置**：`Mofang/handlers/webssh.php:2-5,70-72,198-208`
- **现象**：`ws`（WebSocket 目标）与 `ticket`/`protocol`/`container` 全部取自 URL，未校验目标域名 / 内网网段 / 端口范围，前端直接 `new WebSocket(wsUrl, protocolValue)`。
- **风险**：构造 `webssh.php?ws=ws://127.0.0.1:6379/...` 诱导管理员打开，可利用受害浏览器探测内网端口 / 访问内网服务 / 作为任意目标的反向代理（反射型 SSRF）。页面本身无鉴权，任何人可拼装。
- **建议**：服务端白名单校验 `ws`（仅允许面板所在域、拒绝回环/内网/保留地址、限定端口范围），并要求携带会话/票据校验。

### H6. Mofang `eyvescloud.php` 关闭 TLS 证书校验 → API Key 可被中间人窃取

- **位置**：`Mofang/eyvescloud.php:133-134`
  ```php
  CURLOPT_SSL_VERIFYPEER => false,
  CURLOPT_SSL_VERIFYHOST => false,
  ```
- **风险**：与 EYVESCLOUD 后端之间携带 `Authorization: Bearer <apiKey>` 与 `X-API-Key` 的全部请求都关闭证书校验，即使配置 HTTPS 也等于明文裸奔，攻击者可中间人截获 Key 并冒充模块执行开/关/删/重装/改密/防火墙操作（README 又明确建议 HTTPS）。
- **建议**：恢复 `true`，并为内置 CA 配置 `CURLOPT_CAINFO`。

---

## 三、中危问题（Medium）

### M1. 子用户 / 容器口令明文落库并在接口原样回显

- **位置**：`internal/config/store_sqlite.go:1045-1051,870-889,824,838`；`internal/api/subuser.go:916,1066,1096,1127`；API 列表接口下发 `ssh_password`。
- **风险**：`sub_users.password`、`containers.ssh_password` 明文存 SQLite；`HandleSubUserList` 对**所有子用户回显明文密码**。DB 文件一旦被读即批量泄露口令。
- **建议**：口令哈希后落库；密码仅在创建/单用户一次性返回；列表接口不再下发 `ssh_password`。

### M2. 配置缺失 scope 的旧 API Key 默认升级为 `["*"]`

- **位置**：`internal/config/config.go:1201-1208`
  ```go
  if len(AppConfig.ApiKeys[i].Scopes) == 0 { AppConfig.ApiKeys[i].Scopes = []string{"*"} }
  ```
- **风险**：历史无 scope 的 Key 在启动归一化后自动获得最高权限，等于默认 root。
- **建议**：缺失 scope 应保持受限（无权限或显式提示补齐），而非放大为 `*`。

### M3. `sh -c` / `bash -c` 拼接外部输出执行（命令注入形状的反模式）

- **位置**：`internal/lxc/lxc.go:2366-2382`（`getContainerVethByNS`）、`internal/lxc/portmap.go:1205-1209`（防火墙清理 `while read rule; do iptables $rule; done`）。
- **风险**：`pid`/`ifIdx`/`iptables -S` 输出在拼入 shell 前未做强校验与引号处理。当前来源多为数值所以暂不可利用，但一旦上游输出被污染即可在宿主 root 执行任意命令；`$rule` 含空格还会被重新分词误删规则。
- **建议**：改用逐参数 argv（`exec.Command` 传参数数组），或对变量做严格白名单/引号转义（参考同文件 `shellQuote` 用法）。

### M4. `AppConfig` 全局字段多处无锁读写 → 数据竞争

- **位置**：`internal/api/security.go:826-828`（`SecurityAutoShutdown`/`ARPProtectionEnabled` 直接写，未持 `AppConfigMu`，与后台 scanner 并发读）、`internal/config/config.go:2317-2324`、`2261-2285`。
- **风险**：内存数据竞争，极端可导致配置错乱/崩溃（`go test -race` 可发现）。
- **建议**：对 `AppConfig` 及指标存储建立统一加锁入口，禁止散点直写。

### M5. 子用户审计日志过滤用 `Contains` 子串匹配 → 可能越权读他人记录

- **位置**：`internal/api/subuser.go:1134-1149`
  ```go
  if log.User == username || strings.HasPrefix(log.User, "user:") && strings.Contains(log.User, username)
  ```
- **风险**：用户名互为子串（如 `user-abc` 与 `user-abc-de`）时相互读到对方操作记录；伪造 `Actor` 为 `user:<name>` 可影响过滤。
- **建议**：改为精确匹配（`==` 或等价的稳定分隔解析）。

### M6. JWT 校验策略不完整

- **位置**：`internal/api/auth.go:178-236`
- **风险**：未校验 `Issuer`/`Audience`/`nbf`；管理 token 泄露后 24h 内有效，仅改密码全局失效；`tokenFromRequest` 只读 `Authorization` 头但注释声称也读 cookie（文档/实现不一致）。
- **建议**：补全 issuer/audience/nbf；必要时引入短期刷新 token。

### M7. 请求体/响应无大小上限

- **位置**：`handlers.go:256-262`（`io.ReadAll(r.Body)` 无限制）、`nodes.go:460`。
- **风险**：大 JSON 请求体可能造成内存占用/慢速读取 DoS。
- **建议**：`http.MaxBytesReader` 限制请求体，限制/流式返回大列表。

### M8. Mofang 防火墙规则字段无服务端校验，原样透传

- **位置**：`Mofang/eyvescloud.php:1468-1593`（`$rules` 原样转发 `/firewall`）。
- **风险**：端口范围、IP/CIDR、描述均不校验即透传给后端；若后端拼入 iptables/shell 则构成规则注入。
- **建议**：在 PHP 侧做端口/IP/CIDR 白名单校验后再转发。

### M9. Mofang 客户区 func/ID 由请求完全控制 + 敏感数据/debug 回显

- **位置**：`Mofang/eyvescloud.php:616-629,947-980,1612-1716`。
- **风险**：`func`/`id/service_id` 直接取自请求；`eyvescloud_info_ajax` 把 EYVESCLOUD 容器 `usage`、`debug`（含回显的 `$_GET`/payload）直接返回给客户。横向越权与敏感信息泄露是否可利用取决于魔方框架外层是否做会话+归属校验（当前文件未见）。
- **建议**：在函数入口对 `func` 白名单 + 服务归属做强校验；关闭生产环境 debug 回显。

### M10. Mofang 票据经 URL/查询串明文传送 + Host 头注入

- **位置**：`Mofang/eyvescloud.php:361-382,1433-1449`；`webssh.php:5,8,72`。
- **风险**：`eyvescloud_webssh_url()` 用 `$_SERVER['HTTP_HOST']` 生成 handler 地址，可被 Host 头注入指向攻击者域名；票据放 URL 易被日志/Referer 泄露。
- **建议**：Host 头校验/白名单；票据改用短时一次性凭证走 header/cookie。

---

## 四、低危问题（Low）

- **L1** `main.go:117-126`：非 server 模式且无终端时会尝试 `systemctl start eyvescloud`，无 systemd 环境会刷错误日志（不致命）。
- **L2** `server.go:74-75 vs 137`：`/api/v1/host-info`、`/api/v1/dashboard`、`/api/v1/images*` 等鉴权深度低于 `/api/*`（`Auth` vs `Admin`），信息暴露面扩大。随 H1/H2 一并统一基线。
- **L3** `api/ipv6.go:20-77`：`assignIPv6ByRuntime`/`updatePublicIPv4ByRuntime` 未强制地址必须属于配置的公网池，可写入任意地址（仅管理员/operator 可调，风险受限）。
- **L4** `handlers.go:754`：到期容器仅部分操作被拦截（resetSSHPassword 检查 `IsExpired`），start/restart/reinstall/改资源等未统一"到期即禁"，生命周期策略不一致。
- **L5** `apikey.go:300-306`：`legacyHashKey` 用按位 XOR 弱哈希校验旧 Key 前缀，扩容暴力面（有 `needsRehash` 升级，兼容包袱）。
- **L6** `config.go:1930-1940`：`FindContainerByUUID` 在 `RLock` 内返回切片元素指针，锁释放后被写锁整体替换可读取陈旧/悬空数据（部分调用方已自行复制，未全覆盖）。
- **L7** 前端：`AuthContext.tsx` 用 `atob` 裸解 JWT claims 驱动 `isSubUser/isReadOnly`；token 存 `localStorage`（一旦 XSS 即可窃取）。建议以服务端 `/check-auth` 为唯一权威，token 移入 httpOnly cookie / Memory。
- **L8** 前端轮询重建：`Containers.tsx:156-165`、`ImageManagement.tsx:75-79` 的 `setInterval` 依赖数组含每轮变化的对象导致 interval 反复重建、请求放大；建议把依赖收敛为布尔量/ref。
- **L9** 前端契约飘移：`migrate-export`、`nodes/{id}/install-script` 的返回形态与全局 `APIResponse<T>` 约定不一致；`BaseURL '/api'` 与文档的 `/api/v1` 并存易造成 404/格式错误。建议统一。
- **L10** 前端：策略触发记录 `h.value.toFixed(2)` 缺 `||0` 兜底，服务端缺该字段时整页白屏。
- **L11** Mofang：`templates/info.html:164` 客户区明文渲染 SSH 密码；`templates/nat.html:110,144` 首次服务端渲染变量未确认转义（前端 AJAX 已转义）；JWT 经 URL `$_GET.jwt` 传递（Referer/日志泄露）。
- **L12** Mofang `webssh.php:15`：`echo QUERY_STRING` 未转义，但响应为 `Content-Type: text/plain`，现代浏览器按纯文本渲染，反射型 XSS 基本被 neutralize（依 Content-Type 而定，仍建议转义）。

---

## 五、与文档 / 声明不一致的问题

1. **产品命名三套并存**：文档/README 用 `EyvesCloud`，仓库为 `VMCLOUD`，二进制/DB 目录/服务用 `eyvescloud`。文档站 `docs/.vitepress/config.ts` 社交链接曾指向不存在的 `github.com/EyvesCloud/EyvesCloud`（已改为 `FenhaoLost/VMCLOUD`）。请统一对外品牌与仓库地址。
2. **发行版来源错位**：`install.sh` 默认 `FenhaoLost/VMCLOUD`（见 H4），与 README/部署文档宣称"安装本仓库"不一致（文档已加 `EYVESCLOUD_REPO` 说明）。
3. **Go 版本号不一致**：README/shiq 声明 Go 1.24，`backend/go.mod` 为 `go 1.25.0`。
4. **README 内容重复 + 格式损坏**：原 README 功能/技术栈/预览/免责声明/致谢/Star 历史大量整段重复，并在段落间夹带遗留 Markdown 表格碎片与孤立 `<` 字符；已整体重写去重。
5. **Mofang README 声称"Origin 校验由 EYVESCLOUD 后端完成、前端无法伪造"**：但 `webssh.php` 接受任意 `ws` 目标，魔方侧无任何校验（见 H5），该声明与实际不符，且未提及 TLS 关闭（H6）与防火墙规则透传（M8）。

---

## 六、建议修复优先级

| 优先级 | 事项 | 位置 |
| --- | --- | --- |
| P0 | 统一 `/api/v1/*` 与 `/api/*` 鉴权基线（H1/H2）；修复空绑定 API Key 越权（H3） | server.go / subuser.go / handlers.go |
| P0 | `install.sh` 默认仓库改为本仓库（H4） | install.sh |
| P1 | Mofang：webssh 目标白名单 + TLS 校验 + 客户区校验/脱敏（H5/H6/M8/M9/M10） | Mofang/ |
| P1 | API Key 缺失 scope 不再默认 `["*"]`（M2）；明文口令改哈希 + 列表不下发（M1） | config.go / store_sqlite.go |
| P2 | 落地 shell 命令 argv 参数化（M3）；AppConfig 统一加锁（M4）；日志精确匹配（M5）；JWT 补 issuer/audience（M6） | lxc/portmap/security/subuser/auth |

---

## 七、企业化改进路线图

按企业级要求持续推进，各模块以「完成 → 自验证（build/vet/test）→ 补文档」闭环。

| 模块 | 内容 | 状态 |
| --- | --- | --- |
| **A 账号与访问安全** | TOTP 两步验证（RFC 6238 + 一次性备份码）、密码强度策略（≥10 位含字母数字）、登录风控（滑动窗口限流 5 次/10 分钟） | ✅ 完成（后端 `totp.go`/`twofa.go` + 配置持久化；前端设置页 2FA 面板 + 登录两步流程；文档已更新） |
| **B 审计合规** | 审计日志导出（CSV/JSON）、保留期设置（`audit/settings`）+ 后台定时清理、失败操作审计归因 | ✅ 完成（`enterprise.go` + 后台 `StartAuditRetention`；前端「审计合规」面板） |
| **C 容灾恢复** | 配置 JSON 快照的自动/手动备份、下载、还原、保留份数轮转，后台调度器 | ✅ 完成（`enterprise.go` + `StartBackupScheduler`；前端「容灾备份」面板） |
| **D 可观测性** | 公开健康检查 `/api/health` + 管理员运行时详情（Go 运行时/内存/goroutine/容器/节点/任务），前台启动计时 | ✅ 完成（前端「API 限流」面板内展示健康指标） |
| **E API 治理** | `/api/v1/openapi.json` 精简契约、按客户端 IP 对版本化接口限流（可配置每分钟阈值），`rate-limit/settings` | ✅ 完成（`apiRateLimitMiddleware` + `AllowVersionedRequest`；前端「API 限流」面板） |
| **F 多租户** | 租户 CRUD + 容器/vCPU/内存/磁盘配额，容器创建时强配额校验（`tenant` 字段贯通 LXC/KVM），删除需为空租户 | ✅ 完成（`enterprise.go` + `lxc.ContainerConfig.Tenant`；前端「多租户」页 + 侧栏入口） |

---

## 八、复查（第二轮源码审计）与修复记录

对全量源码（后端 + 前端）做第二轮审计，逐项核对后确认并修复了以下问题：

| 编号 | 级别 | 问题 | 处置 |
| --- | --- | --- | --- |
| R1 | 高 | 前端备份下载 / 审计导出用 `<a href>` 直连受保护接口，无法携带 `Authorization` → 浏览器必然 401 不可用 | ✅ 修：改用 `fetch(带 Bearer) → Blob → objectURL` 下载（`Settings.tsx` `downloadWithAuth`） |
| R2 | 高 | `enterprise.go` 配置备份「下载/还原」可指向任意目录/任意文件（任意文件读、还原触发任意删除链） | ✅ 修：备份目录固定为 `DataDir/backups`；下载/还原仅允许 `Backups` 已登记文件名（`config.BackupDirectory` / `IsBackupFileKnown`） |
| R3 | 高 | `checkTenantQuota` 锁外使用共享切片指针 `&Tenants[i]`（数据竞争） | ✅ 修：锁内值拷贝（`enterprise.go`） |
| R4 | 高 | `server.go` 每请求无锁读 `APIRateLimit`、`auth.go` 每请求无锁读 `JWTSecret`（高频竞争） | ✅ 修：新增持锁 getter `config.GetAPIRateLimit` / `GetJWTSecret` 并改用 |
| R5 | 高 | `notify.go` 无锁写 `AppConfig.Notifications`（与后台协程竞争） | ✅ 修：改走 `config.MutateGlobal` |
| R6 | 中 | `enterprise.go` 多处无锁读 `BackupSettings/Backups/AuditLogs`（`HandleBackupSettings`/`StartBackupScheduler`/`pruneBackupFiles`/`HandleAuditSettings`） | ✅ 修：统一走 `GetBackupSettings`/`GetBackupCount`/`GetAuditLogCount` |
| R7 | 中 | 租户配额 `int()/int64()` 对小数 vcpu/disk 截断 → 配额被低估 | ✅ 修：改用 `math.Ceil` |
| R8 | 中 | 版本化限流 map 只在 >10000 时删空桶，活跃 IP 下无界增长 | ✅ 修：>5000 时清理过期/空键 |
| R9 | 中 | 还原旧备份后未重新归一化/迁移，新字段可能缺失 | ✅ 修：还原后调用 `config.ReconcileConfig()` |
| R10 | 低 | 剪贴板 `navigator.clipboard` 在 HTTP 下可能不存在且“假成功” | ✅ 修：`copyText` 带 execCommand 回退、仅成功后提示 |
| R11 | 低 | 登录 2FA 输入 `inputMode="numeric"` 在手机端无法输入十六进制备份码 | ✅ 修：移除 numeric 键盘 |
| R12 | 低(遗留) | 子用户/容器口令明文落库并回显（M1）；节点 token 明文 HTTP 传输；2FA 端点可被 `*` scope 的 API Key 触发 | ⚪ 未改：明文口令大改造风险高，沿用既有 M1 结论单独立项；token/2FA 属既有认证模型设计，需整体演进 |

> 复查验证：`go build` / `go vet` / `go test`（api/config/cli/server/lxc/kvm/safehttp 全绿）、前端 `tsc` + `vite build` 通过。