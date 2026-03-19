---
name: gve
description: GVE（Go + Vite + Embed）全栈项目脚手架使用指南。覆盖 gve CLI 全部命令（init、dev、build、run、ui add/update/diff/push、api add/update/new/generate/push、sync、status、doctor、registry build）、Thrift IDL 编写规范、UI/API 资产管理、项目目录约定、前端样式规范和开发工作流。当用户提到 gve、wk-ui、wk-api、gve.lock、thrift、api generate、Go embed、单二进制部署、UI 资产安装或 API 契约管理时使用。即使用户只提到 "生成 API"、"写 thrift"、"添加组件" 等模糊表述，只要上下文涉及 GVE 生态也应触发。
---

# GVE 使用指南（v2）

**规范来源**：本 skill 的权威文档仅限本目录内文件（`SKILL.md`、`reference.md`、`thrift-spec.md`）。勿引用当前打开项目下的 `docs/` 或其它路径，否则在非 gve 仓库中会指向错误内容。

GVE 是一个 Go + Vite + embed 全栈脚手架，包含三个仓库：

| 仓库 | 职责 |
|------|------|
| `gve` | 单一 CLI 工具（Go 实现） |
| `wk-ui` (workstation-ui) | UI 资产库：`scaffold/`（骨架）+ `ui/`（原子组件）+ `components/`（业务组件）+ `global/`（全局配置） |
| `wk-api` (workstation-api) | API 契约库（仅 Thrift IDL） |

## 命令速查

```bash
# 项目初始化
gve init <project-name>           # 生成 Go 骨架 + scaffold 前端 + 自动 pnpm install
gve init <name> --scaffold dashboard  # 指定骨架模板（默认 default）

# 日常开发
gve dev                           # 并发启动 Air (Go) + Vite (前端)，自动检测 pnpm/npm，自动传递 VITE_BACKEND_TARGET
gve build                         # 构建单二进制（内嵌 site/dist/），支持 pnpm→npm 降级
gve run                           # 后台运行（智能判断是否需要重新构建）
gve run stop | restart | status | logs

# UI 资产（支持 category 前缀）
gve ui add <asset>[@version]      # 安装 UI 资产 + 递归解析 peerDeps + 自动 pnpm install
gve ui add <name>                 # 简写：自动搜索 ui/ → components/ → global/
gve ui list                       # 按 category 分组列出已安装资产（含 description）
gve ui diff <asset>               # 对比本地与资产库的差异（支持 upgrade/keep/merge/skip）
gve ui update [asset]             # 升级到最新版（有本地改动时交互确认）
gve ui push <name> [--version x.y.z] [--source dir] [--desc "..."] [--dry-run]
                                  # 扫描 TSX import，自动生成 meta.json，发布到 wk-ui registry

# API 契约
gve api add <project>/<resource>[@version]   # 安装 API 契约
gve api update                               # 升级 API 契约到最新版本
gve api new <project>/<resource> [version]   # 在项目内创建 thrift 骨架（默认 v1）
gve api generate                             # 从 thrift AST 生成 Go struct + HTTP client 到 internal/api，生成 TS client 到 site/src/api
gve api push <project>/<resource> [--version vN] [--source dir] [--dry-run]
                                             # 发布 thrift 文件到 wk-api registry（自动 git commit + push）

# 协作与状态
gve sync                          # 按 gve.lock 还原所有资产（自动跳过 scaffold 类型）
gve status                        # 显示所有资产的版本与可用更新
gve doctor                        # 检查环境（Go ≥1.22、Node ≥18、pnpm、Git、Air）

# 资产库维护（在 wk-ui 或 wk-api 目录执行）
gve registry build                # 扫描 scaffold/ui/components/global/ 自动生成 registry.json
                                  # 检测 meta.Files 中的 CSS + 目录中未声明的 CSS 文件
```

---

## wk-ui 仓库结构（v2）

```
wk-ui/
├── registry.json                       # 版本索引（由 gve registry build 自动生成）
│
├── scaffold/                           # 模块1: 项目骨架
│   ├── default/                        # 默认骨架（React + Tailwind + Vite）
│   │   └── v1.0.0/
│   │       ├── meta.json              # category: "scaffold", dest: "site"
│   │       └── ...
│   └── dashboard/                      # Dashboard 骨架（含侧栏 + 布局）
│       └── v1.0.0/
│
├── ui/                                 # 模块2: 自研 UI 原子组件
│   ├── spinner/
│   │   └── v1.0.0/
│   │       ├── meta.json
│   │       └── spinner.tsx            # 纯 .tsx，禁止 .css
│   └── input-group/
│       └── v1.0.0/
│
├── components/                         # 模块3: 业务复杂组件
│   └── data-table/
│       └── v2.0.0/
│           ├── meta.json              # 可声明 peerDeps
│           └── data-table.tsx
│
└── global/                             # 模块4: 全局配置资产
    └── theme/
        └── v1.0.0/
            ├── meta.json              # dest: "site/src/app/styles"
            └── globals.css            # 唯一允许 .css 的 category
```

### 四模块说明

| 目录 | 职责 | 安装时机 | 安装路径 |
|------|------|---------|---------|
| `scaffold/` | 项目骨架 | `gve init` 一次性 | 由 `dest` 指定（通常 `site/`） |
| `ui/` | 纯 UI 原子（无业务逻辑） | `gve ui add` 按需 | `shared/wk/ui/` |
| `components/` | 有业务逻辑的复合组件 | `gve ui add` 按需 | `shared/wk/components/` |
| `global/` | 全局配置（CSS 变量、主题等） | `gve ui add` 按需 | 由 `dest` 指定 |

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
└── site/                             # 前端（scaffold 初始化）
    ├── embed.go                      # go:embed all:dist（不修改）
    ├── package.json / pnpm-lock.yaml
    ├── app.json                  # 品牌配置（name、displayName）
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
        │
        └── shared/                   # ★ 跨 views 复用的代码
            ├── shadcn/               # ★ shadcn/ui 组件（npx shadcn add 安装，扁平存放）
            ├── wk/                   # ★ wk-ui 自研资产（gve ui add 安装）
            │   ├── ui/              # UI 原子组件
            │   └── components/      # 业务复杂组件
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
| `src/shared/shadcn/` | shadcn/ui 组件（扁平存放） | 直接修改源码 |
| `src/shared/wk/ui/` | wk-ui 原子组件 | 手动在此目录创建组件 |
| `src/shared/wk/components/` | wk-ui 业务组件 | 手动在此目录创建组件 |
| `src/shared/lib/` | 通用工具函数 | 包含业务逻辑 |
| `src/shared/hooks/` | 通用 hooks（无业务状态） | 依赖特定 feature 的数据 |
| `src/shared/types/` | 跨模块 TS 类型 | 包含运行时逻辑 |

**依赖方向（单向）**：`views → shared`，`shared` 内部不互相依赖，`app` 只引用 `views` 和 `shared`。

---

## 常用工作流

### 初始化新项目
```bash
gve init my-app                    # 一键创建：Go 骨架 + 前端 + pnpm install + shadcn + wk 组件
gve init my-app --scaffold dashboard  # 使用 dashboard 骨架
cd my-app
gve dev                            # 直接可运行，无需手动 pnpm install
```

`gve init` 完整流程：
1. 校验项目名 + 创建目录
2. 渲染 Go 后端骨架（go.mod、main.go、Makefile、.gitignore、gve.lock）
3. 拉取 wk-ui registry → 选择 scaffold → 复制前端文件 + placeholder 目录
4. 品牌名替换（`__PROJECT_NAME__` → 实际项目名，递归扫描 `site/src/` 下所有 `.ts/.tsx/.js/.jsx/.json/.html` 文件）
5. 自动 `pnpm install`（pnpm 不可用时降级 npm）
6. 安装 scaffold 声明的 shadcn 组件（`meta.shadcnDeps`，使用 `--overwrite` 避免交互提示）
7. 安装 scaffold 声明的 wk 默认组件（`meta.defaultAssets`）+ 递归 peerDeps
8. 再次 `pnpm install`（刷新新注入的 npm deps）
9. 更新 gve.lock

### 团队协作
```bash
git pull
gve sync          # 按 gve.lock 还原所有缺失资产（自动跳过 scaffold）
```

### 升级资产
```bash
gve status                        # 查看哪些资产有更新
gve ui update ui/spinner          # 升级（有本地改动时提示 upgrade/keep/merge/skip）
git add gve.lock site/src/shared/wk/ui/spinner.tsx
git commit -m "chore: upgrade ui/spinner to v1.1.0"
```

### 发布 UI 资产新版本（推荐：gve ui push）

```bash
# 在业务项目中，修改完组件后一键发布
gve ui push spinner --version 1.1.0 --desc "Spinner with CVA variants"

# 自动扫描 TSX import → 识别 npm deps + wk-ui peerDeps → 生成 meta.json → git commit + push
# 流程：定位源目录 → 扫描 import → 确定版本 → 构建 meta → 发布到 registry → 更新 gve.lock

# 不指定版本 → 自动从 gve.lock 当前版本 patch +1
gve ui push spinner

# 预览模式（不实际写入）
gve ui push data-table --dry-run

# 指定任意源目录
gve ui push my-comp --source ./custom/path/ --version 1.0.0
```

**TSX Import 扫描器**会自动：
- 识别 npm 依赖（如 `class-variance-authority`、`@tanstack/react-table`）→ 写入 meta.json `deps`
- 识别 wk-ui 内部组件依赖（如 `@/shared/wk/ui/spinner`）→ 写入 meta.json `peerDeps`
- 跳过相对路径、项目别名（`@/`）、静态资源（.css/.svg）、React 宿主依赖
- 自动去除注释中的 import（block comment + line comment）

### 发布 UI 资产（手动方式，在 wk-ui 仓库）

> 仅在无法使用 `gve ui push` 时使用。
1. 创建 `{category}/{name}/v{x.y.z}/` 目录（如 `ui/spinner/v1.1.0/`）
2. 编写 `meta.json` + 资产文件（仅 `.tsx`，禁止 `.css`）
3. 执行 `gve registry build` 更新 registry.json
4. `git add . && git commit`

### 发布 API 契约新版本（推荐：gve api push）

```bash
# 在业务项目中，编辑完 thrift 文件后一键发布
gve api push ai-worker/task                    # 自动检测版本（从源目录名推断）
gve api push ai-worker/task --version v2       # 指定版本
gve api push ai-worker/task --source ./mythrift/ --version v1  # 指定源目录
gve api push ai-worker/task --dry-run          # 预览模式
```

`gve api push` 完整流程：
1. **定位源目录**：`--source` 优先，否则查找 `api/{project}/{resource}/` 下最高版本目录
2. **确定版本号**：`--version` > 从源目录名推断
3. **验证 .thrift 文件**：源目录中至少有一个 .thrift 文件
4. **发布到 registry cache**：git pull → 版本冲突检查 → 复制 .thrift 文件 → 重建 registry.json → git commit + push
5. **更新 gve.lock**

### 发布 API 契约（手动方式，在 wk-api 仓库）

> 仅在无法使用 `gve api push` 时使用。
1. 在 `{project}/{resource}/v{N}/` 创建或修改文件
2. 手动更新 `registry.json`（或执行 `gve registry build`）
3. `git add . && git commit`

---

## gve.lock 格式（v2）

```json
{
  "version": "2",
  "ui": {
    "registry": "https://github.com/castle-x/wk-ui.git",
    "assets": {
      "scaffold/default": { "version": "1.0.0" },
      "ui/spinner": { "version": "1.0.0" },
      "components/data-table": { "version": "2.0.0" },
      "global/theme": { "version": "1.0.0" }
    }
  },
  "api": {
    "registry": "https://github.com/castle-x/wk-api.git",
    "assets": {
      "ai-worker/task": { "version": "v1" }
    }
  }
}
```

**规则**：
- `gve.lock` 始终提交 Git
- `.gve/` 目录不提交
- v2 key 带 category 前缀（如 `"ui/spinner"` 而非 `"spinner"`）
- `gve init` 直接生成 v2 格式（不兼容 v1）

---

## UI 资产规范（wk-ui 维护者）

**meta.json 十一字段：**

```json
{
  "$schema": "https://gve.dev/schema/meta.json",
  "name": "data-table",
  "version": "2.0.0",
  "category": "component",
  "description": "Data table with sort, filter, pagination, row selection.",
  "dest": "",
  "deps": ["@tanstack/react-table@^8.0.0"],
  "peerDeps": ["ui/button", "ui/input-group", "ui/spinner"],
  "files": ["data-table.tsx"],
  "defaultAssets": [],
  "shadcnDeps": []
}
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `$schema` | 否 | JSON Schema URL，IDE 自动补全 |
| `name` | 是 | 与目录名一致 |
| `version` | 是 | semver |
| `category` | 否 | `"scaffold"` \| `"ui"` \| `"component"` \| `"global"`。可省略（从目录推导） |
| `description` | 否 | 一句话描述，`gve ui list` 时展示 |
| `dest` | 否 | 有 dest = 全局资产；无 dest = 按 category 决定安装位置 |
| `deps` | 否 | npm 依赖（支持 `name@version` 格式，如 `"lucide-react@^0.300.0"`），`gve ui add` 时自动写入项目 `package.json` |
| `peerDeps` | 否 | wk-ui 内部组件间依赖（registry key，如 `"ui/button"`），安装时**递归**解析（BFS，最大深度 5） |
| `files` | 是 | 需复制的文件列表。**ui/ 和 components/ 禁止 .css 文件** |
| `defaultAssets` | 否 | **scaffold 专用**：骨架默认安装的 wk 组件 key 列表（如 `["ui/spinner"]`） |
| `shadcnDeps` | 否 | **scaffold 专用**：骨架依赖的 shadcn 组件名列表（如 `["button", "card", "sidebar"]`） |

### 安装路径映射

| category | 无 dest 时安装到 | 示例 |
|----------|-----------------|------|
| `scaffold` | 由 dest 指定（通常 `site/`） | `site/package.json`, `site/src/app/...` |
| `ui` | `site/src/shared/wk/ui/` | `shared/wk/ui/spinner.tsx` |
| `component` | `site/src/shared/wk/components/` | `shared/wk/components/data-table.tsx` |
| `global` | 由 dest 指定 | `site/src/app/styles/globals.css` |

### 样式硬约束

- `ui/` 和 `components/` 资产 = 纯 `.tsx` 文件，**禁止附带 .css / .module.css**
- 所有样式必须通过 Tailwind 类名写在 `.tsx` 中
- 使用 `cn()` 合并类名，使用 `cva()` 管理变体
- 唯一例外：`global/` 目录可包含 `.css`（其本质就是 CSS 配置）

---

## 前端样式规范（硬性约束）

**纯 Tailwind，禁止独立 CSS 文件**

```tsx
// 简单组件 — 纯 Tailwind
import { cn } from '@/shared/lib/cn'
export const Button = ({ className, ...props }) =>
  <button className={cn("px-4 py-2 bg-primary text-white rounded", className)} {...props} />
```

```tsx
// 复杂组件 — Tailwind + cva 变体
import { cva, type VariantProps } from "class-variance-authority"
const spinnerVariants = cva("animate-spin", {
  variants: {
    size: { sm: "h-4 w-4", md: "h-6 w-6", lg: "h-8 w-8" },
  },
  defaultVariants: { size: "md" },
})
```

**禁止**：
- 全局裸选择器（`.title { }`）
- 独立 `.css` 文件（包括 `.module.css`）
- CSS-in-JS（Emotion / styled-components）

**唯一全局 CSS 文件**：`site/src/app/styles/globals.css`（CSS 变量 + Tailwind 入口）

---

## 详细参考（仅以本 skill 目录内文档为准）

**约定**：GVE 规范与参考仅以本 skill 所在目录中的文件为准，勿引用当前项目下的 `docs/` 或其它外部路径。

- 完整架构、项目目录规范、工作流与 gve.lock：见本目录 [reference.md](reference.md)
- wk-ui / wk-api 结构、registry.json、API 四文件规范、scaffold 内容：见本目录 [reference.md](reference.md)
- Thrift IDL 编写规范、支持类型、Service 定义、完整示例：见本目录 [thrift-spec.md](thrift-spec.md)

## 可选扩展

- 若项目需要在 Go 进程内集成 PocketBase（单二进制 + 内置认证 + SPA 服务），可选使用独立 skill：`go-pocketbase-integration`。
- 该扩展不是 gve 默认能力；使用时应遵循本 skill 的项目结构与命令工作流约束。
