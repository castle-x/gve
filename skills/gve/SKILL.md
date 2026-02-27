---
name: gve
description: GVE（Go + Vite + Embed）全栈项目脚手架使用指南。在 GVE 项目中工作时触发，包含 gve CLI 命令速查、UI/API 资产管理规范、项目目录约定、前端样式规范和工作流。当用户提到 gve、wk-ui、wk-api、gve.lock、gve init、gve dev、gve build、UI 资产安装或 API 契约管理时使用。
---

# GVE 使用指南

GVE 是一个 Go + Vite + embed 全栈脚手架，包含三个仓库：

| 仓库 | 职责 |
|------|------|
| `gve` | 单一 CLI 工具（Go 实现） |
| `wk-ui` (workstation-ui) | UI 资产库（组件包装 + 全局配置） |
| `wk-api` (workstation-api) | API 契约库（Thrift IDL + 生成代码） |

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
├── gve.lock                      # 资产版本锁定文件（提交 Git）
├── .gitignore
├── .gve/                         # 运行时数据（不提交 Git）
│   ├── run.pid
│   └── logs/
│
├── cmd/server/main.go            # Go 应用入口
├── internal/
│   ├── handler/                  # HTTP 处理层
│   └── service/                  # 业务逻辑层
│
├── api/                          # API 契约（gve api add 安装到此）
│   └── {project}/{resource}/{vN}/
│       ├── {resource}.thrift
│       ├── {resource}.go
│       ├── client.go
│       └── client.ts
│
└── site/                         # 前端
    ├── embed.go                  # go:embed all:dist
    ├── package.json / pnpm-lock.yaml
    ├── vite.config.ts / tsconfig.json / biome.json
    ├── index.html
    └── src/
        ├── app/                  # 框架层（routes、providers、styles）
        ├── views/                # 业务页面
        └── shared/ui/            # UI 资产安装目录（gve ui add → 此处）
```

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

## 详细参考

- 完整架构设计：见 `docs/2026-02-26-gve-design.md`
- wk-ui / wk-api 结构细节、registry.json 格式、API 文件规范：见 [reference.md](reference.md)
