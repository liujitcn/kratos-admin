# @liujitcn/kratos-uni-app

`apps/uni-app` is the private uni-app host for `__PROJECT_NAME__`. It assembles modules and provides H5 and WeChat Mini Program build entry points; reusable business implementations belong in modules.

## Directory Structure

```text
apps/uni-app
├── src
│   ├── pages/bootstrap      # Fixed startup page
│   ├── App.vue              # Root component and global styles
│   ├── main.ts              # Vue and Kratos uni-app startup
│   ├── manifest.json        # uni-app platform configuration
│   ├── module-manifest.ts   # Single module manifest
│   └── pages.json           # Fixed bootstrap route
├── index.html               # H5 HTML entry
├── package.json             # Host commands and runtime dependencies
└── vite.config.ts           # Module assembly and uni-app plugins
```

## Module Assembly

`src/module-manifest.ts` registers core and system by default, then loads local and published modules in declaration order. Keep business pages, APIs, and state in their owning modules; the host only maintains composition and platform configuration.

## Commands

Run from the workspace root:

```bash
pnpm dev:h5
pnpm dev:mp-weixin
pnpm build:h5
pnpm build:mp-weixin
```
