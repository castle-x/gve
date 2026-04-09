# GVE v2：UI 资产架构重设计

**日期**：2026-03-17
**状态**：已确认，待实施
**影响范围**：gve CLI、wk-ui 仓库结构、项目目录约定、gve.lock 格式、meta.json 协议

---

## 目录

1. [变更总览](#1-变更总览)
2. [wk-ui 仓库结构重设计](#2-wk-ui-仓库结构重设计)
3. [registry.json 格式升级](#3-registryjson-格式升级)
4. [meta.json 协议升级](#4-metajson-协议升级)
5. [gve.lock 格式升级](#5-gvelock-格式升级)
6. [GVE CLI 命令行为变更](#6-gve-cli-命令行为变更)
7. [项目安装路径映射](#7-项目安装路径映射)
8. [样式约束](#8-样式约束)
9. [迁移计划](#9-迁移计划)
10. [验收标准](#10-验收标准)

---

## 1. 变更总览

### 1.1 核心变更点

| 维度 | v1（当前） | v2（目标） |
|------|-----------|-----------|
| wk-ui 仓库结构 | `assets/` 扁平目录 | `scaffold/` + `ui/` + `components/` 三模块 |
| 骨架名称 | `base-setup` | `scaffold/{name}`（如 `scaffold/default`） |
| registry.json key | `"button"` | `"ui/button"`（带 category 前缀） |
| meta.json 字段 | 5 字段 | 9 字段（新增 `$schema`、`category`、`description`、`peerDeps`） |
| gve.lock key | `"button"` | `"ui/button"`（带 category 前缀） |
| gve.lock version | `"1"` | `"2"` |
| 项目安装路径 (ui) | `shared/ui/{name}/` | `shared/wk/ui/{name}/` |
| 项目安装路径 (component) | `shared/ui/{name}/` | `shared/wk/components/{name}/` |
| shadcn 安装路径 | `shared/ui/` | `shared/shadcn/` |
| CSS 文件 | 允许 `.module.css` | **禁止**，纯 `.tsx` + Tailwind |
| `gve ui search` | 不存在 | **暂不实现**（预留 `description` 字段） |

### 1.2 不变的部分

- wk-api 仓库结构和工作流不变
- `gve api` 系列命令不变
- Thrift IDL 规范不变
- `gve dev` / `gve build` / `gve run` 不变
- `gve registry build` 基本逻辑不变（扫描目录变更）

---

## 2. wk-ui 仓库结构重设计

### 2.1 v1 结构（当前）

```
wk-ui/
├── registry.json
└── assets/
    ├── base-setup/
    │   └── v1.0.0/
    │       ├── meta.json          # dest: "site"
    │       └── ...
    ├── button/
    │   └── v1.0.0/
    │       ├── meta.json
    │       └── button.tsx
    ├── data-table/
    │   └── v2.0.0/
    │       ├── meta.json
    │       ├── data-table.tsx
    │       └── data-table.module.css   ← v2 不再允许
    └── theme/
        └── v1.0.0/
            ├── meta.json          # dest: "site/src/app/styles"
            └── globals.css
```

### 2.2 v2 结构（目标）

```
wk-ui/
├── registry.json                          # 统一索引（由 gve registry build 自动生成）
│
├── scaffold/                              # 模块1: 项目骨架
│   ├── default/                           # 默认骨架（React + Tailwind + Vite）
│   │   └── v1.0.0/
│   │       ├── meta.json                  # category: "scaffold", dest: "site"
│   │       ├── embed.go
│   │       ├── package.json
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
│   └── dashboard/                         # Dashboard 骨架（含侧栏 + 布局）
│       └── v1.0.0/
│           ├── meta.json                  # category: "scaffold", dest: "site"
│           └── ...
│
├── ui/                                    # 模块2: 自研 UI 原子组件
│   ├── spinner/
│   │   └── v1.0.0/
│   │       ├── meta.json                  # category: "ui"
│   │       └── spinner.tsx                # 纯 .tsx，无 .css
│   ├── input-group/
│   │   └── v1.0.0/
│   │       ├── meta.json
│   │       └── input-group.tsx
│   └── sub-nav/
│       └── v1.0.0/
│           ├── meta.json
│           └── sub-nav.tsx
│
├── components/                            # 模块3: 业务复杂组件
│   ├── data-table/
│   │   └── v2.0.0/
│   │       ├── meta.json                  # category: "component", peerDeps 声明
│   │       └── data-table.tsx             # 纯 .tsx，无 .css
│   └── file-tree/
│       └── v1.0.0/
│           ├── meta.json
│           └── file-tree.tsx
│
└── global/                                # 模块4: 全局配置资产（保留 dest 机制）
    └── theme/
        └── v1.0.0/
            ├── meta.json                  # category: "global", dest: "site/src/app/styles"
            └── globals.css
```

### 2.3 目录说明

| 顶层目录 | 职责 | 资产特征 |
|----------|------|----------|
| `scaffold/` | 项目骨架/脚手架 | 一次性安装（`gve init`），`dest: "site"` |
| `ui/` | 自研 UI 原子组件（纯展示，无业务逻辑） | 单 `.tsx` 文件，安装到 `shared/wk/ui/{name}/` |
| `components/` | 业务复杂组件（可包含逻辑、可依赖 ui/） | 单 `.tsx` 文件，安装到 `shared/wk/components/{name}/`，可声明 `peerDeps` |
| `global/` | 全局配置资产（CSS 变量、主题等） | 由 `dest` 字段指定安装路径 |

### 2.4 命名规范

- 资产目录名：kebab-case（如 `data-table`、`input-group`）
- 版本目录名：`v{semver}`（如 `v1.0.0`、`v2.1.0`）
- 资产文件名：与目录名一致（如 `spinner.tsx`）

---

## 3. registry.json 格式升级

### 3.1 v1 格式（当前）

```json
{
  "button": {
    "latest": "1.2.0",
    "versions": {
      "1.0.0": { "path": "assets/button/v1.0.0" },
      "1.2.0": { "path": "assets/button/v1.2.0" }
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

### 3.2 v2 格式（目标）

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

### 3.3 变更要点

1. **新增顶层字段**：`$schema`（可选）、`version: "2"`（必须）
2. **key 带 category 前缀**：`"ui/spinner"` 而非 `"spinner"`，key 与物理路径一致
3. **path 字段更新**：从 `"assets/button/v1.0.0"` 变为 `"ui/spinner/v1.0.0"`（与实际目录对应）

### 3.4 `gve registry build` 扫描逻辑变更

**v1**：扫描 `assets/*/v*/meta.json`

**v2**：扫描以下四个目录：

```
scaffold/*/v*/meta.json
ui/*/v*/meta.json
components/*/v*/meta.json
global/*/v*/meta.json
```

生成的 registry key = `{directory}/{asset-name}`（如 `ui/spinner`、`components/data-table`）。

---

## 4. meta.json 协议升级

### 4.1 v1 字段（当前，5 字段）

```json
{
  "name": "button",
  "version": "1.0.0",
  "dest": "",
  "deps": [],
  "files": ["button.tsx"]
}
```

### 4.2 v2 字段（目标，9 字段）

```json
{
  "$schema": "https://gve.dev/schema/meta.json",
  "name": "data-table",
  "version": "2.0.0",
  "category": "component",
  "description": "A full-featured data table with sorting, filtering, pagination.",
  "dest": "",
  "deps": ["@tanstack/react-table"],
  "peerDeps": ["ui/button", "ui/input-group", "ui/spinner"],
  "files": ["data-table.tsx"]
}
```

### 4.3 字段定义

| 字段 | 类型 | 必填 | 新增 | 说明 |
|------|------|------|------|------|
| `$schema` | string | 否 | ✅ | JSON Schema URL，IDE 自动补全和校验 |
| `name` | string | 是 | - | 资产名，必须与目录名一致 |
| `version` | string | 是 | - | semver 版本号 |
| `category` | enum | 否 | ✅ | `"scaffold"` \| `"ui"` \| `"component"` \| `"global"`。可省略，从所在顶层目录自动推导 |
| `description` | string | 否 | ✅ | 一句话描述资产用途。`gve ui list` 时展示，未来用于搜索 |
| `dest` | string | 否 | - | 有值 = 安装到指定路径（全局资产）；空字符串或缺失 = 按 category 决定安装位置 |
| `deps` | string[] | 否 | - | npm 依赖列表，`gve ui add` 时自动写入项目 `package.json` |
| `peerDeps` | string[] | 否 | ✅ | **wk-ui 内部的组件间依赖**。值为 registry key（如 `"ui/button"`）。安装时自动检测并提示安装缺失的依赖 |
| `files` | string[] | 是 | - | 需复制的文件列表（不含 meta.json 自身）。**v2 禁止包含 `.css` / `.module.css` 文件**（global/ 目录的 `.css` 除外） |

### 4.4 category 推导规则

如果 `category` 字段缺失或为空，GVE CLI 从文件物理路径推导：

| 所在目录 | 推导为 |
|----------|--------|
| `scaffold/` | `"scaffold"` |
| `ui/` | `"ui"` |
| `components/` | `"component"` |
| `global/` | `"global"` |

如果 `category` 字段存在但与目录不匹配，`gve registry build` 应报 warning。

### 4.5 peerDeps 解析策略

当执行 `gve ui add components/data-table` 时：

1. 读取 `data-table` 的 `meta.json`，获取 `peerDeps: ["ui/button", "ui/input-group", "ui/spinner"]`
2. 对照项目 `gve.lock`，检查每个 peerDep 是否已安装
3. 对于未安装的 peerDep，列出清单并**自动安装**（使用 latest 版本），无需用户逐一确认
4. 输出安装日志：

```
Installing components/data-table@2.0.0...
  → peerDep ui/button: already installed v1.2.0 ✓
  → peerDep ui/input-group: installing v1.0.0...
  → peerDep ui/spinner: installing v1.0.0...
✓ Installed components/data-table to shared/wk/components/data-table/
✓ gve.lock updated
```

5. 如果 peerDep 自身也有 peerDeps，递归解析（需检测循环依赖，最大深度 5 层）

### 4.6 meta.json 各 category 示例

#### scaffold（骨架）

```json
{
  "name": "default",
  "version": "1.0.0",
  "category": "scaffold",
  "description": "Minimal React + Tailwind + Vite scaffold.",
  "dest": "site",
  "deps": [],
  "files": [
    "embed.go",
    "package.json",
    "vite.config.ts",
    "tsconfig.json",
    "biome.json",
    "index.html",
    ".gitignore",
    "src/app/main.tsx",
    "src/app/routes.tsx",
    "src/app/providers.tsx",
    "src/app/styles/globals.css",
    "src/shared/lib/cn.ts"
  ]
}
```

#### ui（UI 原子）

```json
{
  "name": "spinner",
  "version": "1.0.0",
  "category": "ui",
  "description": "Animated loading spinner with size variants.",
  "deps": [],
  "files": ["spinner.tsx"]
}
```

#### component（业务组件）

```json
{
  "name": "data-table",
  "version": "2.0.0",
  "category": "component",
  "description": "Data table with sort, filter, pagination, row selection.",
  "deps": ["@tanstack/react-table"],
  "peerDeps": ["ui/button", "ui/input-group", "ui/spinner"],
  "files": ["data-table.tsx"]
}
```

#### global（全局配置）

```json
{
  "name": "theme",
  "version": "1.0.0",
  "category": "global",
  "description": "OKLCH design tokens and Tailwind theme configuration.",
  "dest": "site/src/app/styles",
  "deps": [],
  "files": ["globals.css"]
}
```

---

## 5. gve.lock 格式升级

### 5.1 v1 格式（当前）

```json
{
  "version": "1",
  "ui": {
    "registry": "https://github.com/castle-x/wk-ui.git",
    "assets": {
      "button": { "version": "1.2.0" },
      "base-setup": { "version": "1.0.0" }
    }
  },
  "api": {
    "registry": "https://github.com/castle-x/wk-api.git",
    "assets": {
      "nanomind/auth": { "version": "v1" }
    }
  }
}
```

### 5.2 v2 格式（目标）

```json
{
  "version": "2",
  "ui": {
    "registry": "https://github.com/castle-x/wk-ui.git",
    "assets": {
      "scaffold/default": { "version": "1.0.0" },
      "ui/spinner": { "version": "1.0.0" },
      "ui/input-group": { "version": "1.0.0" },
      "components/data-table": { "version": "2.0.0" },
      "global/theme": { "version": "1.0.0" }
    }
  },
  "api": {
    "registry": "https://github.com/castle-x/wk-api.git",
    "assets": {
      "nanomind/auth": { "version": "v1" }
    }
  }
}
```

### 5.3 变更要点

1. `version` 从 `"1"` 升级到 `"2"`
2. `ui.assets` 的 key 从 `"button"` 变为 `"ui/button"`，带 category 前缀
3. `"base-setup"` 变为 `"scaffold/default"`
4. api 部分结构不变

### 5.4 向后兼容

- GVE CLI v2 读取到 `version: "1"` 的 lock 文件时，应自动升级为 v2 格式（给旧 key 加上 `"ui/"` 前缀，`"base-setup"` 映射为 `"scaffold/default"`）
- 升级后在下一次 `gve ui add` / `gve ui sync` 时覆写 lock 文件为 v2 格式

---

## 6. GVE CLI 命令行为变更

### 6.1 `gve init`

**v1 行为**：硬编码安装 `base-setup` 资产

**v2 行为**：从 `scaffold/` 目录选择骨架

```bash
$ gve init my-app

? Select scaffold:
  ❯ default    — Minimal React + Tailwind + Vite
    dashboard  — Dashboard layout + Sidebar + Auth

Creating my-app...
✓ Go skeleton created
✓ scaffold/default@1.0.0 installed to site/
✓ gve.lock created
```

**实现要点**：
1. `gve init` 先拉取 wk-ui registry.json
2. 过滤出所有 `scaffold/*` 的资产
3. 如果只有一个 scaffold，直接安装；如果多个，交互式选择
4. 安装后写入 `gve.lock`，key 为 `"scaffold/default"`

### 6.2 `gve ui add`

**v1 行为**：`gve ui add button` → 在 `assets/` 中找 `button` → 安装到 `shared/ui/button/`

**v2 行为**：支持带 category 前缀的安装

```bash
# 完整写法
gve ui add ui/spinner           # 安装到 shared/wk/ui/spinner/
gve ui add ui/spinner@1.0.0     # 指定版本
gve ui add components/data-table # 安装到 shared/wk/components/data-table/

# 简写（自动查找）
gve ui add spinner              # 按优先级搜索：ui/ → components/ → global/
gve ui add data-table           # 找到 components/data-table
```

**简写解析优先级**：

1. 精确匹配 `ui/{name}`
2. 精确匹配 `components/{name}`
3. 精确匹配 `global/{name}`
4. 找不到 → 报错 `Asset "{name}" not found in registry. Try: gve ui add ui/{name} or components/{name}`

**安装目标路径**：

| category | dest 缺失时安装到 | 示例 |
|----------|-----------------|------|
| `scaffold` | 由 `dest` 字段指定（通常 `site/`） | `site/package.json`, `site/src/app/...` |
| `ui` | `site/src/shared/wk/ui/{name}/` | `shared/wk/ui/spinner/spinner.tsx` |
| `component` | `site/src/shared/wk/components/{name}/` | `shared/wk/components/data-table/data-table.tsx` |
| `global` | 由 `dest` 字段指定 | `site/src/app/styles/globals.css` |

**peerDeps 自动解析**：见 [4.5 节](#45-peerdeps-解析策略)。

### 6.3 `gve ui list`

**v2 行为**：按 category 分组展示

```bash
$ gve ui list

SCAFFOLD
  scaffold/default         v1.0.0    site/

UI
  ui/spinner               v1.0.0    shared/wk/ui/spinner/
  ui/input-group           v1.0.0    shared/wk/ui/input-group/

COMPONENTS
  components/data-table    v2.0.0    shared/wk/components/data-table/

GLOBAL
  global/theme             v1.0.0    site/src/app/styles/
```

**实现要点**：
1. 从 `gve.lock` 读取已安装资产
2. 按 category 前缀分组
3. 展示 registry key、版本、安装路径

### 6.4 `gve ui diff`

**v2 行为**：增强 diff，支持交互式合并

```bash
$ gve ui diff ui/spinner

Comparing ui/spinner (local v1.0.0 vs registry v1.1.0):

--- local: site/src/shared/wk/ui/spinner/spinner.tsx
+++ registry: ui/spinner/v1.1.0/spinner.tsx

@@ -5,7 +5,9 @@
 export function Spinner({ size = "md", className }: SpinnerProps) {
+  const sizeClasses = {
+    sm: "h-4 w-4",
     md: "h-6 w-6",
+    lg: "h-8 w-8",
+  };
   return (
-    <div className={cn("animate-spin h-6 w-6", className)}>
+    <div className={cn("animate-spin", sizeClasses[size], className)}>

? Action: [u]pgrade  [k]eep  [m]erge  [s]kip
```

**四种操作**：

| 操作 | 说明 | 行为 |
|------|------|------|
| `upgrade` (u) | 直接覆盖为 registry 最新版 | 复制 registry 文件覆盖本地，更新 gve.lock |
| `keep` (k) | 保留本地版本 | 不做任何改动，但 gve.lock 版本更新为 registry 最新（标记为已审查） |
| `merge` (m) | 手动合并 | 生成 `.patch` 文件到 `.gve/patches/{asset}.patch`，用户手动应用 |
| `skip` (s) | 跳过 | 不做任何改动，gve.lock 也不更新 |

### 6.5 `gve ui sync`

**v2 行为**：批量升级，对有本地改动的组件特别标注

```bash
$ gve ui sync

Checking for updates...

  ui/spinner               1.0.0 → 1.1.0  (minor)
  ui/input-group           1.0.0 → 1.0.0  ✓ up to date
  components/data-table    2.0.0 → 2.1.0  (minor, has local changes!)

? Sync all? [Y/n/select]
```

**本地改动检测**：
1. 将本地文件与 `gve.lock` 中记录版本的 registry 源文件做 diff
2. 如果本地有改动（哈希不一致），标注 `has local changes!`
3. 有本地改动的资产，sync 时进入 `gve ui diff` 的交互式流程

### 6.6 `gve registry build`

**v2 行为**：扫描四个目录

```bash
# 在 wk-ui 仓库根目录执行
$ gve registry build

Scanning scaffold/...    2 assets found
Scanning ui/...          5 assets found
Scanning components/...  3 assets found
Scanning global/...      1 asset found

✓ registry.json updated (11 assets, 0 errors, 0 warnings)
```

**实现要点**：
1. 扫描目录从 `assets/*/v*/meta.json` 改为 `{scaffold,ui,components,global}/*/v*/meta.json`
2. registry key = `{category_dir}/{asset_name}`
3. 校验：`meta.json` 中 `category` 字段（如有）必须与所在目录一致
4. 校验：`meta.json` 中 `files` 列表不包含 `.css` / `.module.css`（`global/` 目录除外）
5. 输出 registry.json 格式为 v2（带 `$schema` 和 `version: "2"`）

### 6.7 `gve sync`

不受影响。按 `gve.lock` 还原所有资产，安装路径由 category 决定。

---

## 7. 项目安装路径映射

### 7.1 完整目录结构（v2）

```
site/src/shared/
├── shadcn/                  ← npx shadcn add 安装（shadcn/ui 组件，扁平存放）
│   ├── button.tsx
│   ├── dialog.tsx
│   ├── command.tsx
│   ├── sidebar.tsx
│   └── ...
│
├── wk/                      ← gve ui add 安装（自研资产）
│   ├── ui/                  ← category: "ui"
│   │   ├── spinner/
│   │   │   └── spinner.tsx
│   │   ├── input-group/
│   │   │   └── input-group.tsx
│   │   └── sub-nav/
│   │       └── sub-nav.tsx
│   └── components/          ← category: "component"
│       ├── data-table/
│       │   └── data-table.tsx
│       └── file-tree/
│           └── file-tree.tsx
│
├── lib/                     ← 工具函数
├── hooks/                   ← 通用 hooks & Zustand stores
├── types/                   ← 共享类型
└── docs/                    ← 文档站共享组件
```

### 7.2 shadcn 配置变更

`site/components.json` 需要更新组件安装目录：

```json
{
  "$schema": "https://ui.shadcn.com/schema.json",
  "style": "radix-nova",
  "rsc": false,
  "tsx": true,
  "tailwind": {
    "config": "",
    "css": "src/app/styles/globals.css",
    "baseColor": "neutral",
    "cssVariables": true,
    "prefix": ""
  },
  "aliases": {
    "components": "@/shared/shadcn",
    "utils": "@/shared/lib/utils",
    "ui": "@/shared/shadcn",
    "lib": "@/shared/lib",
    "hooks": "@/shared/hooks"
  },
  "iconLibrary": "lucide"
}
```

**关键变更**：
- `aliases.components`：从 `@/shared/ui` → `@/shared/shadcn`
- `aliases.ui`：从 `@/shared/ui` → `@/shared/shadcn`

### 7.3 Import 路径变更

```tsx
// v1
import { Button } from "@/shared/ui/button";
import { Spinner } from "@/shared/ui/wk/spinner/spinner";

// v2
import { Button } from "@/shared/shadcn/button";         // shadcn 组件
import { Spinner } from "@/shared/wk/ui/spinner/spinner"; // wk-ui 组件
import { DataTable } from "@/shared/wk/components/data-table/data-table"; // wk-ui 业务组件
```

### 7.4 三层组件来源对比

| 来源 | 管理工具 | 安装路径 | Import 路径前缀 | 是否可修改 |
|------|----------|----------|-----------------|-----------|
| shadcn/ui | `npx shadcn add` | `shared/shadcn/` | `@/shared/shadcn/` | 通过 className 扩展，不改源码 |
| wk-ui (UI 原子) | `gve ui add ui/xxx` | `shared/wk/ui/{name}/` | `@/shared/wk/ui/` | 可以，diff 追踪 |
| wk-ui (业务组件) | `gve ui add components/xxx` | `shared/wk/components/{name}/` | `@/shared/wk/components/` | 可以，diff 追踪 |

---

## 8. 样式约束

### 8.1 核心规则

**wk-ui 资产（ui/ 和 components/）禁止包含独立 CSS 文件。**

- ❌ 禁止 `.css`
- ❌ 禁止 `.module.css`
- ❌ 禁止 `.scss` / `.less`
- ✅ 所有样式必须通过 Tailwind 类名写在 `.tsx` 文件中
- ✅ 使用 `cn()` 合并类名
- ✅ 使用 `cva()` 管理多变体
- ✅ 使用 CSS 变量（通过 Tailwind token 引用）

**唯一例外**：`global/` 目录的资产（如 `global/theme`）可以包含 `.css` 文件（因其本质就是 CSS 配置）。

### 8.2 `gve registry build` 校验

在 v2 中，`gve registry build` 应对 `ui/` 和 `components/` 目录下的资产做以下校验：

1. `meta.json` 的 `files` 数组中不包含任何 `.css` 后缀的文件
2. 资产目录中不存在 `.css` 文件（即使未在 `files` 中声明）
3. 违反时输出 warning：`Warning: {asset} contains CSS files. ui/ and components/ assets should use Tailwind classes only.`

---

## 9. 迁移计划

### 9.1 wk-ui 仓库迁移步骤

#### Step 1: 创建新目录结构

```bash
cd wk-ui

# 创建四个顶层目录
mkdir -p scaffold ui components global
```

#### Step 2: 迁移现有资产

```bash
# base-setup → scaffold/default
mv assets/base-setup scaffold/default

# theme（全局配置）→ global/theme
mv assets/theme global/theme

# 纯 UI 组件 → ui/
mv assets/button ui/button
mv assets/spinner ui/spinner
# ... 其他 UI 原子

# 业务复杂组件 → components/
mv assets/data-table components/data-table
# ... 其他业务组件
```

#### Step 3: 更新所有 meta.json

为每个资产的 meta.json 添加新字段：

```bash
# 以 ui/spinner 为例
cat > ui/spinner/v1.0.0/meta.json << 'EOF'
{
  "$schema": "https://gve.dev/schema/meta.json",
  "name": "spinner",
  "version": "1.0.0",
  "category": "ui",
  "description": "Animated loading spinner with size variants.",
  "deps": [],
  "files": ["spinner.tsx"]
}
EOF
```

#### Step 4: 删除 .css 文件

对 `ui/` 和 `components/` 目录下所有资产：
1. 检查是否存在 `.css` / `.module.css` 文件
2. 将 CSS 中的样式迁移为 Tailwind 类名写入 `.tsx` 文件
3. 删除 `.css` 文件
4. 更新 `meta.json` 的 `files` 字段

#### Step 5: 添加 peerDeps

对 `components/` 目录下的资产，分析其 import 引用，添加 `peerDeps` 字段。

#### Step 6: 删除旧 assets/ 目录

```bash
rm -rf assets/
```

#### Step 7: 重新生成 registry.json

```bash
gve registry build
```

#### Step 8: 验证

```bash
# 确认 registry.json 格式正确
cat registry.json | jq .

# 确认所有 key 带 category 前缀
cat registry.json | jq 'keys'
```

### 9.2 GVE CLI 迁移步骤

#### Step 1: 更新 registry.json 解析逻辑

- 支持读取 v2 格式的 registry.json（带 `version: "2"`）
- 兼容读取 v1 格式（无 `version` 字段或 `version: "1"`）

#### Step 2: 更新 meta.json 解析逻辑

- 支持读取新增的 `category`、`description`、`peerDeps`、`$schema` 字段
- 缺失时使用默认值（category 从路径推导）

#### Step 3: 更新 `gve ui add` 安装路径逻辑

```go
func getInstallPath(category string, name string, dest string) string {
    if dest != "" {
        return dest // 全局资产，使用 dest 指定路径
    }
    switch category {
    case "scaffold":
        return "site" // scaffold 必须有 dest
    case "ui":
        return fmt.Sprintf("site/src/shared/wk/ui/%s", name)
    case "component":
        return fmt.Sprintf("site/src/shared/wk/components/%s", name)
    case "global":
        return "" // global 必须有 dest，缺失则报错
    default:
        return fmt.Sprintf("site/src/shared/wk/ui/%s", name) // 向后兼容
    }
}
```

#### Step 4: 实现 peerDeps 解析

```go
func resolvePeerDeps(meta Meta, lock LockFile, registry Registry) []string {
    var toInstall []string
    for _, dep := range meta.PeerDeps {
        if _, installed := lock.UI.Assets[dep]; !installed {
            toInstall = append(toInstall, dep)
        }
    }
    return toInstall
}
```

#### Step 5: 更新 `gve.lock` 读写逻辑

- 读取时检查 `version` 字段
- `version: "1"` → 自动升级为 v2 格式（key 加 `"ui/"` 前缀）
- 写入时始终使用 v2 格式

#### Step 6: 更新 `gve ui list` 输出

按 category 分组展示。

#### Step 7: 更新 `gve ui diff` 交互

增加 `merge` 和 `skip` 选项。

#### Step 8: 更新 `gve registry build` 扫描逻辑

扫描四个目录，生成带 category 前缀的 key。

#### Step 9: 更新 `gve init` 骨架选择

从 `scaffold/` 目录中列出可用骨架，交互式选择。

### 9.3 项目迁移步骤（使用 gve 的业务项目）

> 注意：这些步骤在业务项目升级到 gve v2 后执行。

#### Step 1: 移动 shadcn 组件

```bash
cd site/src/shared
mkdir -p shadcn
mv ui/*.tsx shadcn/        # 将 shadcn 组件移到 shadcn/ 目录（扁平）
```

#### Step 2: 创建 wk 目录

```bash
mkdir -p wk/ui wk/components
# 将原 shared/ui/ 中由 gve 管理的资产移到对应目录
```

#### Step 3: 更新 components.json

将 `aliases.ui` 和 `aliases.components` 改为 `@/shared/shadcn`。

#### Step 4: 全局替换 import 路径

```bash
# shadcn 组件
sed -i 's|@/shared/ui/|@/shared/shadcn/|g' $(find site/src -name "*.tsx" -o -name "*.ts")

# wk-ui 组件（如有）
# 手动调整 import 路径到 @/shared/wk/ui/ 或 @/shared/wk/components/
```

#### Step 5: 升级 gve.lock

运行 `gve sync`，自动将 gve.lock 升级为 v2 格式。

---

## 10. 验收标准

### 10.1 wk-ui 仓库

- [ ] 顶层目录为 `scaffold/`、`ui/`、`components/`、`global/`，旧 `assets/` 目录已删除
- [ ] 所有 meta.json 包含 `category` 字段（或可从路径推导）
- [ ] `ui/` 和 `components/` 目录下无 `.css` 文件
- [ ] `components/` 目录下的资产已声明 `peerDeps`
- [ ] `gve registry build` 生成 v2 格式的 registry.json
- [ ] registry.json 所有 key 带 category 前缀

### 10.2 GVE CLI

- [ ] `gve init` 支持从 `scaffold/` 选择骨架
- [ ] `gve ui add` 支持 `ui/xxx`、`components/xxx` 完整写法
- [ ] `gve ui add` 简写能正确解析（按优先级搜索）
- [ ] `gve ui add` 安装到正确的目标路径
- [ ] `gve ui add` 自动解析 peerDeps 并安装缺失依赖
- [ ] `gve ui list` 按 category 分组展示
- [ ] `gve ui diff` 支持 upgrade/keep/merge/skip 四种操作
- [ ] `gve ui sync` 检测本地改动并标注
- [ ] `gve.lock` 读写 v2 格式
- [ ] `gve.lock` v1 → v2 自动升级
- [ ] `gve registry build` 扫描四个目录
- [ ] `gve registry build` 校验 CSS 文件

### 10.3 业务项目

- [ ] shadcn 组件在 `shared/shadcn/` 扁平存放
- [ ] wk-ui 组件在 `shared/wk/ui/` 和 `shared/wk/components/`
- [ ] 所有 import 路径正确
- [ ] `components.json` 的 `aliases.ui` 指向 `@/shared/shadcn`
- [ ] `pnpm lint` + `pnpm typecheck` 通过
