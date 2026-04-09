# GVE 详细参考文档（v2）

## 目录

1. [wk-ui 资产库完整结构](#1-wk-ui-资产库完整结构)
2. [wk-api 契约库完整结构](#2-wk-api-契约库完整结构thrift-only)
3. [gve CLI 缓存机制](#3-gve-cli-缓存机制)
4. [scaffold 资产说明](#4-scaffold-资产说明)
5. [Go Embed 机制](#5-go-embed-机制)
6. [gve dev 行为说明](#6-gve-dev-行为说明)
7. [gve run 日志管理](#7-gve-run-日志管理)
8. [前端技术栈版本](#8-前端技术栈版本)
9. [环境变量 / 配置](#9-环境变量--配置)
10. [后端项目结构规范](#10-后端项目结构规范)
11. [前端项目结构规范](#11-前端项目结构规范)
12. [peerDeps 解析机制](#12-peerdeps-解析机制)

---

## 1. wk-ui 资产库完整结构

```
wk-ui/
├── registry.json                       # 版本索引（由 gve registry build 自动生成，v2 格式）
│
├── scaffold/                           # 模块1: 项目骨架
│   ├── default/
│   │   └── v1.0.0/
│   │       ├── meta.json              # category: "scaffold", dest: "site"
│   │       ├── embed.go
│   │       ├── package.json           # React 19、Tailwind 4、Vite 7
│   │       ├── app.json              # 品牌配置（name、displayName）
│   │       ├── vite.config.ts
│   │       ├── tsconfig.json
│   │       ├── biome.json
│   │       ├── index.html
│   │       ├── .gitignore
│   │       ├── src/app/main.tsx
│   │       ├── src/app/routes.tsx
│   │       ├── src/app/providers.tsx
│   │       ├── src/app/styles/globals.css
│   │       └── src/shared/lib/cn.ts
│   └── dashboard/
│       └── v1.0.0/
│           ├── meta.json
│           └── ...                    # Dashboard 布局、侧栏等
│
├── ui/                                 # 模块2: 自研 UI 原子组件
│   ├── spinner/
│   │   └── v1.0.0/
│   │       ├── meta.json
│   │       ├── README.md              # 使用说明（Props、用法、变体）
│   │       └── spinner.tsx            # 纯 .tsx，禁止 .css
│   ├── input-group/
│   │   └── v1.0.0/
│   │       ├── meta.json
│   │       └── input-group.tsx
│   └── sub-nav/
│       └── v1.0.0/
│           ├── meta.json
│           └── sub-nav.tsx
│
├── components/                         # 模块3: 业务复杂组件
│   ├── data-table/
│   │   └── v2.0.0/
│   │       ├── meta.json             # peerDeps: ["ui/button", "ui/input-group"]
│   │       └── data-table.tsx
│   └── file-tree/
│       └── v1.0.0/
│           ├── meta.json
│           └── file-tree.tsx
│
└── global/                             # 模块4: 全局配置资产
    └── theme/
        └── v1.0.0/
            ├── meta.json              # category: "global", dest: "site/src/app/styles"
            └── globals.css            # global/ 是唯一允许 .css 的 category
```

### registry.json 格式（v2）

```json
{
  "$schema": "https://gve.dev/schema/registry.json",
  "version": "2",

  "scaffold/default": {
    "latest": "1.0.0",
    "versions": {
      "1.0.0": { "path": "scaffold/default/v1.0.0" }
    }
  },
  "scaffold/dashboard": {
    "latest": "1.0.0",
    "versions": {
      "1.0.0": { "path": "scaffold/dashboard/v1.0.0" }
    }
  },
  "ui/spinner": {
    "latest": "1.0.0",
    "versions": {
      "1.0.0": { "path": "ui/spinner/v1.0.0" }
    }
  },
  "ui/input-group": {
    "latest": "1.0.0",
    "versions": {
      "1.0.0": { "path": "ui/input-group/v1.0.0" }
    }
  },
  "components/data-table": {
    "latest": "2.0.0",
    "versions": {
      "2.0.0": { "path": "components/data-table/v2.0.0" }
    }
  },
  "global/theme": {
    "latest": "1.0.0",
    "versions": {
      "1.0.0": { "path": "global/theme/v1.0.0" }
    }
  }
}
```

**重要**：
- `registry.json` 是生成产物，不要手动修改版本顺序或 latest 字段，由 `gve registry build` 维护
- v2 的 key 带 category 前缀，如 `"ui/spinner"` 而非 `"spinner"`
- `version: "2"` 标识新格式

### `gve registry build` 扫描逻辑

扫描以下四个目录下的 `*/v*/meta.json`：

```
scaffold/*/v*/meta.json
ui/*/v*/meta.json
components/*/v*/meta.json
global/*/v*/meta.json
```

registry key = `{category_dir}/{asset_name}`（如 `ui/spinner`、`components/data-table`）。

**校验规则**：
1. `meta.json` 中 `category` 字段（如有）必须与所在目录一致
2. `ui/` 和 `components/` 目录下的资产，`files` 列表不得包含 `.css` 后缀文件
3. `ui/` 和 `components/` 目录中存在但未声明在 `meta.Files` 中的 `.css` 文件也会被检测
4. 违反时输出 warning

### 新增 UI 资产完整流程（推荐：gve ui push）

在业务项目中修改完组件后直接发布：

```bash
# 1. 修改组件（如 site/src/shared/wk/ui/spinner.tsx）
# 2. 发布新版本（自动扫描 import、生成 meta.json、commit + push）
gve ui push spinner --version 1.1.0 --desc "Spinner with CVA variants"

# 不指定版本 → 自动 patch +1（从 gve.lock 或 registry）
gve ui push spinner

# 预览
gve ui push spinner --dry-run
```

`gve ui push` 完整流程：
1. **定位源目录**：`--source` 优先，否则查找 `shared/wk/ui/{name}.tsx` 或 `shared/wk/components/{name}.tsx`
2. **扫描 TSX import**：遍历 .ts/.tsx（排除 .d.ts），识别 npm deps 和 wk-ui peerDeps
3. **确定版本号**：`--version` > gve.lock patch+1 > registry latest patch+1 > 1.0.0
4. **构建 meta.json**：合并扫描结果 + 文件列表
5. **发布到 registry cache**：git pull → 版本冲突检查 → 复制文件 → 重建 registry.json → git commit + push
6. **更新 gve.lock**

### 新增 UI 资产完整流程（手动方式）

> 仅在无法使用 `gve ui push` 时使用。

```bash
# 1. 创建目录（注意选择正确的 category 目录）
mkdir -p ui/my-component/v1.0.0

# 2. 编写资产文件（仅 .tsx，禁止 .css）
# 编写 my-component.tsx

# 3. 编写 meta.json
cat > ui/my-component/v1.0.0/meta.json << 'EOF'
{
  "$schema": "https://gve.dev/schema/meta.json",
  "name": "my-component",
  "version": "1.0.0",
  "category": "ui",
  "description": "A brief description of the component.",
  "deps": [],
  "files": ["my-component.tsx"]
}
EOF

# 4. 编写 README.md 使用说明（Props、基础用法、CVA 变体）

# 5. 更新 registry.json
gve registry build

# 6. 提交
git add ui/my-component/ registry.json
git commit -m "feat(ui): add ui/my-component v1.0.0"
```

### 新增业务组件完整流程（推荐：gve ui push）

```bash
# 修改 components/data-table 后发布
gve ui push data-table --version 1.1.0 --desc "Data table with filtering"
# 自动识别 peerDeps: ["ui/spinner"]，npm deps: ["@tanstack/react-table", "lucide-react"]
```

### 新增业务组件完整流程（手动方式）

> 仅在无法使用 `gve ui push` 时使用。

```bash
# 1. 创建目录
mkdir -p components/my-widget/v1.0.0

# 2. 编写资产文件
# 编写 my-widget.tsx

# 3. 编写 meta.json（注意声明 peerDeps）
cat > components/my-widget/v1.0.0/meta.json << 'EOF'
{
  "$schema": "https://gve.dev/schema/meta.json",
  "name": "my-widget",
  "version": "1.0.0",
  "category": "component",
  "description": "A complex widget that depends on spinner and button.",
  "deps": ["some-npm-package"],
  "peerDeps": ["ui/spinner", "ui/button"],
  "files": ["my-widget.tsx"]
}
EOF

# 4. 编写 README.md 使用说明（Props、基础用法、CVA 变体、peerDeps 说明）

# 5. 更新 registry.json
gve registry build

# 6. 提交
git add components/my-widget/ registry.json
git commit -m "feat(components): add components/my-widget v1.0.0"
```

---

## 2. wk-api 契约库完整结构（thrift only）

> API 部分在 v2 中无变化，保持 v1 结构。

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

### 新增 API 契约流程（推荐：gve api push）

在业务项目中编辑完 thrift 文件后直接发布：

```bash
# 发布到 wk-api registry（自动 git commit + push）
gve api push ai-worker/task                    # 自动检测版本
gve api push ai-worker/task --version v2       # 指定版本
gve api push ai-worker/task --source ./mythrift/ --version v1
gve api push ai-worker/task --dry-run          # 预览模式
```

`gve api push` 完整流程：
1. **定位源目录**：`--source` 优先，否则查找 `api/{project}/{resource}/` 下最高版本目录
2. **确定版本号**：`--version` > 从源目录名推断（如 `v1/` → push `v1`）
3. **验证 .thrift 文件**：源目录中至少有一个 `.thrift` 文件
4. **发布到 registry cache**：git pull → 版本冲突检查 → 复制 `.thrift` 文件 → 重建 `registry.json` → git commit + push
5. **更新 gve.lock**

与 UI push 的关键区别：
- **无需扫描**：thrift 自包含，不需要分析 import
- **无 meta.json**：API registry 无 meta
- **版本格式**：major-only（v1, v2），不是 semver

### 新增 API 契约流程（手动方式，在 wk-api 仓库）

> 仅在无法使用 `gve api push` 时使用。

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

## 4. scaffold 资产说明

`scaffold/default` 是骨架类资产（`category: "scaffold"`, `dest: "site"`），由 `gve init` 安装，提供：

- `site/embed.go` — `go:embed all:dist` 嵌入指令
- `site/package.json` — 最小依赖集（react、react-dom、react-router、clsx、tailwind-merge、tailwindcss、vite 等）
- `site/app.json` — 品牌配置（`name` 和 `displayName` 字段，使用 `__PROJECT_NAME__` 占位符）
- `site/vite.config.ts` — `@/` 别名、`/api/*` 代理到 Go 后端（`:8080`）
- `site/tsconfig.json` — strict 模式、路径别名
- `site/biome.json` — Lint + Format 规则
- `site/index.html` — Vite 入口（支持 `__PROJECT_NAME__` 占位符替换）
- `site/app.json` — 品牌配置（`name` 和 `displayName` 字段，使用 `__PROJECT_NAME__` 占位符；前端通过 `import app from "app.json"` 读取品牌信息）
- `site/src/app/main.tsx` — `ReactDOM.createRoot` 挂载
- `site/src/app/routes.tsx` — 路由表（初始一个首页，含 API 调用示例）
- `site/src/app/providers.tsx` — 全局 Provider 壳
- `site/src/app/styles/globals.css` — `@import "tailwindcss"` + CSS 变量
- `site/src/shared/lib/cn.ts` — `clsx + tailwind-merge` 封装

### scaffold 专用 meta.json 字段

scaffold 类型的 meta.json 可额外声明两个字段，`gve init` 会自动处理：

```json
{
  "name": "dashboard",
  "category": "scaffold",
  "dest": "site",
  "defaultAssets": ["ui/spinner", "components/settings-dropdown"],
  "shadcnDeps": ["button", "card", "dialog", "sidebar", "tooltip"],
  "files": [...]
}
```

- `defaultAssets`：`gve init` 时自动安装的 wk-ui 组件（含递归 peerDeps 解析）
- `shadcnDeps`：`gve init` 时自动通过 `npx shadcn@latest add` 安装的 shadcn 组件。如果骨架已自带定制的 shadcn 文件（如 dashboard-02 的 hugeicons 迁移版），对应组件会被跳过，不执行 shadcn add。对于仍需安装的组件，使用 `--overwrite` 避免交互提示，但安装前会备份 shadcn 目录中已有的 `.tsx` 文件，安装后自动恢复，防止传递依赖覆盖定制文件

### 品牌名替换

scaffold 文件中可使用 `__PROJECT_NAME__` 占位符，`gve init` 会递归扫描 `site/src/` 下所有 `.ts/.tsx/.js/.jsx/.json/.html` 文件，将 `__PROJECT_NAME__` 替换为实际项目名。同时也替换 `site/package.json` 和 `site/index.html`。

`scaffold/dashboard` 在 `scaffold/default` 基础上额外提供：
- 侧栏布局（SidebarProvider + AppSidebar + SidebarInset）
- Header 组件
- 基本页面骨架
- 依赖的 shadcn 组件配置（通过 `shadcnDeps` 声明）

### nanomind scaffold 首页 SayHello 演示

scaffold 首页包含一个 API 调用示例区域：输入框 + "Say Hello" 按钮，调用生成的 `HelloServiceClient`，展示前后端联通效果。

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
- **包管理器自动检测**：优先检测 `pnpm-lock.yaml`（用 pnpm），其次 `package-lock.json`（用 npm），再次 `exec.LookPath`
- Vite 开发服务器在 `site/` 目录执行 `{pm} dev`（pm = 检测到的包管理器）
- **自动传递 `VITE_BACKEND_TARGET`**：`gve dev` 会自动将 `VITE_BACKEND_TARGET=http://localhost:{port}` 传递给 Vite 进程，确保前端代理始终指向 Go 后端的实际端口
- 启动前自动执行 `{pm} install`（pnpm 不可用时降级为 npm）
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
| Vite | 7.x+ |
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

### i18n（国际化）

CLI 输出支持中文/英文双语。语言检测优先级：

| 环境变量 | 说明 |
|----------|------|
| `GVE_LANG` | 最高优先级，设为 `en` 或 `zh` |
| `LANG` | 系统语言环境变量 |
| `LC_ALL` | 系统语言环境变量 |
| 默认 | `zh`（中文） |

```bash
# 切换为英文输出
export GVE_LANG=en
```

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
│       └── globals.css       # CSS 变量定义 + @import "tailwindcss"（唯一全局 CSS）
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
    ├── shadcn/               # shadcn/ui 组件（npx shadcn add 安装，扁平存放）
    │   ├── button.tsx
    │   ├── dialog.tsx
    │   ├── sidebar.tsx
    │   └── ...
    ├── wk/                   # wk-ui 自研资产（gve ui add 安装）
    │   ├── ui/               # UI 原子组件（扁平存放）
    │   │   ├── spinner.tsx
    │   │   └── input-group.tsx
    │   └── components/       # 业务复杂组件（扁平存放）
    │       └── data-table.tsx
    ├── lib/                  # 纯工具函数（无副作用，无业务逻辑）
    │   ├── cn.ts             # clsx + tailwind-merge（scaffold 提供）
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

### 11.4 shared 三层组件来源

| 来源 | 安装方式 | 目录 | Import 路径 | 可否修改 |
|------|---------|------|-------------|---------|
| shadcn/ui | `npx shadcn add` | `shared/shadcn/` | `@/shared/shadcn/{name}` | 通过 className 扩展 |
| wk-ui (UI 原子) | `gve ui add ui/xxx` | `shared/wk/ui/` | `@/shared/wk/ui/{name}` | 可以，diff 追踪 |
| wk-ui (业务组件) | `gve ui add components/xxx` | `shared/wk/components/` | `@/shared/wk/components/{name}` | 可以，diff 追踪 |

```tsx
// 正确：从各自来源引入
import { Button } from "@/shared/shadcn/button";           // shadcn
import { Spinner } from "@/shared/wk/ui/spinner";          // wk-ui 原子
import { DataTable } from "@/shared/wk/components/data-table"; // wk-ui 业务

// 错误：手动在 shared/wk/ 下创建组件
// 错误：在 shared/shadcn/ 下手动创建非 shadcn 组件
```

### 11.5 禁止事项汇总

| 禁止行为 | 正确做法 |
|----------|----------|
| 直接在 `src/` 下创建 `.tsx` 文件 | 放入 `app/`、`views/` 或 `shared/` |
| 在 `shared/wk/` 手写组件 | 通过 `gve ui add` 安装，或放 `views/{feature}/components/` |
| 在 `shared/shadcn/` 创建非 shadcn 组件 | shadcn 目录仅由 `npx shadcn add` 管理 |
| `views/featureA` import `views/featureB` | 将共用部分提取到 `shared/` |
| `src/app/` 写业务状态或 API 调用 | 放入对应 `views/` 或 `shared/hooks/` |
| 在 `shared/lib/` 写业务逻辑 | 放入对应 `views/` 或 `service` 层 |
| 任意创建顶层目录（如 `src/widgets/`、`src/entities/`）| 遵循 `app/views/shared` 三层结构 |
| 创建独立 `.css` / `.module.css` 文件 | 样式内聚在 `.tsx` 中（Tailwind + cn + cva） |

---

## 12. peerDeps 解析机制

### 12.1 概念

`peerDeps` 声明了一个 wk-ui 资产对其他 wk-ui 资产的依赖关系。典型场景：`components/data-table` 依赖 `ui/button`、`ui/spinner`。

### 12.2 递归解析流程

`gve ui add` 使用 BFS（广度优先）递归解析 peerDeps 链：

```
gve ui add components/data-table
  │
  ├── 读取 meta.json → peerDeps: ["ui/button", "ui/spinner"]
  │
  ├── BFS 遍历 peerDeps 链（最大深度 5 层）
  │   ├── ui/button → 检查其 peerDeps（如有）→ 递归
  │   └── ui/spinner → 检查其 peerDeps（如有）→ 递归
  │
  ├── 拓扑排序：叶子节点先安装（确保依赖顺序）
  │
  ├── 检查 gve.lock
  │   ├── ui/button: 已安装 v1.2.0 ✓（跳过）
  │   └── ui/spinner: 未安装 ✗（安装 latest）
  │
  ├── 安装 components/data-table 到 shared/wk/components/
  │
  ├── 更新 gve.lock
  │
  └── 自动执行 pnpm install（如果有新 npm deps 被注入）
```

### 12.3 循环依赖 & 钻石依赖检测

- **visited set**：防止重复访问同一节点
- **钻石依赖**（A→B→D, A→C→D）：D 只安装一次（合法，不报错）
- **循环依赖**（A→B→A）：跳过已访问节点，不死循环
- **最大深度 5 层**，超出时打印警告并停止

### 12.4 npm deps 自动安装

- `gve ui add` 安装组件后，如果 `injectDeps` 向 `package.json` 注入了新的 npm 依赖，自动执行 `runNodeInstall`
- `deps` 字段支持 `name@version` 格式（如 `"lucide-react@^0.300.0"`），不指定版本时默认 `"latest"`

### 12.5 scaffold 资产的特殊处理

- `gve sync` 和 `gve ui update` **自动跳过** `scaffold/` 前缀的资产（scaffold 只在 `gve init` 时使用，不参与后续升级）
- scaffold 资产的 meta.json 可额外声明 `defaultAssets`（wk 组件）和 `shadcnDeps`（shadcn 组件），在 `gve init` 时自动安装
