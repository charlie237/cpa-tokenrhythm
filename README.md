# Token Rhythm Balance (CLIProxyAPI 插件)

[CLIProxyAPI (CPA)](https://github.com/router-for-me/CLIProxyAPI) 的标准动态库插件，用于在管理面板中自动查询并展示 [Token Rhythm](https://tokenrhythm.studio) 账户余额、充值/赠送额度和用量统计。

插件通过 CPA 的 Management API + `host.http.do` 代理发起请求，只需在配置中填入一个或多个 `tr_session`，无需在浏览器或插件里保存账号密码。

## 功能

- 多 session 管理：可配置多个 Token Rhythm 账号，面板内切换查看。
- 并发轮询：按 `refresh_interval_seconds` 自动刷新，按 `poll_concurrency` 并行拉取各账号。
- 自动调用 Token Rhythm 接口（钱包为必选，其余为尽力而为，单个失败不影响余额展示）：
  - `GET /api/wallet/summary`（钱包余额，必选）
  - `GET /api/usage-summary`（用量与消耗、新人奖励）
  - `GET /api/usage/panel`（增强用量：按模型/客户端分组，含缓存、推理 token、USD 成本与节省）
  - `GET /api/me`（账号名称、手机号脱敏、状态）
  - `GET /api/wallet/expiring-credits`（各批次赠送额度的剩余与过期时间）
  - `GET /api/api-keys`（API Key 列表）
  - `POST /api/api-keys`（创建 API Key，完整密钥只返回一次）
- 在 CPA 面板“Token Rhythm 账户”菜单内展示：
  - 多账号卡片、可用/赠送/充值/冻结余额，低余额红色高亮
  - 调用次数、成功/失败、成功率、输入/输出/缓存/合计 Tokens、折合 USD
  - 按模型用量表
  - 即将过期额度明细表
  - 新人奖励信息
  - API Key 列表与创建（创建成功后完整 Key 只展示一次）
- 页面自动刷新（间隔可配），也有“刷新全部”按钮。
- 低于 `low_balance_threshold` 时高亮为红色。
- 提供受 Management 认证保护的 JSON 接口，便于监控/自动化。
- 服务端按 session 缓存，避免频繁请求上游。

## 已验证的 Token Rhythm 接口

均以 `tr_session` Cookie 鉴权，`GET`，返回 `{"code":0,"message":"ok","data":...}`：

| 接口 | 用途 |
|---|---|
| `/api/wallet/summary` | 钱包余额（可用/赠送/充值/冻结/负债） |
| `/api/usage-summary` | 汇总用量、成本、下次过期时间、新人奖励 |
| `/api/usage/panel` | 增强用量面板（summary/byModel/byClientApp） |
| `/api/me` | 账号基础信息与状态 |
| `/api/wallet/expiring-credits?page=1&pageSize=20` | 即将过期额度批次 |
| `/api/api-keys` | API Key 列表 |
| `POST /api/api-keys` | 创建 API Key（body: `{"name":"..."}`，需 `X-CSRF-Token`） |
| `/api/api-keys/usage` | 逐 Key 用量 |
| `/api/wallet/transactions`、`/api/call-logs/page` | 钱包流水、调用日志（分页） |
| `/api/models` | 可用模型列表 |
| `/api/referrals/me` | 邀请返利信息 |


## 构建

需要 Go 1.26+ 与 CGO（Windows 下需 MinGW-w64 或 MSVC 提供 `gcc`）。

```bash
# Windows
go build -buildmode=c-shared -o bin/tokenrhythm-balance.dll .

# Linux / macOS（或 make build）
make build
```

产物为 `bin/tokenrhythm-balance.dll`（Linux `.so`，macOS `.dylib`）。插件 ID 为产物文件名去掉扩展名，即 `tokenrhythm-balance`。

## 从 GitHub 安装（推荐，Linux/Windows/macOS）

CPA 插件商店读的是 **GitHub Release 里的 zip**，不是源码。仓库：https://github.com/Charlie237/cpa-tokenrhythm

在 CPA 的 `config.yaml` 里打开插件并加上这个源：

```yaml
plugins:
  enabled: true
  dir: "plugins"
  store-sources:
    - "https://raw.githubusercontent.com/Charlie237/cpa-tokenrhythm/main/registry.json"
  configs:
    tokenrhythm-balance:
      enabled: true
      priority: 1
      sessions:
        - id: main
          name: "主账号"
          tr_session: "tr_session=sess_xxxxxxxx; tr_csrf=xxxxxxxx"
      base_url: "https://tokenrhythm.studio"
      refresh_interval_seconds: 60
      poll_concurrency: 3
      low_balance_threshold: 10
```

然后：

1. 重启 CPA
2. 打开管理面板 → 插件商店，安装 **Token Rhythm Balance**
3. 商店会按当前系统下载 `tokenrhythm-balance_<version>_linux_amd64.zip`（或 windows/darwin），解压到 `plugins/<GOOS>/<GOARCH>/`

打 `v0.2.0` 这类 tag 会自动编各平台动态库并发布 Release。

## 手动安装

把动态库放到 CPA 的 `plugins/`（或 `plugins/<GOOS>/<GOARCH>/`）目录，在 `config.yaml` 中启用：

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    tokenrhythm-balance:
      enabled: true
      priority: 1
      tr_session: "sess_xxxxxxxxxxxxxxxxxxxx"
      sessions:
        - id: main
          name: "主账号"
          tr_session: "tr_session=sess_xxxxxxxxxxxxxxxxxxxx; tr_csrf=xxxxxxxx"
        - id: backup
          name: "备用账号"
          tr_session: "tr_session=sess_yyyyyyyyyyyyyyyyyyyy; tr_csrf=yyyyyyyy"
      base_url: "https://tokenrhythm.studio"
      refresh_interval_seconds: 60
      poll_concurrency: 3
      low_balance_threshold: 10
```

重启 CPA 后，在管理面板插件菜单中打开“Token Rhythm 账户”。

创建 API Key 必须使用包含 `tr_csrf` 的完整 Cookie 串；仅填 `sess_...` 可以查询余额，但不能创建 Key。

### 配置项

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|
| `tr_session` | string | 否* | - | 单账号 Token Rhythm 会话 Cookie。配置了 `sessions` 时忽略 |
| `sessions` | array | 否* | - | 多账号列表。每项：`id`、`name`、`tr_session`。创建 Key 需要完整 Cookie（含 `tr_csrf`） |
| `base_url` | string | 否 | `https://tokenrhythm.studio` | 站点地址 |
| `refresh_interval_seconds` | integer | 否 | `60` | 前端自动刷新与服务端缓存间隔（5–86400 秒） |
| `poll_concurrency` | integer | 否 | `3` | 同时轮询的 session 数量（1–16） |
| `low_balance_threshold` | number | 否 | `10` | 可用余额低于该值标记为“低余额” |

\* `tr_session` 与 `sessions` 至少配置一种。

## 接口

均需 Management 认证（`Authorization: Bearer <管理密钥>` 或 `X-Management-Key`），资源页除外。

- 资源页（面板 iframe）：`GET /v0/resource/plugins/tokenrhythm-balance/balance`
- 余额/用量/Key 汇总：`GET /v0/management/tokenrhythm/balance`（`?refresh=1` 强制刷新，`?session=<id>` 指定回填到顶层字段的账号）
- Session 列表：`GET /v0/management/tokenrhythm/sessions`
- API Key 列表：`GET /v0/management/tokenrhythm/api-keys?session=<id>`
- 创建 API Key：`POST /v0/management/tokenrhythm/api-keys`，body：`{"session":"<id>","name":"可选名称"}`。成功时 `api_key.key` 为完整密钥，只返回这一次。

## 安全说明

- 资源页为静态壳，复用当前面板的管理会话（仅内存中使用，不落盘），自身不包含 `tr_session`。
- `tr_session` / `tr_csrf` 保存在 CPA 插件配置中，请按机密信息管理，不要在日志或截图中泄露。
- 创建 API Key 时完整密钥只在创建响应里返回一次，插件不会把明文 Key 写入余额缓存或列表接口。
- 插件仅为管理员可信配置，安装前请确认来源可信。

## 说明

本项目依赖本地 `CLIProxyAPI` 仓库（`go.mod` 中的 `replace`）以获得 `sdk/pluginabi` 与 `sdk/pluginapi`。若独立发布，可改为依赖已发布的模块版本。
