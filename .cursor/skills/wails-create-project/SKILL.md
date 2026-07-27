---
name: wails-create-project
description: >-
  Scaffolds a new Wails v3 desktop app with Vue 3 + TypeScript, Ant Design Vue
  (zh-CN), and Tailwind CSS v4, plus team conventions (version.go, Taskfile,
  run-dev/build-release). Use when creating a new Wails project, running
  wails3 init, bootstrapping a Vue desktop app, or before adding GitHub
  auto-update / release workflows.
---

# Wails 创建项目（Vue3 + Ant Design Vue + Tailwind）

**存放位置：** 本仓库 `.cursor/skills/`（项目技能，勿写入 `~/.cursor/skills/`）。

用 **Wails v3** 初始化桌面项目，默认前端栈：

- Vue 3 + TypeScript + Vite
- Ant Design Vue（中文 `zh_CN` + dayjs `zh-cn`）
- Tailwind CSS v4（`@tailwindcss/vite`）

创建完成后，再接入 `wails-github-update-release`。

## 与更新发版技能的关系

**新项目固定顺序：**

1. **先**按本技能创建项目
2. **再**按 `.cursor/skills/wails-github-update-release/SKILL.md` 接入更新与发版

已有可运行工程 → 可跳过本技能，直接用更新发版技能。

## 开始前收集

| 占位符 | 含义 | 示例 |
|--------|------|------|
| `DIR` | 项目目录 | `d:\Project\z\go\my-app` |
| `PRODUCT` | 产品显示名 / 二进制名 | `ZDisk` |
| `MODULE` | Go module path | `my-app` |
| `COMPANY` | 公司名 | `Z` |
| `IDENTIFIER` | 包标识 | `com.example.myapp` |
| `BIN_DIR` | 输出目录 | 默认 `dist` |

模板固定为 **`vue`**（Vue + TypeScript）。非 Vue 栈需用户明确说明后再偏离默认。

## 环境要求

- Go 1.25+
- Node.js 18+
- Wails v3：`go install github.com/wailsapp/wails/v3/cmd/wails3@latest`
- Windows 打包另需 NSIS（可后装）

## 创建步骤

### 1. wails3 init

```bat
cd /d PARENT_DIR
wails3 init -n PRODUCT -d DIR -t vue -mod MODULE ^
  -productname PRODUCT -productcompany COMPANY ^
  -productidentifier IDENTIFIER -productversion 1.0.0 ^
  -productdescription "应用描述"
cd DIR
```

注意：`-d` 为项目目录；若 `DIR` 与 `-n` 名称不同，CLI 可能再套一层子目录，需把内容提升到目标根目录。

### 2. 前端：Ant Design Vue + Tailwind

在 `frontend/`：

```bat
npm install ant-design-vue dayjs
npm install -D tailwindcss @tailwindcss/vite
```

**`vite.config.ts`**（保留 wails 插件，加上 tailwind）：

```ts
import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import tailwindcss from "@tailwindcss/vite";
import wails from "@wailsio/runtime/plugins/vite";

export default defineConfig({
  plugins: [vue(), tailwindcss(), wails("./bindings")],
});
```

**`src/style.css`** 开头：

```css
@import "tailwindcss";
```

**`src/main.ts`**：

```ts
import { createApp } from "vue";
import Antd from "ant-design-vue";
import "ant-design-vue/dist/reset.css";
import dayjs from "dayjs";
import "dayjs/locale/zh-cn";
import App from "./App.vue";
import "./style.css";

dayjs.locale("zh-cn");
createApp(App).use(Antd).mount("#app");
```

**`App.vue`**：用 `a-config-provider` 包一层，`:locale="zhCN"`（`ant-design-vue/es/locale/zh_CN`）。组件优先用 `a-*`，不要默认上 Naive UI。

### 3. 团队约定落地

- [ ] `version.go`：`AppName` / `AppVersion`，与 `build/config.yml` → `info.version` 一致
- [ ] `Taskfile.yml`：`APP_NAME: "PRODUCT"`，`BIN_DIR: "dist"`
- [ ] `build/config.yml`：`productName` / `companyName` / `version`
- [ ] `scripts/run-dev.bat`：GOPATH/bin 入 PATH → `wails3 task dev`
- [ ] `scripts/build-release.bat`：检测 `wails3` + `makensis`（自动探测 NSIS 目录）→ build + package
- [ ] `.gitignore`：`frontend/bindings/`、`frontend/dist/`、`dist/`、`node_modules`
- [ ] README：开发 / 打包说明（中文）

### 4. 首次依赖与试跑

```bat
cd frontend && npm install && cd ..
scripts\run-dev.bat
```

### 5. 再接入更新发版

可运行后读取并执行：

`.cursor/skills/wails-github-update-release/SKILL.md`

## 默认选择

| 项 | 默认 |
|----|------|
| 模板 | `vue`（Vue3 + TS） |
| UI | Ant Design Vue + zh_CN |
| 样式 | Tailwind CSS v4 |
| BIN_DIR | `dist` |
| 版本常量 | `AppVersion` |
| 更新发版 | **不**在创建时接入 |

## 不要做的事

- 空目录直接铺更新发版代码
- 默认改用 Naive UI / React（除非用户明确要求）
- 提交 `frontend/bindings/`
- `productName` / `APP_NAME` / `version.go` 三者不一致

## 验收

```bat
scripts\run-dev.bat
scripts\build-release.bat
```

然后进入 `wails-github-update-release`。
