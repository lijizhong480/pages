# Pages

一个面向团队和 AI Agent 的轻量 HTML 发布平台。使用工作空间隔离权限，以项目组织页面，支持浏览器和 API 发布。

## 产品截图

### 登录页

![Pages 登录页](docs/images/login.png)

### 工作空间

![Pages 工作空间](docs/images/workspace.png)

## 核心能力

- 工作空间和项目管理
- 用户登录、用户管理和平台角色管理
- 工作空间成员授权，支持所有者、管理员、编辑者、查看者
- 工作空间级 API Token，支持 `read`、`write`、`admin` 权限
- HttpOnly 会话 Cookie、CSRF 防护和 PBKDF2 密码哈希
- Token 明文仅创建时返回，服务端只保存 SHA-256 哈希
- 支持 multipart、JSON 和原始 HTML 上传
- 页面列表、覆盖更新和删除
- 上传先生成独立预览版本，确认后再正式发布
- 完整版本历史、任意版本预览和一键版本切换/回滚
- 旧版数据自动迁移，旧 `/p/{slug}` 地址继续可用
- 本地文件持久化和原子索引写入
- 上传页面通过 CSP sandbox 与管理端隔离
- 单文件最大 5MB，slug 严格校验

## 启动

```bash
PAGE_TOKEN='换成一个长随机字符串' go run .
```

打开 <http://localhost:8080>。首次访问会引导创建第一个平台管理员账号，原有工作空间会自动授权给该管理员。

`PAGE_TOKEN` 继续作为自动化和紧急管理使用的平台级 API Token；浏览器管理端使用用户登录，不再保存管理 Token。

| 变量 | 默认值 | 说明 |
|---|---|---|
| `PAGE_TOKEN` | `dev-token` | 平台管理员密钥，生产环境必须设置 |
| `PAGE_ADDR` | `:8080` | 监听地址 |
| `PAGE_DATA` | `./data` | 元数据和页面存储目录 |

## 数据结构

```text
data/
├── state.json
└── workspaces/
    └── {workspace}/projects/{project}/pages/{slug}.html
```

首次启动新版服务时，原 `data/index.json` 和 `data/pages` 中的页面会复制到 `default/legacy`。旧文件不会被删除。

## API 示例

### 创建工作空间和项目

```bash
curl -X POST http://localhost:8080/api/workspaces \
  -H "Authorization: Bearer $PAGE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"slug":"acme","name":"Acme 团队"}'

curl -X POST http://localhost:8080/api/workspaces/acme/projects \
  -H "Authorization: Bearer $PAGE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"slug":"website","name":"品牌官网"}'
```

### 创建工作空间 Token

```bash
curl -X POST http://localhost:8080/api/workspaces/acme/tokens \
  -H "Authorization: Bearer $PAGE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"AI 发布助手","scopes":["read","write"]}'
```

响应中的 `secret` 只显示一次。之后可使用该 Token 管理 `acme` 工作空间，但无法访问其他空间。

### 创建预览版本

```bash
curl -X POST http://localhost:8080/api/workspaces/acme/projects/website/pages \
  -H "Authorization: Bearer $WORKSPACE_TOKEN" \
  -F "slug=home" \
  -F "title=首页" \
  -F "file=@./index.html"
```

也支持 JSON：

```bash
curl -X POST http://localhost:8080/api/workspaces/acme/projects/website/pages \
  -H "Authorization: Bearer $WORKSPACE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"slug":"hello","title":"Hello","html":"<!doctype html><h1>Hello</h1>"}'
```

上传不会立即改变线上内容。响应包含 `latestVersion` 和 `latestPreviewUrl`，确认预览后发布：

```bash
curl -X POST http://localhost:8080/api/workspaces/acme/projects/website/pages/home/publish \
  -H "Authorization: Bearer $WORKSPACE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"version":"ver_xxx"}'
```

查询历史版本：

```text
GET /api/workspaces/{workspace}/projects/{project}/pages/{slug}/versions
```

正式展示地址保持不变：

```text
GET /p/{workspace}/{project}/{slug}
```

## 权限

平台角色：

| 角色 | 能力 |
|---|---|
| 平台管理员 | 管理用户、所有工作空间及平台配置 |
| 普通成员 | 只能访问明确授权的工作空间 |

工作空间角色：

| 角色 | 能力 |
|---|---|
| 所有者 | 管理成员、项目、页面和 API Token |
| 空间管理员 | 管理成员、项目、页面和 API Token |
| 编辑者 | 查看项目并发布、覆盖、删除页面 |
| 查看者 | 查看项目和页面列表 |

API Token Scope：

| Scope | 能力 |
|---|---|
| `read` | 查看所属工作空间、项目和页面 |
| `write` | 上传、覆盖和删除页面 |
| `admin` | 管理项目和工作空间 Token，并包含读写权限 |

只有平台管理员 Token 可以创建或删除工作空间。

## Docker

在 `.env` 中设置 `PAGE_TOKEN` 后运行：

```bash
docker compose up -d --build
```

`./data` 会挂载到容器的 `/app/data`。

## 验证

```bash
go test ./...
go vet ./...
```

## 当前边界

当前版本聚焦单 HTML 页面发布。面向 AI 的下一阶段是 Deployment/Preview/Release：先生成预览、通过校验或审批后再切换生产版本，并支持回滚和多文件站点。
