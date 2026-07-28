# 已确认问题与复现总表

## 文档范围

本文只记录本项目在生产定制、代理转发、模型参数、渠道容量、代码发布和 Supabase 迁移过程中已经确认的问题，以及对应的复现或只读核验方法。

- 代码分支：`repro/touwaeriol-custom-20260720`
- 最终 GPT-5.6 适配提交：`69726bf`
- 生产状态最后在线核验日期：`2026-07-20`
- Supabase 迁移核验日期：`2026-07-19`
- 文档整理日期：`2026-07-28`

生产环境可能已经在 `2026-07-20` 后发生变化，因此所有生产状态都必须重新执行本文的只读命令确认，不能仅凭历史记录判断。

## 状态说明

| 状态 | 含义 |
|---|---|
| 已修复并有回归测试 | 当前分支包含修复，自动化测试能够覆盖对应失败场景 |
| 代码已修复，生产待核验 | 仓库已有修复，但最后一次在线核验时生产尚未确认部署最新版本 |
| 配置或运行环境问题 | 不是单纯修改 Go/Python 源码就能解决，需要检查进程、端口、凭据、余额、数据库或依赖 |
| 历史错误方案 | 曾经出现但已经撤销，复现用于防止再次引入 |

## 一键取得可复现代码

```bash
git clone --branch repro/touwaeriol-custom-20260720 \
  git@github.com:jkjk02/new-api.git
cd new-api
git merge-base --is-ancestor 69726bf HEAD && echo '包含最终适配提交'
```

预期输出为：

```text
包含最终适配提交
```

如果已经有本地仓库：

```bash
git fetch origin repro/touwaeriol-custom-20260720
git switch repro/touwaeriol-custom-20260720
git pull --ff-only
git status --short --branch
```

## 基础环境检查

```bash
go version
python3 --version
node --version
bun --version
git status --short --branch
```

项目要求 Go `1.22+`，前端首选 Bun。若 `bun --version` 直接提示 `command not found`，说明前端报错首先是环境缺少 Bun，不是 TypeScript 代码编译失败。

安装前端依赖后，可执行完整本地验证：

```bash
go test ./... -count=1
python3 -m py_compile deploy/retry-proxy/gpt56_retry_proxy.py
python3 -m unittest discover -s deploy/tests -v
cd web
bun install --frozen-lockfile
cd default
bun run typecheck
bun run build
cd ../..
git diff --check
```

## 问题总表

| 编号 | 问题 | 典型现象 | 复现类型 | 当前状态 |
|---|---|---|---|---|
| ENV-01 | 使用了错误或不完整的 Git 分支 | 找不到定制文件、测试数量不一致、最终 GPT-5.6 规则缺失 | 本地 Git | 已建立独立复现分支 |
| ENV-02 | 本机缺少 Bun | 执行前端命令立即出现 `bun: command not found` | 本地环境 | 配置或运行环境问题 |
| DEP-01 | 历史直连容器占用公网 `3000` | retry proxy 因 `Address already in use` 重启，客户端实际绕过代理 | 生产只读核验 | 最后核验时存在，需重新确认 |
| DEP-02 | 生产代理文件落后于 GitHub | 本地测试通过，但线上仍可能缺少流式前导缓冲和 64 KiB 限制 | 生产哈希核验 | 代码已修复，生产待核验 |
| PROXY-01 | 代理未转发完整 HTTP 方法 | 后台创建、修改或删除操作返回 HTTP `501` | Python 回归测试或 HTTP 请求 | 已修复并有回归测试 |
| PROXY-02 | 可重试 HTTP 状态未重试 | 上游瞬时 `500/502/503/504` 直接返回客户端 | Python 回归测试 | 已修复并有回归测试 |
| PROXY-03 | 非流式 HTTP 200 内含 `response.failed` | HTTP 状态看似成功，但 Responses 事件实际失败 | Python 回归测试 | 已修复并有回归测试 |
| PROXY-04 | 流式请求在有效输出前失败 | 收到 `response.created` 后马上 `response.failed`，客户端拿到半截失败流 | Python 回归测试 | 已修复并有回归测试 |
| PROXY-05 | 输出开始后错误重放 | 客户端可能收到重复文本、重复工具调用或两个响应拼接 | Python 回归测试 | 已修复并有回归测试 |
| PROXY-06 | 流式前导缓冲没有硬上限 | 上游持续发送无有效输出事件时缓冲持续增长 | Python 回归测试 | 已修复并有回归测试 |
| PROXY-07 | 全局并发被误认为单渠道容量 | 设置代理并发 `800` 后仍出现单 Key RPM/TPM 或并发饱和 | 单元测试与压力测试 | 代理全局限制保留，渠道容量已补充 |
| SYS-01 | 代理文件描述符不足 | 日志出现 `Too many open files`，高并发连接失败 | 生产只读核验 | 配置或运行环境问题 |
| MODEL-01 | GPT-5.6 SOL 不接受部分采样参数 | 上游返回 HTTP `400`，指出 `temperature` 或 `top_p` 不支持 | 参数配置测试或真实渠道请求 | 已修复并有回归测试 |
| MODEL-02 | GPT-5.6 SOL 输出预算过低 | HTTP `200`，但最终 `response.incomplete`，无可见正文 | 参数配置测试或真实渠道请求 | 已修复并有回归测试 |
| MODEL-03 | Claude Fable 5 参数不兼容或空回 | `temperature/top_p/top_k` 导致失败，低 `max_tokens` 可能无正文 | 参数配置测试或真实渠道请求 | 已修复并有回归测试 |
| MODEL-04 | 同模型不同渠道的参数覆盖漂移 | 一个渠道正常，另一个渠道对同一请求持续 `400` 或空回 | SQL/后台只读核验 | 仓库有统一配置，生产需核验 |
| MSG-01 | 将 `/v1/messages` 错误转换为 Chat Completions | Anthropic 原生字段丢失，流式 Messages 无法正确工作 | 路径转发核验 | 历史错误方案，已撤销 |
| ROUTE-01 | 原系统没有单渠道并发、RPM、TPM 限制 | 静态权重继续把请求送入已达到上游额度的渠道 | Go 回归测试或压测 | 已修复并有回归测试 |
| ROUTE-02 | 高优先级或加权候选饱和后仍被选择 | 有可用备用渠道时仍返回限流或上游错误 | Go 回归测试 | 已修复并有回归测试 |
| ROUTE-03 | 渠道亲和命中已饱和渠道 | affinity 强制复用旧渠道，无法切换到可用渠道 | Go 回归测试 | 已修复并有回归测试 |
| ROUTE-04 | 所有匹配渠道饱和时错误不明确 | 无法区分渠道容量耗尽与普通供应商错误 | Go 回归测试 | 已修复并有回归测试 |
| ROUTE-05 | 并发租约未释放或 TPM 未核销 | 请求结束后渠道永久显示繁忙，或 TPM 可被少报/绕过 | Go 回归测试 | 已修复并有回归测试 |
| ERR-01 | 把 `401/402/无渠道` 当成可重试源码故障 | 重试多次仍失败，掩盖 Token、余额或渠道数据库状态 | HTTP 与数据库核验 | 配置或运行环境问题 |
| DB-01 | Supabase Direct Connection 只有 IPv6 | 原生产服务器无 IPv6 出口，直连超时或无法建立连接 | 网络只读核验 | 已改用 Session Pooler，仍需按环境确认 |
| DB-02 | 数据迁移完成被误认为生产已经切换 | Supabase 数据完整，但生产 New API 仍连接原 PostgreSQL | 环境变量与业务请求核验 | 迁移完成，生产切换未在报告中确认 |

## 详细复现方式

### ENV-01：错误或不完整的 Git 分支

查看远端两个相关分支：

```bash
git ls-remote --heads origin \
  integration/touwaeriol-custom \
  repro/touwaeriol-custom-20260720
```

历史集成分支指向：

```text
0d0fab6
```

复现分支必须包含最后四个遗漏修改对应的提交：

```text
69726bf
```

确认最终 GPT-5.6 输出预算规则存在：

```bash
git show --stat 69726bf
git show 69726bf:deploy/channel-overrides/azure-gpt-5.6-sol.json
```

### ENV-02：缺少 Bun

```bash
bun --version
cd web/default
bun run typecheck
```

问题环境的直接证据：

```text
bun: command not found
```

这类失败发生在前端编译器启动之前。安装 Bun 和依赖后，才有资格判断是否存在 TypeScript 或构建错误。

### DEP-01：端口 `3000` 被历史直连容器占用

以下命令全部是只读核验：

```bash
sudo ss -lntp | grep -E ':3000|:3001'
sudo systemctl status gpt56-retry-proxy.service --no-pager
sudo journalctl -u gpt56-retry-proxy.service -n 100 --no-pager
docker ps --format 'table {{.Names}}\t{{.Ports}}\t{{.Image}}'
```

问题成立的证据：

- `3000` 的监听进程不是 `gpt56_retry_proxy.py`；
- 日志重复出现 `Address already in use`；
- 某个旧 New API 容器直接绑定公网 `3000`；
- `3001` 才是当前主 New API 容器。

端到端对比：

```bash
curl -fsS http://127.0.0.1:3000/api/status
curl -fsS http://127.0.0.1:3001/api/status
```

两个端口都返回 HTTP 200 并不能证明代理生效，仍必须确认 `3000` 的监听 PID。

### DEP-02：线上代理落后于仓库

```bash
sha256sum deploy/retry-proxy/gpt56_retry_proxy.py
sudo sha256sum /opt/gpt56-retry-proxy/gpt56_retry_proxy.py
```

问题成立的证据是两个 SHA-256 不一致。`2026-07-20` 的历史核验显示服务器文件对应首版代理，而仓库已包含后续流式安全修复；当前状态必须重新执行命令确认。

### PROXY-01 至 PROXY-07：代理转发、重试、流式安全和全局并发

运行全部代理回归测试：

```bash
python3 -m unittest discover -s deploy/tests -v
```

运行单个场景：

```bash
python3 -m unittest -v \
  deploy.tests.test_gpt56_retry_proxy.RetryProxyTests.test_management_http_methods_are_forwarded

python3 -m unittest -v \
  deploy.tests.test_gpt56_retry_proxy.RetryProxyTests.test_retryable_http_status_is_retried

python3 -m unittest -v \
  deploy.tests.test_gpt56_retry_proxy.RetryProxyTests.test_non_streaming_response_failed_is_retried

python3 -m unittest -v \
  deploy.tests.test_gpt56_retry_proxy.RetryProxyTests.test_streaming_response_failed_before_output_is_retried

python3 -m unittest -v \
  deploy.tests.test_gpt56_retry_proxy.RetryProxyTests.test_streaming_failure_after_output_is_not_replayed

python3 -m unittest -v \
  deploy.tests.test_gpt56_retry_proxy.RetryProxyTests.test_sse_prelude_reader_enforces_hard_byte_limit

python3 -m unittest -v \
  deploy.tests.test_gpt56_retry_proxy.RetryProxyTests.test_responses_requests_respect_upstream_concurrency
```

这些测试通过表示当前修复存在。要验证测试确实能够捕获历史问题，应在临时 worktree 中检出对应修复之前的提交，而不要破坏当前工作目录。

### SYS-01：文件描述符不足

```bash
sudo systemctl show gpt56-retry-proxy.service -p LimitNOFILE
sudo journalctl -u gpt56-retry-proxy.service --since '24 hours ago' --no-pager \
  | grep -iE 'too many open files|emfile'
```

问题成立的证据：日志出现 `Too many open files` 或 `EMFILE`，且有效 `LimitNOFILE` 低于部署约定值 `65536`。

### MODEL-01、MODEL-02：GPT-5.6 SOL 参数问题

配置回归测试：

```bash
python3 -m unittest -v \
  deploy.tests.test_gpt56_retry_proxy.ChannelOverrideProfileTests.test_azure_profile_matches_confirmed_sampling_compatibility
```

检查配置内容：

```bash
cat deploy/channel-overrides/azure-gpt-5.6-sol.json
```

真实渠道复现模板：

```bash
export NEW_API_BASE_URL='http://127.0.0.1:3000'
export NEW_API_TOKEN='使用本地或测试环境 Token，不要写入 Git'

curl -N -sS "$NEW_API_BASE_URL/v1/responses" \
  -H "Authorization: Bearer $NEW_API_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "gpt-5.6-sol",
    "input": "只回复 OK",
    "temperature": 0.2,
    "top_p": 0.9,
    "max_output_tokens": 100,
    "stream": true
  }'
```

历史问题的证据包括：

- HTTP `400` 指出 `temperature` 或 `top_p` 不受支持；
- HTTP `200`，但事件以 `response.incomplete` 结束；
- `incomplete_details.reason` 为 `max_output_tokens`；
- 没有任何可见文本输出。

### MODEL-03：Claude Fable 5 参数不兼容或空回

```bash
python3 -m unittest -v \
  deploy.tests.test_gpt56_retry_proxy.ChannelOverrideProfileTests.test_fable_profile_contains_all_production_rules
cat deploy/channel-overrides/claude-fable-5.json
```

真实渠道测试应分别发送带 `temperature`、`top_p`、`top_k` 和较小 `max_tokens` 的请求。问题证据是参数校验错误，或请求成功但没有可见正文。不要在文档或命令历史中粘贴生产 Key。

### MODEL-04：不同渠道的参数覆盖发生漂移

仓库侧核验：

```bash
python3 -m unittest -v \
  deploy.tests.test_gpt56_retry_proxy.ChannelOverrideProfileTests.test_sql_maps_all_production_channels_to_the_correct_profile
```

后台核验步骤：

1. 打开四个目标渠道：`AWS-B`、`0718-OR`、`az-ch0718`、`07-19-AZ-COLIN-OF-001`；
2. 查看每个渠道的参数覆盖；
3. 对同一模型、同一请求分别执行渠道测试；
4. 如果只有一个渠道报参数错误或空回，则确认存在配置漂移。

`2026-07-20` 的历史核验发现 `0718-OR` 缺少按 `model` 匹配的 `max_tokens=512` 规则；当前生产是否仍存在必须重新检查。

### MSG-01：错误转换 `/v1/messages`

路径转发核验：

```bash
curl -sS -o /dev/null -w '%{http_code}\n' \
  -X POST http://127.0.0.1:3000/v1/messages \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer TEST_TOKEN' \
  -d '{}'
```

核验重点不是请求是否认证成功，而是代理日志和上游日志中的目标路径仍为 `/v1/messages`，不能被改写成 `/v1/chat/completions`。历史转换方案会改变 Anthropic 协议语义并丢失原生字段。

### ROUTE-01 至 ROUTE-05：渠道容量和饱和路由

运行容量状态测试：

```bash
go test ./service -run 'TestChannelCapacity' -count=1 -v
```

覆盖场景：

- 并发租约在请求结束后释放；
- RPM 在请求释放后仍保留滚动窗口计数；
- TPM 根据实际 usage 与预估 token 进行差额核销。

运行路由测试：

```bash
go test ./middleware -run 'TestSelectChannelByCapacity' -count=1 -v
go test ./model -run 'TestGetRandomSatisfiedChannelExcluding' -count=1 -v
```

覆盖场景：

- 加权候选饱和时跳过该渠道；
- 高优先级渠道饱和时回落到其他可用优先级；
- 所有候选饱和时返回容量耗尽；
- affinity 指向饱和渠道时切换到可用渠道；
- 排除饱和渠道不会破坏缓存候选集合。

人工压测复现时，至少准备两个支持同一模型的测试渠道：

1. 渠道 A 设置较小并发或 RPM，例如并发 `1`；
2. 渠道 B 保持可用；
3. 同时发送至少两个请求；
4. 检查第二个请求是否被分流到 B；
5. 再将 A、B 都设为 `1` 并同时占满，确认额外请求返回明确限流错误；
6. 请求结束后重新发送，确认并发租约已释放。

不要使用生产 Key 做无上限压力测试。

### ERR-01：401、402 和无渠道错误被误判为代理问题

```bash
curl -i -sS http://127.0.0.1:3000/v1/responses \
  -H 'Authorization: Bearer INVALID_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{"model":"gpt-5.6-sol","input":"ping"}'
```

判断方式：

- `401`：优先检查客户端 Token；
- `402`：优先检查余额和额度；
- `503` 且提示无可用渠道：检查渠道启用状态、分组、模型映射和数据库；
- 不应把以上错误伪装成 Responses 重试逻辑或源码编译问题。

### DB-01：Supabase Direct Connection 的 IPv6 可达性

```bash
getent ahosts "$SUPABASE_DIRECT_HOST"
ip -6 route
ping -6 -c 3 "$SUPABASE_DIRECT_HOST"
```

问题成立的证据：目标只解析出 IPv6 地址，而生产服务器没有可用 IPv6 默认路由或连接持续超时。迁移报告使用 `us-west-2` Session Pooler `5432` 完成恢复，不能据此推断 Direct Connection 在所有服务器都可用。

### DB-02：迁移完成但生产没有切换

容器环境只读核验：

```bash
docker inspect new-api \
  --format '{{range .Config.Env}}{{println .}}{{end}}' \
  | grep '^SQL_DSN=' \
  | sed -E 's#(://)[^@]+@#\1***:***@#'
```

只显示脱敏后的主机信息，不要把完整 DSN 粘贴到终端记录、Issue 或 Git。

迁移完整性基准见：

```text
../server_migration/SUPABASE_MIGRATION_REPORT_20260719.md
```

报告确认数据已恢复到 Supabase，但同时明确记录生产 New API DSN 尚未切换。正式判定切换完成还必须验证登录、Token、渠道、Responses、Chat Completions 和双 Key 分流。

## 最短复现检查单

只检查代码是否完整：

```bash
git rev-parse --short HEAD
go test ./... -count=1
python3 -m unittest discover -s deploy/tests -v
cd web/default && bun run typecheck && bun run build
```

只检查生产入口是否真的经过代理：

```bash
sudo ss -lntp | grep -E ':3000|:3001'
sudo systemctl status gpt56-retry-proxy.service --no-pager
sudo journalctl -u gpt56-retry-proxy.service -n 100 --no-pager
```

只检查模型参数问题：

```bash
python3 -m unittest -v \
  deploy.tests.test_gpt56_retry_proxy.ChannelOverrideProfileTests
```

只检查渠道容量问题：

```bash
go test ./service ./middleware ./model \
  -run 'ChannelCapacity|SelectChannelByCapacity|GetRandomSatisfiedChannelExcluding' \
  -count=1 -v
```

## 关联原始记录

- `CUSTOM_DEPLOYMENT.zh-CN.md`：生产部署、代理、渠道参数和历史问题矩阵；
- `.trellis/spec/backend/production-new-api-adaptations.md`：生产适配技术约束；
- `.trellis/tasks/07-20-channel-capacity-routing/`：渠道容量需求、设计和实施计划；
- `.trellis/tasks/07-20-merge-custom-touwaeriol/`：上游合并过程和风险；
- `../server_migration/SUPABASE_MIGRATION_REPORT_20260719.md`：Supabase 迁移结果；
- `../server_migration/EXPORT_STATUS.zh-CN.md`：生产导出和恢复验证结果。
