---
name: gve-frontend-i18n
description: GVE 前端国际化技能，用于处理 React 项目中的多语言文本管理。当需要添加、修改或翻译 UI 文本时，必须使用此技能。适用场景包括：新增 UI 组件时添加文本、修改现有文本、扩展语言支持、创建新的翻译命名空间。触发关键词：i18n、国际化、翻译、多语言、t()、useTranslation。
---

# GVE 前端国际化技能

本技能提供 GVE 全栈项目前端国际化的完整工作流程和规范指导。

## 技术栈

| 组件 | 技术 | 说明 |
|------|------|------|
| i18n 核心 | `i18next` + `react-i18next` | React 生态最成熟方案 |
| 状态管理 | `Zustand` + `persist` | 与项目状态管理一致 |
| 持久化 | `localStorage` | key: `{project}-locale` |
| 默认语言 | `zh-CN` | 简体中文优先 |
| 支持语言 | `zh-CN`、`en` | 可扩展 |

## 目录结构

```
site/src/
├── i18n/
│   ├── index.ts              # i18next 初始化配置
│   ├── types.ts              # TypeScript 类型增强
│   └── locales/
│       ├── zh-CN/            # 简体中文
│       │   ├── common.json   # 通用文本（按钮、状态）
│       │   ├── auth.json     # 认证模块
│       │   ├── dashboard.json # 仪表盘模块
│       │   └── theme.json    # 主题相关
│       └── en/               # 英文
│           ├── common.json
│           ├── auth.json
│           ├── dashboard.json
│           └── theme.json
└── shared/
    ├── hooks/
    │   └── use-locale.ts     # Zustand 语言状态管理
    └── shadcn/
        └── locale-switcher.tsx # 语言切换组件
```

## 工作流程

### 1. 添加新 UI 文本

当创建新组件或功能需要 UI 文本时：

**Step 1**: 确定命名空间
- 通用文本（按钮、状态）→ `common`
- 认证相关 → `auth`
- 仪表盘/主界面 → `dashboard`
- 主题/配色 → `theme`
- 编辑器 → `editor`（如需要可新建）

**Step 2**: 添加翻译 key（中文先行）

```json
// locales/zh-CN/{namespace}.json
{
  "moduleName": {
    "action": "操作名称",
    "description": "描述文本"
  }
}
```

**Step 3**: 补充英文翻译

```json
// locales/en/{namespace}.json
{
  "moduleName": {
    "action": "Action Name",
    "description": "Description text"
  }
}
```

**Step 4**: 组件中使用

```tsx
import { useTranslation } from "react-i18next";

function MyComponent() {
  const { t } = useTranslation("namespace");
  
  return <span>{t("moduleName.action")}</span>;
}
```

### 2. 翻译 Key 命名规范

```
{namespace}.{module}.{usage}
```

| 层级 | 说明 | 示例 |
|------|------|------|
| namespace | 功能域 | `auth`, `dashboard`, `theme` |
| module | 子模块 | `login`, `user`, `accent` |
| usage | 具体用途 | `title`, `placeholder`, `error` |

**示例**：
- `auth.login.title` → "登录到你的工作空间"
- `auth.changePassword.minLength` → "密码至少 8 个字符"
- `theme.accent.blue` → "蓝色"
- `dashboard.user.logout` → "退出登录"

### 3. 插值语法

支持动态值注入：

```json
{
  "greeting": "你好，{{name}}！",
  "itemCount": "共 {{count}} 项"
}
```

```tsx
t("greeting", { name: "张三" })  // → "你好，张三！"
t("itemCount", { count: 5 })     // → "共 5 项"
```

### 4. 跨命名空间引用

```tsx
// 方式1：多命名空间 hook
const { t } = useTranslation(["auth", "common"]);
t("auth:login.title");
t("common:button.confirm");

// 方式2：直接使用 i18n.t()
import { i18n } from "@/i18n";
i18n.t("common:button.confirm");
```

### 5. 新增语言支持

若需添加新语言（如 `zh-TW`）：

1. 创建 `locales/zh-TW/` 目录
2. 复制所有 JSON 文件并翻译
3. 更新 `i18n/index.ts` resources 配置
4. 更新 `use-locale.ts` 的 `SUPPORTED_LOCALES`
5. 更新 `locale-switcher.tsx` 的 `LOCALE_LABELS`

### 6. 新增命名空间

若需添加新功能模块（如 `settings`）：

1. 创建 `locales/zh-CN/settings.json` 和 `locales/en/settings.json`
2. 在 `i18n/index.ts` 中添加导入和注册
3. 更新 `ns: [...]` 数组

## 核心代码参考

详细代码示例和配置请查阅 [references/implementation.md](./references/implementation.md)。

## 规则检查清单

添加 UI 文本前，确认以下事项：

- [ ] 确定正确的命名空间
- [ ] Key 命名遵循 `module.usage` 格式
- [ ] 同时添加 zh-CN 和 en 翻译
- [ ] 组件中使用 `useTranslation()` hook
- [ ] 没有硬编码的 UI 文本字符串

## 禁止事项

| ❌ 禁止 | ✅ 正确做法 |
|---------|------------|
| 硬编码中文 `"登录"` | `t("auth.login.submit")` |
| 仅添加中文翻译 | 同时添加 zh-CN 和 en |
| 在组件外直接调用 `t()` | 使用 `i18n.t()` 或在组件内使用 hook |
| Key 使用纯英文描述 `"loginButton"` | 使用层级结构 `"login.submit"` |
| 重复定义相同文本 | 提取到 `common` 命名空间复用 |
