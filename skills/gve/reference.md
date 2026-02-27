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

## 2. wk-api 资产库完整结构

```
wk-api/
├── registry.json
│
├── ai-console/
│   └── user/
│       ├── v1/
│       │   ├── user.thrift         # Thrift IDL
│       │   ├── user.go             # Go 结构体（thrift-to-go 生成）
│       │   ├── client.go           # Go HTTP Client
│       │   └── client.ts           # TypeScript fetch Client
│       └── v2/                     # 破坏性变更才升大版本
│           ├── user.thrift
│           ├── user.go
│           ├── client.go
│           └── client.ts
│
└── ai-worker/
    └── task/
        └── v1/
            ├── task.thrift
            ├── task.go
            ├── client.go
            └── client.ts
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

# 编写四个文件：new-service.thrift / new-service.go / client.go / client.ts

# 更新 registry.json（手动编辑或 gve registry build）

git add ai-worker/new-service/ registry.json
git commit -m "feat(api): add ai-worker/new-service v1"
```

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
