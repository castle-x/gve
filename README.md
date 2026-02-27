# gve

**GVE** (Go + Vite + Embed) 是一个全栈项目脚手架 CLI，用于管理 Go 后端 + Vite 前端的一体化项目，并通过 Go `embed` 将前端打包进单一二进制文件。

## 功能

- `gve init` — 初始化项目（Go 骨架 + 前端框架）
- `gve dev` — 并发启动 Go 后端（Air 热重载）和 Vite 开发服务器
- `gve build` — 构建嵌入前端的单二进制文件，支持交叉编译
- `gve run` — 后台运行服务，支持 stop / restart / status / logs
- `gve ui add/sync/diff/list` — UI 组件资产管理（基于 wk-ui）
- `gve api add/sync` — API 契约管理（基于 wk-api）
- `gve sync` — 团队协作：按 gve.lock 还原所有资产
- `gve status` — 查看资产版本与可用更新
- `gve doctor` — 检查开发环境依赖

## 安装

**环境要求**：Go 1.22+

```bash
go install github.com/castle-x/gve/cmd/gve@latest
```

验证安装：

```bash
gve version
gve doctor
```

## 快速上手

```bash
# 初始化新项目
gve init my-app
cd my-app

# 安装前端依赖
cd site && pnpm install && cd ..

# 启动开发服务器
gve dev

# 安装 UI 组件
gve ui add button

# 安装 API 契约
gve api add example-project/user@v1

# 构建单二进制
gve build
```

## 配套资产库

| 仓库 | 说明 |
|------|------|
| [castle-x/wk-ui](https://github.com/castle-x/wk-ui) | UI 组件资产库（React + Tailwind） |
| [castle-x/wk-api](https://github.com/castle-x/wk-api) | API 契约库（Thrift + 生成代码） |

## Cursor Skill

本仓库附带 GVE 使用指南 Skill，安装后 Cursor Agent 能自动掌握 GVE 的命令、目录约定和工作流：

```bash
cp -r skills/gve ~/.cursor/skills/gve
```

## 从源码构建

```bash
git clone git@github.com:castle-x/gve.git
cd gve
make install   # 编译并安装到 $GOPATH/bin
make test      # 运行测试
```
