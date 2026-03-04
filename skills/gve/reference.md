# GVE 详细参考文档

## 1. wk-ui 资产库完整结构

```
wk-ui/
├── registry.json                       # 版本索引（由 gve registry build 自动生成）
│
└── assets/
    ├── base-setup/
    │   └── v1.0.0/
    │       ├── meta.json               # dest: "site"（全局资产）
    │       ├── embed.go
    │       ├── package.json            # React 19、Tailwind 4、Vite 6
    │       ├── vite.config.ts
    │       ├── tsconfig.json
    │       ├── biome.json
    │       ├── index.html
    │       ├── .gitignore
    │       ├── src/app/main.tsx
    │       ├── src/app/routes.tsx
    │       ├── src/app/providers.tsx
    │       ├── src/app/styles/globals.css
    │       └── src/shared/lib/cn.ts
    │
    ├── button/
    │   ├── v1.0.0/
    │   │   ├── meta.json
    │   │   └── button.tsx
    │   └── v1.2.0/                     # 新版本 = 新目录
    │       ├── meta.json
    │       └── button.tsx
    │
    ├── data-table/
    │   └── v2.0.0/
    │       ├── meta.json
    │       ├── data-table.tsx
    │       ├── data-table-columns.tsx
    │       └── data-table.module.css
    │
    └── theme/
        └── v1.0.0/
            ├── meta.json               # dest: "site/src/app/styles"（全局资产）
            └── globals.css
```

### registry.json 格式

```json
{
  "button": {
    "latest": "1.2.0",
    "versions": {
      "1.0.0": { "path": "assets/button/v1.0.0" },
      "1.2.0": { "path": "assets/button/v1.2.0" }
    }
  },
  "data-table": {
    "latest": "2.0.0",
    "versions": {
      "2.0.0": { "path": "assets/data-table/v2.0.0" }
    }
  },
  "theme": {
    "latest": "1.0.0",
    "versions": {
      "1.0.0": { "path": "assets/theme/v1.0.0" }
    }
  },
  "base-setup": {
    "latest": "1.0.0",
    "versions": {
      "1.0.0": { "path": "assets/base-setup/v1.0.0" }
    }
  }
}
```

**重要**：`registry.json` 是生成产物，不要手动修改版本顺序或 latest 字段，由 `gve registry build` 维护。

### 新增 UI 资产完整流程

```bash
# 1. 创建目录
mkdir -p assets/my-component/v1.0.0

# 2. 编写资产文件（如 my-component.tsx）

# 3. 编写 meta.json
cat > assets/my-component/v1.0.0/meta.json << 'EOF'
{
  "name": "my-component",
  "version": "1.0.0",
  "deps": ["some-npm-package"],
  "files": ["my-component.tsx"]
}
EOF

# 4. 更新 registry.json
gve registry build

# 5. 提交
git add assets/my-component/ registry.json
git commit -m "feat(ui): add my-component v1.0.0"
```

---

## 2. wk-api 资产库完整结构（thrift only）

```
wk-api/
├── registry.json
│
├── ai-console/
│   └── user/
│       ├── v1/
│       │   └── user.thrift         # Thrift IDL
│       └── v2/                     # 破坏性变更才升大版本
│           └── user.thrift
│
└── ai-worker/
    └── task/
        └── v1/
            └── task.thrift
```

### API registry.json 格式

```json
{
  "ai-console/user": {
    "latest": "v2",
    "versions": {
      "v1": { "path": "ai-console/user/v1" },
      "v2": { "path": "ai-console/user/v2" }
    }
  },
  "ai-worker/task": {
    "latest": "v1",
    "versions": {
      "v1": { "path": "ai-worker/task/v1" }
    }
  }
}
```

### 版本策略

- **大版本**（`v1`、`v2`）：有破坏性变更才升
- **目录即版本**：`v1/` 和 `v2/` 并存，业务项目自行选择
- **零工具链依赖**：使用方从资产库拉取，不需要安装 Thrift 编译器

### 新增 API 契约流程

```bash
# 在 wk-api 仓库内
mkdir -p ai-worker/new-service/v1

# 编写 thrift 文件：new-service.thrift

# 更新 registry.json（手动编辑或 gve registry build）

git add ai-worker/new-service/ registry.json
git commit -m "feat(api): add ai-worker/new-service v1"
```

### 项目内自建 API（无需安装 thriftgo）

`gve` 内置 `github.com/cloudwego/thriftgo`，使用方无需在本机安装 thriftgo 二进制。

```bash
# 在业务项目目录
gve api new my-app/task         # 生成 api/my-app/task/v1/task.thrift
# 编辑 thrift：补充 struct / service 方法
gve api generate                # 生成 internal/api/.../{resource}.go + client.go，以及 site/src/api/.../client.ts
```

默认行为：
- 仅扫描规范目录：`api/*/*/v*/{resource}.thrift`
- 任意一个 thrift 生成失败时立即中断（fail-fast）
- `api/` 目录仅保留 `.thrift`（共享契约源）
- Go 生成物输出到 `internal/api/{project}/{resource}/{version}/`
- TS 生成物输出到 `site/src/api/{project}/{resource}/{version}/`
- 永不覆盖 `.thrift`；会覆盖生成物 `*.go`、`client.go`、`client.ts`

---

## 3. gve CLI 缓存机制

- UI 缓存：`~/.gve/cache/ui/` — clone/pull wk-ui
- API 缓存：`~/.gve/cache/api/` — clone/pull wk-api
- `gve ui add` / `gve api add` 先更新缓存，再从缓存复制到项目

**缓存刷新**：每次执行 add/sync 命令时自动 `git pull`。

---

## 4. base-setup 资产说明

`base-setup` 是特殊的全局资产（`dest: "site"`），由 `gve init` 自动安装，提供：

- `site/embed.go` — `go:embed all:dist` 嵌入指令
- `site/package.json` — 最小依赖集（react、react-dom、react-router、clsx、tailwind-merge、tailwindcss、vite 等）
- `site/vite.config.ts` — `@/` 别名、`/api/*` 代理到 Go 后端（`:8080`）
- `site/tsconfig.json` — strict 模式、路径别名
- `site/biome.json` — Lint + Format 规则
- `site/index.html` — Vite 入口
- `site/src/app/main.tsx` — `ReactDOM.createRoot` 挂载
- `site/src/app/routes.tsx` — 路由表（初始一个空首页）
- `site/src/app/providers.tsx` — 全局 Provider 壳
- `site/src/app/styles/globals.css` — `@import "tailwindcss"` + CSS 变量
- `site/src/shared/lib/cn.ts` — `clsx + tailwind-merge` 封装

---

## 5. Go Embed 机制

`site/embed.go` 在 Go 包中暴露前端静态资源：

```go
package site

import "embed"
import "io/fs"

//go:embed all:dist
var distFS embed.FS

var DistDirFS, _ = fs.Sub(distFS, "dist")
```

在 `cmd/server/main.go` 中集成：

```go
import "myapp/site"
import "net/http"

// API 路由优先
mux.Handle("/api/", apiHandler)
// 静态文件兜底（SPA 模式）
mux.Handle("/", http.FileServer(http.FS(site.DistDirFS)))
```

---

## 6. gve dev 行为说明

- 检测 `air` 是否安装：已安装用 Air 热重载，否则用 `go run ./cmd/server`
- Vite 开发服务器在 `site/` 目录执行 `pnpm dev`
- 输出前缀：`[go]`（蓝色）/ `[vite]`（绿色）
- `Ctrl+C` 同时终止两个进程

---

## 7. gve run 日志管理

```
{project}/.gve/logs/
├── app.log               # symlink → 当前日期文件
├── app-2026-02-26.log    # 当日日志
└── app-2026-02-20.log.gz # 7天前自动 gzip 压缩（30天后删除）
```

---

## 8. 前端技术栈版本

| 技术 | 版本要求 |
|------|---------|
| pnpm | 9.x |
| Vite | 6.x+ |
| React | 19.x |
| TypeScript | 5.7+ |
| Radix UI | 最新 |
| Tailwind CSS | 4.x |
| Go | ≥ 1.22 |
| Node.js | ≥ 18 |

---

## 9. 环境变量 / 配置

`gve` 读取以下默认配置（`~/.gve/config.json` 可覆盖）：

| 配置项 | 默认值 |
|--------|--------|
| UIRegistry | `github.com/castle-x/wk-ui` |
| APIRegistry | `github.com/castle-x/wk-api` |
| CacheDir | `~/.gve/cache/` |

---

## 10. 后端项目结构规范

### 10.1 分层职责

```
cmd/server/main.go         路由注册 + 服务启动，不写任何业务逻辑
internal/handler/          HTTP 层
internal/service/          业务层
internal/repo/（可选）      数据访问层
internal/model/（可选）     领域模型 / 数据结构
api/                       只读，由 gve api add 管理
```

### 10.2 main.go 职责

```go
// cmd/server/main.go — 只注册路由 + 启动
func main() {
    mux := http.NewServeMux()

    // API 路由优先
    userHandler := handler.NewUserHandler(service.NewUserService())
    mux.HandleFunc("/api/users", userHandler.List)
    mux.HandleFunc("/api/users/{id}", userHandler.Get)

    // 静态文件兜底（SPA）
    mux.Handle("/", http.FileServer(http.FS(site.DistDirFS)))

    log.Fatal(http.ListenAndServe(":8080", mux))
}
```

### 10.3 handler 层职责

```go
// internal/handler/user_handler.go
// 只做：解析请求 → 调用 service → 返回 JSON
type UserHandler struct { svc *service.UserService }

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
    users, err := h.svc.ListUsers(r.Context())
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(users)
}
```

- **禁止**：handler 层直接读写数据库，或包含业务规则判断。

### 10.4 service 层职责

```go
// internal/service/user_service.go
// 只做业务逻辑，不知道 HTTP 协议
type UserService struct { /* repo 依赖注入 */ }

func (s *UserService) ListUsers(ctx context.Context) ([]model.User, error) {
    // 业务校验、数据组装等
}
```

- **禁止**：service 依赖 `net/http`、`http.Request`、`http.ResponseWriter`。

### 10.5 业务扩展原则

- **可以**添加：`internal/repo/`、`internal/model/`、`internal/middleware/`、`internal/config/` 等
- **禁止**：在 `api/` 目录手动创建或修改文件（该目录由 `gve api add` 独占管理）
- **建议**：每个业务资源对应一套文件，如 `user_handler.go` / `user_service.go` / `user_repo.go`，避免堆在同一文件里

---

## 11. 前端项目结构规范

### 11.1 src 完整目录说明

```
site/src/
├── app/                      # 框架初始化层（不放业务代码）
│   ├── main.tsx              # 入口：仅 createRoot + <App> 或 <RouterProvider>
│   ├── routes.tsx            # 路由定义，import views 中的页面
│   ├── providers.tsx         # 全局 Provider 组合（不包含业务逻辑）
│   └── styles/
│       └── globals.css       # CSS 变量定义 + @import "tailwindcss"
│
├── views/                    # 业务页面层
│   ├── home/
│   │   └── index.tsx         # 首页（路由直接引用此文件）
│   ├── settings/
│   │   ├── index.tsx         # 设置页
│   │   └── components/       # 该页面私有组件
│   │       └── ThemeToggle.tsx
│   └── {feature}/
│       ├── index.tsx
│       └── components/
│
└── shared/                   # 跨 views 的公共代码
    ├── ui/                   # UI 资产（gve ui add 安装，禁止手写）
    │   ├── button/
    │   │   └── button.tsx
    │   └── data-table/
    │       ├── data-table.tsx
    │       └── data-table.module.css
    ├── lib/                  # 纯工具函数（无副作用，无业务逻辑）
    │   ├── cn.ts             # clsx + tailwind-merge（base-setup 提供）
    │   └── request.ts        # fetch 封装（按需添加）
    ├── hooks/                # 通用 React hooks（可选）
    │   └── use-debounce.ts
    └── types/                # 跨模块共享的 TypeScript 类型（可选）
        └── common.ts
```

### 11.2 依赖方向（严格单向）

```
app  →  views  →  shared
             ↘  shared
app  →  shared
```

- `views/{featureA}` **禁止** import `views/{featureB}`
- `shared/` 内部各目录 **禁止** 互相依赖
- `app/` **禁止** 包含业务状态或业务逻辑

### 11.3 views 页面命名约定

```tsx
// views/users/index.tsx — 路由引用的入口
export { UsersPage as default } from './UsersPage'

// views/users/UsersPage.tsx — 实际页面组件
export function UsersPage() { ... }

// views/users/components/UserCard.tsx — 页面私有组件
export function UserCard({ user }: { user: User }) { ... }
```

### 11.4 shared/ui 使用约定

- `shared/ui/` 下的组件由 `gve ui add` 安装管理，**不要在此手写组件**
- 使用方式：

```tsx
// 正确：从 shared/ui 引入 gve 管理的组件
import { Button } from '@/shared/ui/button/button'

// 错误：在 shared/ui/ 下手动创建 my-custom.tsx
```

- 若某组件仅在单个 feature 使用 → 放 `views/{feature}/components/`
- 若某组件跨多个 feature 使用但不是 wk-ui 资产 → 评估是否添加到 wk-ui，或临时放 `shared/lib/`

### 11.5 禁止事项汇总

| 禁止行为 | 正确做法 |
|----------|----------|
| 直接在 `src/` 下创建 `.tsx` 文件 | 放入 `app/`、`views/` 或 `shared/` |
| 在 `shared/ui/` 手写组件 | 通过 `gve ui add` 安装，或放 `views/{feature}/components/` |
| `views/featureA` import `views/featureB` | 将共用部分提取到 `shared/` |
| `src/app/` 写业务状态或 API 调用 | 放入对应 `views/` 或 `shared/hooks/` |
| 在 `shared/lib/` 写业务逻辑 | 放入对应 `views/` 或 `service` 层 |
| 任意创建顶层目录（如 `src/widgets/`、`src/entities/`）| 遵循 `app/views/shared` 三层结构 |
