---
name: gve
description: GVE（Go + Vite + Embed）全栈项目脚手架使用指南。覆盖 gve CLI 全部命令（init、dev、build、run、ui add/sync/diff、api add/generate/new、sync、status、doctor、registry build）、Thrift IDL 编写规范、UI/API 资产管理、项目目录约定、前端样式规范和开发工作流。当用户提到 gve、wk-ui、wk-api、gve.lock、thrift、api generate、Go embed、单二进制部署、UI 资产安装或 API 契约管理时使用。即使用户只提到 "生成 API"、"写 thrift"、"添加组件" 等模糊表述，只要上下文涉及 GVE 生态也应触发。
---

# GVE 使用指南

**规范来源**：本 skill 的权威文档仅限本目录内文件（`SKILL.md`、`reference.md`、`thrift-spec.md`）。勿引用当前打开项目下的 `docs/` 或其它路径，否则在非 gve 仓库中会指向错误内容。

GVE 是一个 Go + Vite + embed 全栈脚手架，包含三个仓库：

| 仓库 | 职责 |
|------|------|
| `gve` | 单一 CLI 工具（Go 实现） |
| `wk-ui` (workstation-ui) | UI 资产库（组件包装 + 全局配置） |
| `wk-api` (workstation-api) | API 契约库（仅 Thrift IDL） |

## 命令速查

```bash
# 项目初始化
gve init <project-name>           # 生成 Go 骨架 + 拉取 base-setup 前端框架

# 日常开发
gve dev                           # 并发启动 Air (Go) + Vite (前端)
gve build                         # 构建单二进制（内嵌 site/dist/）
gve run                           # 后台运行（智能判断是否需要重新构建）
gve run stop | restart | status | logs

# UI 资产
gve ui add <asset>[@version]      # 安装 UI 资产
gve ui list                       # 列出已安装资产
gve ui diff <asset>               # 对比本地与资产库的差异
gve ui sync [asset]               # 升级到最新版（有本地改动时交互确认）

# API 契约
gve api add <project>/<resource>[@version]   # 安装 API 契约
gve api sync                                 # 同步更新
gve api new <project>/<resource> [version]   # 在项目内创建 thrift 骨架（默认 v1）
gve api generate                             # 从 thrift AST 生成 Go struct + HTTP client 到 internal/api，生成 TS client 到 site/src/api

# 协作与状态
gve sync                          # 按 gve.lock 还原所有资产（团队同步）
gve status                        # 显示所有资产的版本与可用更新
gve doctor                        # 检查环境（Go ≥1.22、Node ≥18、pnpm、Git、Air）

# 资产库维护（在 wk-ui 或 wk-api 目录执行）
gve registry build                # 扫描 assets/ 自动生成 registry.json
```

---

## 项目目录结构

```
{project}/
├── go.mod / go.sum
├── Makefile
├── gve.lock                          # 资产版本锁定文件（始终提交 Git）
├── .gitignore
├── .gve/                             # 运行时数据（不提交 Git）
│   ├── run.pid
│   └── logs/
│
├── cmd/server/
│   └── main.go                       # 只负责注册路由 + 启动 HTTP server
│
├── internal/                         # 业务实现（按需扩展）
│   ├── handler/                      # ★ HTTP 层：解析请求、调用 service、返回响应
│   ├── service/                      # ★ 业务逻辑层：不依赖 HTTP
│   ├── model/                        # 数据模型（可选，按需加）
│   └── repo/                         # 数据访问层（可选，按需加）
│
├── api/                              # ★ thrift 契约（由 gve api add/new 管理）
│   └── {project}/{resource}/v{N}/
│       ├── {resource}.thrift         # Thrift IDL
│
├── internal/api/                     # ★ gve api generate 生成的 Go 代码
│   └── {project}/{resource}/v{N}/
│       ├── {resource}.go
│       └── client.go
│
└── site/                             # 前端（base-setup 初始化）
    ├── embed.go                      # go:embed all:dist（不修改）
    ├── package.json / pnpm-lock.yaml
    ├── vite.config.ts / tsconfig.json / biome.json / index.html
    └── src/
        ├── app/                      # ★ 框架层，只放初始化代码，不放业务
        │   ├── main.tsx              # 入口：ReactDOM.createRoot，不写业务
        │   ├── routes.tsx            # 路由表
        │   ├── providers.tsx         # 全局 Provider 组合
        │   └── styles/globals.css    # CSS 变量 + Tailwind 入口
        │
        ├── views/                    # ★ 业务页面，按功能模块分子目录
        │   └── {feature}/
        │       ├── index.tsx         # 该功能页面入口（路由引用）
        │       └── components/       # 该功能私有组件（不跨 feature 复用）
        │
        ├── api/                      # ★ gve api generate 生成的 TS client
        │   └── {project}/{resource}/v{N}/client.ts
        └── shared/                   # ★ 跨 views 复用的代码
            ├── ui/                   # ★ UI 资产（gve ui add 安装，禁止手写组件）
            ├── lib/                  # 工具函数（cn.ts、request.ts 等）
            ├── hooks/                # 通用 React hooks（可选）
            └── types/                # 共享 TypeScript 类型（可选）
```

### 后端分层约定

| 层 | 目录 | 职责 | 禁止 |
|----|------|------|------|
| 入口 | `cmd/server/main.go` | 注册路由、启动 server | 写业务逻辑 |
| HTTP 层 | `internal/handler/` | 解析请求参数、调用 service、序列化响应 | 直接操作数据库 |
| 业务层 | `internal/service/` | 业务规则、数据校验、编排调用 | 依赖 `net/http` 类型 |
| 数据层 | `internal/repo/`（可选） | 数据库 / 外部服务调用 | 业务规则 |
| API 契约 | `api/` | thrift 源文件（共享契约） | 存放生成代码 |

**文件命名**：每个业务资源对应一个文件，如 `user_handler.go`、`user_service.go`。

### 前端目录约定

| 目录 | 职责 | 禁止 |
|------|------|------|
| `src/app/` | 框架初始化 | 写业务组件或逻辑 |
| `src/views/{feature}/` | 页面及其私有组件 | 跨 feature 互相 import |
| `src/shared/ui/` | UI 资产（gve 管理） | 手动在此目录创建组件 |
| `src/shared/lib/` | 通用工具函数 | 包含业务逻辑 |
| `src/shared/hooks/` | 通用 hooks（无业务状态） | 依赖特定 feature 的数据 |
| `src/shared/types/` | 跨模块 TS 类型 | 包含运行时逻辑 |

**依赖方向（单向）**：`views → shared`，`shared` 内部不互相依赖，`app` 只引用 `views` 和 `shared`。

---

## 常用工作流

### 初始化新项目
```bash
gve init my-app
cd my-app
gve ui add button
gve api add ai-worker/task@v1
cd site && pnpm install && cd ..
gve dev
```

### 团队协作
```bash
git pull
gve sync          # 按 gve.lock 还原所有缺失资产
```

### 升级资产
```bash
gve status                        # 查看哪些资产有更新
gve ui sync button                # 升级（有本地改动时提示 overwrite/keep/diff）
git add gve.lock site/src/shared/ui/button
git commit -m "chore: upgrade button to v1.3.0"
```

### 发布 UI 资产新版本（在 wk-ui 仓库）
1. 创建 `assets/{name}/v{x.y.z}/` 目录
2. 编写 `meta.json` + 资产文件
3. 执行 `gve registry build` 更新 registry.json
4. `git add . && git commit`

### 发布 API 契约新版本（在 wk-api 仓库）
1. 在 `{project}/{resource}/v{N}/` 创建或修改文件
2. 手动更新 `registry.json`（或执行 `gve registry build`）
3. `git add . && git commit`

---

## gve.lock 格式

```json
{
  "version": "1",
  "ui": {
    "registry": "git.local/workstation/ui",
    "assets": {
      "button": { "version": "1.2.0" },
      "base-setup": { "version": "1.0.0" }
    }
  },
  "api": {
    "registry": "git.local/workstation/api",
    "assets": {
      "ai-worker/task": { "version": "v1" }
    }
  }
}
```

**规则**：`gve.lock` 始终提交 Git；`.gve/` 目录不提交。

---

## 前端样式规范（硬性约束）

**三层隔离：Tailwind 优先 + CSS Modules 兜底 + `cn()` 合并**

```tsx
// 简单组件 — 纯 Tailwind
import { cn } from '@/shared/lib/cn'
export const Button = ({ className, ...props }) =>
  <button className={cn("px-4 py-2 bg-primary text-white rounded", className)} {...props} />
```

```tsx
// 复杂组件 — Tailwind + CSS Modules
import styles from './chart.module.css'
export const Chart = ({ className }) =>
  <div className={cn(styles.root, "w-full", className)} />
```

**禁止**：
- 全局裸选择器（`.title { }`）
- 手写 `.css` 文件（只允许 `.module.css`）
- CSS-in-JS（Emotion / styled-components）

---

## UI 资产规范（wk-ui 维护者）

**meta.json 五字段：**

```json
{
  "name": "data-table",
  "version": "2.0.0",
  "dest": "",
  "deps": ["@tanstack/react-table"],
  "files": ["data-table.tsx", "data-table.module.css"]
}
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `name` | 是 | 与目录名一致 |
| `version` | 是 | semver |
| `dest` | 否 | **有 dest = 全局资产**（安装到指定路径）；**无 dest = 组件**（安装到 `site/src/shared/ui/{name}/`） |
| `deps` | 否 | npm 依赖，`gve ui add` 时自动写入项目 `package.json` |
| `files` | 是 | 需复制的文件列表（不含 meta.json） |

---

## 详细参考（仅以本 skill 目录内文档为准）

**约定**：GVE 规范与参考仅以本 skill 所在目录中的文件为准，勿引用当前项目下的 `docs/` 或其它外部路径。

- 完整架构、项目目录规范、工作流与 gve.lock：见本目录 [reference.md](reference.md)
- wk-ui / wk-api 结构、registry.json、API 四文件规范、base-setup 内容：见本目录 [reference.md](reference.md)
- Thrift IDL 编写规范、支持类型、Service 定义、完整示例：见本目录 [thrift-spec.md](thrift-spec.md)

## 可选扩展

- 若项目需要在 Go 进程内集成 PocketBase（单二进制 + 内置认证 + SPA 服务），可选使用独立 skill：`go-pocketbase-integration`。
- 该扩展不是 gve 默认能力；使用时应遵循本 skill 的项目结构与命令工作流约束。
