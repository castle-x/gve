# GVE CLI 迭代开发方案

**日期**: 2026-03-18
**版本**: v2 新版（不考虑 v1 兼容与迁移）
**范围**: 新功能开发 + Bug 修复 + 功能优化

---

## 一、现状总结

### 已完成的核心功能

| 功能 | 文件位置 | 状态 |
|------|---------|------|
| `gve init` 基础骨架 | `init.go` + `init_frontend.go` | ✅ Go 后端 scaffold + 前端骨架复制 + placeholder 目录 + gve.lock 更新 |
| `gve ui add` v2 | `ui_add.go` + `installer.go` | ✅ category 前缀路径 + 简写解析 + 安装到 `shared/wk/{ui,components}/` + peerDeps 一层解析 + npm deps 注入 |
| `gve ui list` | `ui_list.go` | ✅ 按 category 分组展示 |
| `gve ui diff` | `ui_diff.go` + `differ.go` | ✅ 只读 diff 展示（M/D/A 状态 + unified diff） |
| `gve ui sync` | `ui_sync.go` | ✅ 本地改动检测 + u/d/k/s 交互选项 |
| `gve ui push` | `ui_push.go` + `publisher.go` + `scanner.go` | ✅ TSX import 扫描 → meta.json 生成 → 版本推算 → git pull→copy→registry rebuild→commit→push |
| `gve registry build` | `registry_build.go` + `builder.go` | ✅ v2 四目录扫描 + CSS 文件校验 + category mismatch 校验 |
| `gve dev` | `dev.go` | ✅ Go + Vite 并发启动 + Air 热重载 + 自动 pnpm install |
| `gve build` | `build.go` | ✅ pnpm install→pnpm build→go build + 交叉编译 |
| `gve run` 全套 | `run.go` | ✅ start/stop/restart/status/logs + PID 管理 + 端口检测 + logrotate |
| `gve api` 全套 | `api_*.go` + `thrift_gen.go` | ✅ add/sync/new/generate 完整 Thrift 代码生成链 |
| `gve sync` | `sync.go` | ✅ 按 gve.lock 还原全部 UI + API 资产 |
| `gve status` | `status.go` | ✅ 对比 gve.lock 与 registry 展示版本状态 |
| `gve doctor` | `doctor.go` | ✅ Go/Node/pnpm/Git/Air/GOPATH/gve.lock 环境检查 |
| Meta struct v2 | `meta.go` | ✅ 9 字段（$schema/name/version/category/description/dest/deps/peerDeps/files） |
| registry.json v2 | `registry.go` + `builder.go` | ✅ $schema + version:"2" + v1/v2 双格式解析 |
| gve.lock v2 | `lock.go` | ✅ Version:"2" + category 前缀 key |

---

## 二、迭代任务清单（按优先级排列）

### Phase 1：`gve init` "一键可运行"（P0 — 最高优先级）

> **目标**：`gve init my-app && cd my-app && gve dev` 一气呵成，零手动步骤。
>
> **当前问题**：`gve init` 执行后只复制了骨架文件和创建了 placeholder 目录，用户必须手动执行 `pnpm install`、手动安装 shadcn 组件、手动安装 wk 默认组件，否则 `gve dev` 直接报错。

#### Task 1.1：Meta struct 新增 scaffold 专用字段

**文件**: `internal/asset/meta.go`

在 `Meta` struct 中新增两个 scaffold 专用字段：

```go
type Meta struct {
    Schema      string   `json:"$schema,omitempty"`
    Name        string   `json:"name"`
    Version     string   `json:"version"`
    Category    string   `json:"category,omitempty"`
    Description string   `json:"description,omitempty"`
    Dest        string   `json:"dest,omitempty"`
    Deps        []string `json:"deps,omitempty"`
    PeerDeps    []string `json:"peerDeps,omitempty"`
    Files       []string `json:"files"`
    // ── scaffold 专用字段 ──
    DefaultAssets []string `json:"defaultAssets,omitempty"` // 骨架默认安装的 wk 组件 key 列表
    ShadcnDeps   []string `json:"shadcnDeps,omitempty"`    // 骨架依赖的 shadcn 组件名列表
}
```

- `defaultAssets`：骨架模板依赖的 wk-ui 组件，如 `["ui/theme-provider", "components/settings-dropdown"]`
- `shadcnDeps`：骨架模板依赖的 shadcn 组件，如 `["button", "card", "dialog", "sidebar", ...]`

**影响范围**：所有读取 meta.json 的地方不受影响（新字段 `omitempty` + 向后兼容），仅 `gve init` 流程消费这两个字段。

**工作量**：0.5h

---

#### Task 1.2：提取通用的包管理器检测与执行工具函数

**新文件**: `internal/cmd/pkg_manager.go`（或合并到已有工具文件中）

提取一个通用的 Node 包管理器检测执行函数，供 `gve init`、`gve dev`、`gve build`、`gve ui add` 等多处复用：

```go
// runNodeInstall 在指定目录执行 npm/pnpm install
// 优先使用 pnpm，pnpm 不存在或执行失败时降级为 npm，两者都失败则返回错误。
func runNodeInstall(dir string) error {
    // 1. 检测 pnpm 是否可用（exec.LookPath("pnpm")）
    // 2. 可用 → 执行 pnpm install
    //    2a. 执行成功 → return nil
    //    2b. 执行失败（exit code != 0） → 打印警告，继续尝试 npm
    // 3. pnpm 不可用 或 pnpm 失败 → 检测 npm
    // 4. npm 可用 → 执行 npm install
    //    4a. 执行成功 → return nil
    //    4b. 执行失败 → return error
    // 5. npm 也不可用 → return 带安装指引的错误信息
}
```

**降级策略细节**：

| 步骤 | 条件 | 行为 |
|------|------|------|
| 1 | `exec.LookPath("pnpm")` 成功 | 执行 `pnpm install` |
| 2 | pnpm install 成功 | 返回 nil |
| 3 | pnpm 不存在 **或** pnpm install 失败 | 打印 `⚠ pnpm not available or failed, falling back to npm...` |
| 4 | `exec.LookPath("npm")` 成功 | 执行 `npm install` |
| 5 | npm install 成功 | 返回 nil |
| 6 | npm 也不存在或也失败 | 返回 `fmt.Errorf("neither pnpm nor npm is available or working. Please install pnpm (npm install -g pnpm) or npm (https://nodejs.org)")` |

**同步修改点**：

1. `dev.go` 第 54-63 行（当前硬编码 `pnpm install`）→ 改为调用 `runNodeInstall(siteDir)`
2. `build.go` 第 60-72 行（当前硬编码 `pnpm install`）→ 改为调用 `runNodeInstall(siteDir)`
3. 后续 Task 1.4、1.6 中也会调用此函数

**工作量**：1h

---

#### Task 1.3：`gve init` 品牌名替换

**文件**: `internal/cmd/init_frontend.go`

在 scaffold 文件复制完成后，对以下文件做项目名替换：

**替换规则**：

| 文件 | 搜索 | 替换为 |
|------|------|--------|
| `site/package.json` | `"name": "{scaffold中的默认名}"` | `"name": "{projectName}"` |
| `site/index.html` | `<title>{默认标题}</title>` | `<title>{projectName}</title>` |
| `go.mod` | `module {scaffold默认模块名}` | `module {projectName}` |

**实现方式**：

1. 在 `Meta` struct 中已有 `Name` 字段，scaffold 的 meta.json 中写明模板默认名
2. 使用 `strings.ReplaceAll` 或 `bytes.Replace` 对文件内容做全文替换
3. 提取一个 `replaceBrandName(projectDir, oldName, newName string) error` 函数

**注意事项**：

- 只替换明确的占位符，避免误替换代码中的同名字符串
- scaffold 的 meta.json 中可以约定使用一个明确的占位符，如 `__PROJECT_NAME__`，更安全
- 需要处理替换后写回文件

**工作量**：1h

---

#### Task 1.4：`gve init` 执行 pnpm/npm install

**文件**: `internal/cmd/init_frontend.go` 或 `internal/cmd/init.go`

在 `runInit` 函数中、品牌名替换之后、shadcn 安装之前，执行 npm 依赖安装：

```
// 当前流程（已有）：
// 1. scaffold Go 后端
// 2. 复制前端骨架文件 + placeholder 目录

// 新增步骤（本 Task）：
// 3. 品牌名替换 (Task 1.3)
// 4. 执行 pnpm install / npm install (本 Task)
```

调用 Task 1.2 提取的 `runNodeInstall(siteDir)` 函数。

**输出示例**：

```
Creating project my-app...
  Generating Go backend skeleton...
  Initializing frontend from scaffold/default...
  Replacing project name...
  Installing npm dependencies...
    → pnpm install ✓
```

**工作量**：0.5h（已有工具函数，此处仅调用）

---

#### Task 1.5：`gve init` 安装 shadcn 组件

**文件**: `internal/cmd/init_frontend.go`

在 pnpm install 之后，读取 scaffold meta.json 的 `shadcnDeps` 字段，执行 shadcn 组件安装：

```go
// 5. 安装 shadcn 组件
if len(meta.ShadcnDeps) > 0 {
    fmt.Printf("  Installing shadcn components: %s\n", strings.Join(meta.ShadcnDeps, ", "))
    // 执行：npx shadcn@latest add <components...> --yes --silent
    // shadcn CLI 支持一次传多个组件名
    args := append([]string{"shadcn@latest", "add"}, meta.ShadcnDeps...)
    args = append(args, "--yes")
    cmd := exec.Command("npx", args...)
    cmd.Dir = siteDir
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("shadcn install failed: %w", err)
    }
}
```

**注意事项**：

- `npx` 需要 node_modules 已安装（依赖 Task 1.4 先完成）
- 需要 `components.json` 已存在于 scaffold 骨架中（由 scaffold 资产提供）
- `--yes` 跳过确认提示，适合自动化场景
- 如果 `npx` 不可用，尝试 `pnpm dlx shadcn@latest add ...` 作为降级方案

**工作量**：1.5h

---

#### Task 1.6：`gve init` 安装 defaultAssets（wk 组件）

**文件**: `internal/cmd/init_frontend.go`

在 shadcn 组件安装之后，读取 scaffold meta.json 的 `defaultAssets` 字段，依次安装 wk 组件：

```go
// 6. 安装默认 wk 组件
if len(meta.DefaultAssets) > 0 {
    fmt.Println("  Installing default UI assets...")
    for _, assetKey := range meta.DefaultAssets {
        fmt.Printf("    → %s\n", assetKey)
        ver, err := asset.InstallUIAsset(mgr, assetKey, "", projectDir)
        if err != nil {
            fmt.Printf("    ✗ Failed: %v\n", err)
            continue // 不中断整体 init 流程
        }
        lf.SetUIAsset(assetKey, ver)
        fmt.Printf("    ✓ Installed %s@%s\n", assetKey, ver)
    }
    // 保存更新后的 gve.lock
    if err := lf.Save(lockPath); err != nil {
        return fmt.Errorf("save gve.lock: %w", err)
    }
}
```

**注意事项**：

- defaultAssets 中的组件可能有 `peerDeps`，需要也解析安装（复用 `ui_add.go` 中的 peerDeps 解析逻辑）
- 建议将 peerDeps 解析逻辑从 `ui_add.go` 的 `runUIAdd` 中提取为独立函数（如 `installWithPeerDeps`），供 init 和 ui add 共用
- 单个 defaultAsset 安装失败不应中断整体 init 流程（打印警告继续）
- defaultAssets 安装可能注入新的 npm deps 到 package.json，需要在全部安装完后再执行一次 `runNodeInstall`

**工作量**：2h

---

#### Task 1.7：`gve init` 最终 pnpm install

**文件**: `internal/cmd/init_frontend.go`

在 defaultAssets 安装完成后（如果 defaultAssets 注入了新的 npm deps），需要再执行一次 npm 依赖安装：

```go
// 7. 如果 defaultAssets 注入了新 deps，再次安装
if len(meta.DefaultAssets) > 0 {
    fmt.Println("  Refreshing npm dependencies...")
    if err := runNodeInstall(siteDir); err != nil {
        return fmt.Errorf("final npm install: %w", err)
    }
}
```

**优化**：可以检查 `package.json` 是否实际发生了变更来决定是否需要再次 install，避免无意义的重复安装。

**工作量**：0.5h

---

#### Task 1.8：`gve init` 交互式 scaffold 选择

**文件**: `internal/cmd/init.go`

当 registry 中有多个 scaffold 时，提供交互式选择：

```go
// 在 initFrontend 调用前
scaffolds := reg.ListByCategory("scaffold")
if len(scaffolds) == 0 {
    return fmt.Errorf("no scaffolds available in registry")
}
if len(scaffolds) == 1 {
    scaffoldKey = scaffolds[0]
} else if scaffoldName == "default" { // 用户未显式指定 --scaffold
    // 交互式选择
    fmt.Println("Available scaffolds:")
    for i, s := range scaffolds {
        // 读取每个 scaffold 的 meta.json 获取 description
        desc := getScaffoldDescription(mgr, reg, s)
        fmt.Printf("  [%d] %s — %s\n", i+1, s, desc)
    }
    fmt.Print("Select scaffold [1]: ")
    // 读取用户输入...
    scaffoldKey = scaffolds[selected-1]
}
```

**实现方式**：

- 简单方案：使用 `bufio.NewReader(os.Stdin)` 读取数字输入（与 `ui_sync.go` 的交互风格一致）
- 进阶方案：引入 `charmbracelet/huh` 或 `AlecAivazis/survey` 库实现更美观的 TUI 选择器
- **建议**：先用简单方案（`bufio.Reader`），后续需求明确后再升级为 TUI 库

**展示信息**：每个 scaffold 展示 name + description（来自 meta.json）

**工作量**：1.5h

---

#### `gve init` 完整新流程总结

```
gve init my-app [--scaffold dashboard]
│
├── 1. 校验项目名 + 创建项目目录
├── 2. Scaffold Go 后端骨架（template.Scaffold）
├── 3. 拉取/更新 UI registry 缓存
├── 4. 选择 scaffold（交互式或 --scaffold 指定）    ← [Task 1.8]
├── 5. 复制前端骨架文件 + placeholder 目录（已有）
├── 6. 品牌名替换（package.json/index.html）        ← [Task 1.3]
├── 7. pnpm install（降级 npm install）              ← [Task 1.2 + 1.4]
├── 8. npx shadcn@latest add [shadcnDeps...]         ← [Task 1.5]
├── 9. 安装 defaultAssets (wk 组件 + peerDeps)       ← [Task 1.6]
├── 10. 再次 pnpm install（刷新新注入的 deps）       ← [Task 1.7]
├── 11. 更新 gve.lock 并保存
└── 12. 输出成功信息 + Next steps
```

**Phase 1 总工作量**：约 8.5h

---

### Phase 2：peerDeps 递归解析（P1）

> **目标**：`gve ui add` 安装组件时，递归解析 peerDeps 链，确保所有间接依赖都被自动安装。

#### Task 2.1：peerDeps 递归解析 + 循环检测

**文件**: `internal/asset/installer.go`（新增函数）+ `internal/cmd/ui_add.go`（调用方修改）

**当前问题**：`ui_add.go` 第 79-97 行只解析一层 peerDeps。如果组件 A 依赖 B、B 依赖 C，安装 A 时不会自动安装 C。

**实现方案**：

```go
// ResolvePeerDepsRecursive 递归解析 peerDeps，返回需要安装的完整列表（拓扑排序）。
// maxDepth 限制递归深度（建议 5），防止异常循环。
func ResolvePeerDepsRecursive(
    mgr *Manager,
    rootAsset string,
    installed map[string]bool,
    maxDepth int,
) ([]string, error) {
    // 1. BFS/DFS 遍历 peerDeps 链
    // 2. 维护 visited set 做循环检测
    // 3. 超过 maxDepth 报错：circular or too-deep dependency chain
    // 4. 返回拓扑排序后的安装顺序（叶子节点先装）
}
```

**关键设计**：

| 要素 | 方案 |
|------|------|
| 遍历算法 | BFS（广度优先），确保先安装浅层依赖 |
| 循环检测 | `visited` map，重复访问同一节点时跳过（不报错，因为钻石依赖是合法的） |
| 深度限制 | 最大 5 层，超出时打印警告并停止 |
| 返回格式 | `[]string`（去重、拓扑排序后的 registry key 列表） |
| 容错 | 某个 peerDep 在 registry 中找不到时打印警告，不中断整体安装 |

**修改 `ui_add.go`**：将第 79-97 行的一层 peerDeps 逻辑替换为调用 `ResolvePeerDepsRecursive`。

**提取公共函数**：供 `gve init` 的 defaultAssets 安装也使用。

```go
// installAssetWithPeerDeps 安装资产并递归安装其 peerDeps
// 返回 (已安装的 key→version map, error)
func installAssetWithPeerDeps(mgr *Manager, name, version, projectDir string, lf *lock.LockFile) error
```

**工作量**：3h

**单元测试**：

- 测试线性链 A→B→C 全部安装
- 测试钻石依赖 A→B→D, A→C→D（D 只装一次）
- 测试循环依赖 A→B→A（不死循环，打印警告）
- 测试超深链（>5 层）的截断行为

---

### Phase 3：`gve ui diff` 交互式操作（P2）

> **目标**：`gve ui diff` 不仅展示 diff，还提供操作选项让用户决定如何处理差异。

#### Task 3.1：`gve ui diff` 增加交互式操作

**文件**: `internal/cmd/ui_diff.go`

**当前行为**：只读展示 diff（M/D/A 状态 + unified diff 文本），展示完就退出。

**目标行为**：展示 diff 后，对每个有差异的文件提供四种操作选项：

| 操作 | 快捷键 | 说明 |
|------|--------|------|
| Upgrade | `u` | 用 registry 最新版覆盖本地文件 |
| Keep | `k` | 保留本地文件不动（gve.lock 可选标记为 `reviewed: true`） |
| Merge | `m` | 生成 `.gve/patches/{asset}/{file}.patch` 文件，用户后续手动 `git apply` |
| Skip | `s` | 跳过此文件，不做任何操作 |

**实现细节**：

1. **新增 `--interactive` / `-i` flag**（默认 false，保持只读 diff 的后向兼容性）
2. 交互模式下，对每个 modified 文件展示 diff 后提问 `[u]pgrade / [k]eep / [m]erge / [s]kip:`
3. upgrade 操作：调用 `CopyAsset` 覆盖单个文件
4. merge 操作：
   - 创建 `.gve/patches/{assetName}/` 目录
   - 将 unified diff 内容写入 `{filename}.patch`
   - 打印提示：`Patch saved to .gve/patches/{asset}/{file}.patch — apply with: git apply .gve/patches/{asset}/{file}.patch`
5. keep 操作：可在 gve.lock 的 AssetEntry 中新增可选字段 `reviewed`（标记用户已知晓本地有改动）

**新增目录约定**：`.gve/patches/` — 存放 merge 操作生成的 patch 文件

**工作量**：3h

---

#### Task 3.2：`gve ui sync` 增加 merge 操作

**文件**: `internal/cmd/ui_sync.go`

**当前行为**：交互选项是 `u/d/k/s`（upgrade/diff/keep/skip）

**目标**：增加 `m` (merge) 选项，与 `ui diff --interactive` 的 merge 逻辑一致。

修改 `promptSyncAction` 函数，增加 `[m]erge` 选项：

```
Options:
  [u] upgrade  — discard local changes, install new version
  [d] diff     — show diff, then decide
  [m] merge    — generate .patch file for manual merge
  [k] keep     — skip this asset
  [s] skip     — skip this asset
```

merge 操作的实现：对该 asset 的所有 modified 文件生成 `.gve/patches/{asset}/{file}.patch`。

**工作量**：1.5h

---

### Phase 4：功能优化（P3）

#### Task 4.1：`gve ui list` 显示 description

**文件**: `internal/cmd/ui_list.go`

**当前行为**：`ui_list.go` 展示 `name + version + path`

**目标**：额外展示 meta.json 中的 `description` 字段。

修改 `resolveDestPath` 改为 `resolveAssetInfo` 返回更多信息（dest + description），在输出中展示：

```
UI
  ui/spinner                    v1.0.0     site/src/shared/wk/ui/spinner/
    Loading spinner with customizable size and color
  ui/theme-provider             v1.2.0     site/src/shared/wk/ui/theme-provider/
    Dark/light theme context provider
```

**工作量**：0.5h

---

#### Task 4.2：`gve ui add` 后自动 pnpm install

**文件**: `internal/cmd/ui_add.go`

**当前问题**：`InstallUIAsset` 通过 `injectDeps` 向 `package.json` 注入 npm 依赖后，不会自动执行 `pnpm install`。如果 `gve dev` 已经运行中（`node_modules` 存在），新 deps 不会自动生效。

**目标**：当 `injectDeps` 实际注入了新依赖时，自动执行 `runNodeInstall`。

**实现**：

1. 修改 `injectDeps` 返回 `(changed bool, err error)` 而非仅 `error`
2. 在 `InstallUIAsset` 返回时透传 `depsChanged` 信息（可通过返回值或在 installer 层面暴露）
3. 在 `runUIAdd` 末尾，如果有新 deps 被注入，执行 `runNodeInstall`

**方案 A（简单）**：不改 `InstallUIAsset` 签名，在 `runUIAdd` 中通过对比 `package.json` 的 `mtime` 判断是否有变更。

**方案 B（精确）**：让 `InstallUIAsset` 返回一个 `InstallResult` struct，包含 `InstalledVersion` + `DepsInjected bool`。

**推荐方案 B**，更干净。

**工作量**：1.5h

---

#### Task 4.3：`gve registry build` 扫描目录中未声明的 CSS 文件

**文件**: `internal/asset/builder.go`

**当前行为**：`BuildRegistryV2` 第 137-145 行只检查 `meta.Files` 中声明的 `.css` 文件。

**目标**：额外扫描资产目录中实际存在但未声明在 `meta.Files` 中的 `.css` 文件，给出警告。

**实现**：在现有 CSS 检查逻辑之后，增加一段目录遍历：

```go
// 扫描目录中未声明的 .css 文件
actualFiles, _ := collectAllFiles(versionDir) // 递归列出目录下所有文件
declared := make(map[string]bool)
for _, f := range meta.Files {
    declared[f] = true
}
for _, f := range actualFiles {
    if strings.HasSuffix(f, ".css") && !declared[f] {
        warnings = append(warnings, fmt.Sprintf(
            "Warning: %s has undeclared CSS file %q in directory", registryKey, f))
    }
}
```

**工作量**：0.5h

---

#### Task 4.4：`gve dev` 的包管理器降级支持

**文件**: `internal/cmd/dev.go`

**当前问题**：`dev.go` 第 54-63 行硬编码使用 `pnpm install`；第 79 行 Vite 启动也硬编码 `pnpm dev`。如果用户没有 pnpm，直接报错。

**目标**：

1. install 步骤：使用 Task 1.2 的 `runNodeInstall` 替换
2. Vite 启动步骤：检测可用的包管理器，选择 `pnpm dev` 或 `npm run dev`

**实现**：提取一个 `detectPackageManager(siteDir string) string` 函数，返回 `"pnpm"` 或 `"npm"`：

```go
func detectPackageManager(siteDir string) string {
    // 1. 如果存在 pnpm-lock.yaml → 返回 "pnpm"
    // 2. 如果存在 package-lock.json → 返回 "npm"
    // 3. 如果 pnpm 可用（exec.LookPath）→ 返回 "pnpm"
    // 4. 如果 npm 可用 → 返回 "npm"
    // 5. 默认返回 "pnpm"（留给后续执行时报错）
}
```

**修改点**：

- `dev.go` 第 79 行：`viteOpts.Name` 改为 `detectPackageManager(siteDir)`
- `build.go` 第 75 行：`pnpm build` 改为 `{pm} build`

**工作量**：1h

---

### Phase 5：Bug 修复（穿插在各 Phase 中执行）

#### Bug 5.1：`gve ui diff` 使用简易 diff 算法，大文件 diff 结果不准确

**文件**: `internal/asset/differ.go`

**问题**：`unifiedDiff` 函数（第 81-110 行）使用极简的逐行对比算法，不是真正的 LCS（最长公共子序列）diff。对于中间插入/删除行的场景，输出结果不准确，可能将大量行标记为 `-`/`+` 而非正确的 context 行。

**修复方案**：

- **方案 A（轻量）**：引入 `github.com/sergi/go-diff/diffmatchpatch` 或 `github.com/pmezard/go-difflib` 实现标准 unified diff
- **方案 B（零依赖）**：实现 Myers diff 算法（较复杂，约 150-200 行代码）
- **推荐方案 A**：引入 `github.com/pmezard/go-difflib`，这是 Go 生态最常用的 diff 库，API 简洁

```go
import "github.com/pmezard/go-difflib/difflib"

func unifiedDiff(filename, original, modified string) string {
    diff := difflib.UnifiedDiff{
        A:        difflib.SplitLines(original),
        B:        difflib.SplitLines(modified),
        FromFile: "a/" + filename,
        ToFile:   "b/" + filename,
        Context:  3,
    }
    text, _ := difflib.GetUnifiedDiffString(diff)
    return text
}
```

**工作量**：1h

---

#### Bug 5.2：`gve sync` 不处理 scaffold 类型资产的特殊性

**文件**: `internal/cmd/sync.go`

**问题**：`gve sync` 遍历 `lf.UI.Assets` 安装所有资产，但 scaffold 类型（`scaffold/default`）的 `dest` 是整个 `site/` 目录。执行 `InstallUIAsset` 会覆盖整个 `site/` 目录内容，可能破坏用户的业务代码。

**修复方案**：在 `runSync` 中跳过 scaffold 类型的资产：

```go
for name, entry := range lf.UI.Assets {
    // scaffold 类型资产不参与 sync（它只在 init 时使用）
    if strings.HasPrefix(name, "scaffold/") {
        continue
    }
    // ... 正常安装逻辑
}
```

**工作量**：0.5h

---

#### Bug 5.3：`gve ui sync` 对 scaffold 资产也尝试升级

**文件**: `internal/cmd/ui_sync.go`

**问题**：与 Bug 5.2 同理，`ui sync` 遍历所有资产时也会试图升级 scaffold 资产。

**修复方案**：在 `targets` 遍历中跳过 scaffold 类型。

**工作量**：0.5h

---

#### Bug 5.4：`injectDeps` 写入 `"latest"` 作为版本号

**文件**: `internal/asset/installer.go`

**问题**：第 130 行 `depsMap[dep] = "latest"`。直接写 `"latest"` 到 package.json 的 dependencies 中不规范，pnpm install 时可能每次都重新解析版本，且不锁定版本。

**优化方案**：

- **方案 A（简单）**：改为 `"*"` 或 `"^latest"`（语义更明确）
- **方案 B（精确）**：让 meta.json 的 `deps` 字段支持 `name@version` 格式（如 `["lucide-react@^0.300.0", "sonner@^1.0.0"]`），解析后注入精确版本
- **推荐方案 B**：更精确，且对 scaffold 骨架和组件都有用

修改 `deps` 字段解析：

```go
// "lucide-react@^0.300.0" → name="lucide-react", version="^0.300.0"
// "sonner" → name="sonner", version="latest"
func parseDep(dep string) (name, version string) {
    if idx := strings.LastIndex(dep, "@"); idx > 0 {
        return dep[:idx], dep[idx+1:]
    }
    return dep, "latest"
}
```

**工作量**：1h

---

## 三、完整优先级排列与工时估算

| 序号 | Task | Phase | 优先级 | 工时 | 依赖 |
|------|------|-------|--------|------|------|
| 1 | Meta struct 新增 `defaultAssets` + `shadcnDeps` | P1 | P0 | 0.5h | 无 |
| 2 | 提取 `runNodeInstall` 通用函数（pnpm→npm 降级） | P1 | P0 | 1h | 无 |
| 3 | `gve init` 品牌名替换 | P1 | P0 | 1h | 无 |
| 4 | `gve init` 执行 pnpm/npm install | P1 | P0 | 0.5h | Task 2 |
| 5 | `gve init` 安装 shadcn 组件 | P1 | P0 | 1.5h | Task 1, 4 |
| 6 | `gve init` 安装 defaultAssets | P1 | P0 | 2h | Task 1, 4 |
| 7 | `gve init` 最终 pnpm install | P1 | P0 | 0.5h | Task 2, 6 |
| 8 | `gve init` 交互式 scaffold 选择 | P1 | P0 | 1.5h | 无 |
| 9 | `gve sync` 跳过 scaffold 资产 (Bug 5.2) | P5 | P0 | 0.5h | 无 |
| 10 | `gve ui sync` 跳过 scaffold 资产 (Bug 5.3) | P5 | P0 | 0.5h | 无 |
| 11 | peerDeps 递归解析 + 循环检测 | P2 | P1 | 3h | 无 |
| 12 | `gve ui diff` 交互式操作 | P3 | P2 | 3h | 无 |
| 13 | `gve ui sync` 增加 merge 操作 | P3 | P2 | 1.5h | Task 12 |
| 14 | `gve ui diff` 引入标准 diff 算法 (Bug 5.1) | P5 | P2 | 1h | 无 |
| 15 | `gve ui list` 显示 description | P4 | P3 | 0.5h | 无 |
| 16 | `gve ui add` 后自动 pnpm install | P4 | P3 | 1.5h | Task 2 |
| 17 | `gve dev`/`gve build` 包管理器降级支持 | P4 | P3 | 1h | Task 2 |
| 18 | `gve registry build` 扫描未声明 CSS 文件 | P4 | P3 | 0.5h | 无 |
| 19 | `injectDeps` 支持精确版本号 (Bug 5.4) | P5 | P3 | 1h | 无 |

**总工时估算**：约 22h

---

## 四、实施路线图

```
Week 1 — Phase 1: "gve init 一键可运行"
├── Day 1: Task 1-4 (Meta 字段 + runNodeInstall + 品牌名替换 + init install)
├── Day 2: Task 5-7 (shadcn 安装 + defaultAssets + 最终 install)
├── Day 3: Task 8-10 (交互式 scaffold + Bug fixes)
└── Day 3: 集成测试——验证 gve init → gve dev 全流程

Week 2 — Phase 2+3: "健壮性 + 交互增强"
├── Day 4: Task 11 (peerDeps 递归) + Task 14 (diff 算法修复)
├── Day 5: Task 12-13 (ui diff/sync 交互操作)
└── Day 5: 测试 peerDeps 递归 + diff/sync 交互

Week 3 — Phase 4+5: "体验优化 + Bug 修复"
├── Day 6: Task 15-18 (list description + auto install + dev 降级 + registry CSS)
├── Day 6: Task 19 (injectDeps 精确版本)
└── Day 6: 全功能回归测试
```

---

## 五、测试策略

### 单元测试（`*_test.go`）

| 测试点 | 覆盖范围 |
|--------|---------|
| `runNodeInstall` | pnpm 可用、pnpm 不可用降级 npm、两者都不可用 |
| `replaceBrandName` | package.json/index.html 替换正确性 |
| `ResolvePeerDepsRecursive` | 线性链、钻石依赖、循环检测、深度限制 |
| `parseDep` | `name@version` 解析、无版本默认 `latest` |
| `detectPackageManager` | lockfile 检测、命令可用性检测 |

### 集成测试（`user_journey_test.go` 扩展）

| 场景 | 验证点 |
|------|--------|
| `gve init` 全流程 | 骨架完整、package.json 项目名正确、node_modules 存在、shadcn 组件已安装 |
| `gve ui add` 递归 peerDeps | 安装 A，B 和 C（A 的间接 peerDep）都被安装 |
| `gve ui diff --interactive` | upgrade/keep/merge 三种操作都正确执行 |
| `gve sync` 跳过 scaffold | scaffold 资产不被覆盖 |

---

## 六、注意事项

1. **scaffold 资产的 meta.json 需要配套更新**：新增 `defaultAssets` 和 `shadcnDeps` 字段后，需要在 wk-ui registry 中的 scaffold 资产里填写这些值。这是方案生效的前提。

2. **`runNodeInstall` 的日志要清晰**：降级发生时必须打印明确的警告信息，让用户知道实际使用了哪个包管理器。

3. **`gve init` 失败回滚**：如果 init 流程中间失败（如 shadcn 安装失败），应考虑是否清理已创建的项目目录。建议：品牌名替换和文件复制失败时回滚，但 npm install / shadcn install 失败时不回滚（允许用户手动重试）。

4. **`gve ui push` 的 meta 生成需感知新字段**：scaffold 类型的 push 需要支持 `defaultAssets` 和 `shadcnDeps` 字段的编辑/生成。当前 `ui_push.go` 的 meta 构建逻辑只设置基础 7 个字段，需要为 scaffold 类型增加这两个字段的处理。

5. **向后兼容**：所有 meta.json 新字段都使用 `omitempty`，不影响非 scaffold 类型资产的读写。
