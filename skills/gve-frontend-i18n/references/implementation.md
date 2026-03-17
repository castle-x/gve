# i18n 实现参考

本文档提供 GVE 前端国际化的完整代码实现参考。

## 1. 核心配置文件

### i18n/index.ts

i18next 初始化配置：

```typescript
import i18n from "i18next";
import { initReactI18next } from "react-i18next";

// 导入所有翻译文件
import authEn from "./locales/en/auth.json";
import commonEn from "./locales/en/common.json";
import dashboardEn from "./locales/en/dashboard.json";
import themeEn from "./locales/en/theme.json";
import authZh from "./locales/zh-CN/auth.json";
import commonZh from "./locales/zh-CN/common.json";
import dashboardZh from "./locales/zh-CN/dashboard.json";
import themeZh from "./locales/zh-CN/theme.json";

const resources = {
  "zh-CN": {
    common: commonZh,
    auth: authZh,
    dashboard: dashboardZh,
    theme: themeZh,
  },
  en: {
    common: commonEn,
    auth: authEn,
    dashboard: dashboardEn,
    theme: themeEn,
  },
} as const;

i18n.use(initReactI18next).init({
  resources,
  lng: "zh-CN", // 默认语言（会被 Zustand store 覆盖）
  fallbackLng: "zh-CN",
  ns: ["common", "auth", "dashboard", "theme"],
  defaultNS: "common",
  interpolation: {
    escapeValue: false, // React already escapes
  },
});

export { i18n };
export type { resources };
```

### i18n/types.ts

TypeScript 类型增强（提供自动补全）：

```typescript
import "i18next";
import type { resources } from "./index";

declare module "i18next" {
  interface CustomTypeOptions {
    defaultNS: "common";
    resources: (typeof resources)["zh-CN"];
  }
}
```

## 2. 状态管理

### shared/hooks/use-locale.ts

Zustand store 管理语言状态：

```typescript
import { useEffect } from "react";
import { create } from "zustand";
import { persist } from "zustand/middleware";
import { i18n } from "@/i18n";

export const SUPPORTED_LOCALES = ["zh-CN", "en"] as const;
export type Locale = (typeof SUPPORTED_LOCALES)[number];

interface LocaleState {
  locale: Locale;
  setLocale: (locale: Locale) => void;
}

export const useLocale = create<LocaleState>()(
  persist(
    (set) => ({
      locale: "zh-CN",
      setLocale: (locale) => set({ locale }),
    }),
    { name: "nanomind-locale" }, // localStorage key
  ),
);

/**
 * Sync locale state to i18next.
 * Must be called once in a top-level component (e.g. App or Providers).
 */
export function useLocaleSync() {
  const { locale } = useLocale();

  useEffect(() => {
    i18n.changeLanguage(locale);
  }, [locale]);
}
```

## 3. Provider 集成

### app/providers.tsx

在根 Provider 中初始化 i18n：

```tsx
import "@/i18n"; // 必须在组件之前导入
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ThemeProvider } from "@/shared/hooks/use-theme";
import { useLocaleSync } from "@/shared/hooks/use-locale";
import { TooltipProvider } from "@/shared/ui/tooltip";
import { Toaster } from "@/shared/ui/sonner";

const queryClient = new QueryClient();

// 独立组件确保 hook 在 React 树中调用
function LocaleSync() {
  useLocaleSync();
  return null;
}

function Providers({ children }: { children: React.ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <TooltipProvider>
          <LocaleSync />
          {children}
          <Toaster />
        </TooltipProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}

export { Providers };
```

## 4. UI 组件

### shared/ui/locale-switcher.tsx

语言切换下拉菜单：

```tsx
import { LanguagesIcon } from "lucide-react";
import { type Locale, SUPPORTED_LOCALES, useLocale } from "@/shared/hooks/use-locale";
import { Button } from "@/shared/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu";

const LOCALE_LABELS: Record<Locale, string> = {
  "zh-CN": "简体中文",
  en: "English",
};

function LocaleSwitcher() {
  const { setLocale } = useLocale();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon-xs">
          <LanguagesIcon className="size-4" />
          <span className="sr-only">Switch language</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {SUPPORTED_LOCALES.map((locale) => (
          <DropdownMenuItem
            key={locale}
            onClick={() => setLocale(locale)}
          >
            {LOCALE_LABELS[locale]}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export { LocaleSwitcher };
```

## 5. 翻译文件示例

### locales/zh-CN/common.json

```json
{
  "button": {
    "confirm": "确认",
    "cancel": "取消",
    "save": "保存",
    "delete": "删除",
    "edit": "编辑",
    "create": "新建",
    "close": "关闭"
  },
  "status": {
    "loading": "加载中...",
    "success": "成功",
    "failed": "失败",
    "empty": "暂无数据"
  },
  "validation": {
    "required": "此项为必填",
    "invalidEmail": "请输入有效的邮箱地址"
  }
}
```

### locales/en/common.json

```json
{
  "button": {
    "confirm": "Confirm",
    "cancel": "Cancel",
    "save": "Save",
    "delete": "Delete",
    "edit": "Edit",
    "create": "Create",
    "close": "Close"
  },
  "status": {
    "loading": "Loading...",
    "success": "Success",
    "failed": "Failed",
    "empty": "No data"
  },
  "validation": {
    "required": "This field is required",
    "invalidEmail": "Please enter a valid email"
  }
}
```

### locales/zh-CN/auth.json

```json
{
  "login": {
    "title": "登录到你的工作空间",
    "email": "邮箱",
    "emailPlaceholder": "admin@nanomind.local",
    "password": "密码",
    "passwordPlaceholder": "输入密码",
    "submit": "登录",
    "submitting": "登录中...",
    "success": "登录成功",
    "invalidEmail": "请输入有效的邮箱地址",
    "invalidPassword": "请输入密码",
    "invalidCredentials": "邮箱或密码错误",
    "failed": "登录失败"
  },
  "changePassword": {
    "title": "修改密码",
    "description": "当前使用的是默认密码,请设置一个新密码以确保账户安全。",
    "newPassword": "新密码",
    "newPasswordPlaceholder": "至少 8 个字符",
    "confirmPassword": "确认密码",
    "confirmPasswordPlaceholder": "再次输入新密码",
    "minLength": "密码至少 8 个字符",
    "required": "请确认密码",
    "mismatch": "两次输入的密码不一致",
    "submit": "确认修改",
    "submitting": "修改中...",
    "success": "密码修改成功",
    "failed": "修改密码失败"
  }
}
```

## 6. 组件使用示例

### 基础使用

```tsx
import { useTranslation } from "react-i18next";

function LoginForm() {
  const { t } = useTranslation("auth");

  return (
    <form>
      <h1>{t("login.title")}</h1>
      <label>{t("login.email")}</label>
      <input placeholder={t("login.emailPlaceholder")} />
      <button>{t("login.submit")}</button>
    </form>
  );
}
```

### 多命名空间

```tsx
import { useTranslation } from "react-i18next";

function MyComponent() {
  const { t } = useTranslation(["dashboard", "common"]);

  return (
    <div>
      <h1>{t("dashboard:user.profile")}</h1>
      <button>{t("common:button.save")}</button>
    </div>
  );
}
```

### 带插值

```tsx
function Greeting({ name }: { name: string }) {
  const { t } = useTranslation();

  // JSON: { "greeting": "你好，{{name}}！" }
  return <span>{t("greeting", { name })}</span>;
}
```

### 在非组件中使用

```tsx
import { i18n } from "@/i18n";

// 工具函数中
function showSuccessToast() {
  toast.success(i18n.t("common:status.success"));
}
```

## 7. 新增语言检查清单

添加新语言（以 `ja` 日语为例）：

```bash
# 1. 创建翻译文件目录
mkdir -p site/src/i18n/locales/ja

# 2. 复制基础结构（然后翻译内容）
cp site/src/i18n/locales/en/*.json site/src/i18n/locales/ja/
```

```typescript
// 3. 更新 i18n/index.ts
import authJa from "./locales/ja/auth.json";
import commonJa from "./locales/ja/common.json";
// ...

const resources = {
  // ... existing
  ja: {
    common: commonJa,
    auth: authJa,
    // ...
  },
};
```

```typescript
// 4. 更新 use-locale.ts
export const SUPPORTED_LOCALES = ["zh-CN", "en", "ja"] as const;
```

```typescript
// 5. 更新 locale-switcher.tsx
const LOCALE_LABELS: Record<Locale, string> = {
  "zh-CN": "简体中文",
  en: "English",
  ja: "日本語",
};
```

## 8. 常见问题

### Q: 翻译 key 不存在时显示什么？

默认显示 key 本身。可配置 `fallbackLng` 回退到指定语言。

### Q: 如何处理复数形式？

```json
{
  "item": "{{count}} 项",
  "item_plural": "{{count}} 项"
}
```

注意：中文不区分单复数，英文需要：

```json
{
  "item": "{{count}} item",
  "item_plural": "{{count}} items"
}
```

### Q: 如何在组件挂载前使用翻译？

使用 `i18n.t()` 而非 `useTranslation()` hook：

```typescript
import { i18n } from "@/i18n";

const defaultTitle = i18n.t("common:app.title");
```

### Q: TypeScript 报错找不到翻译 key？

确保 `i18n/types.ts` 正确配置且 tsconfig 包含该文件。
