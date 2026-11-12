# @liujitcn/kratos-uni-app

`apps/uni-app` is the private uni-app x host for `__PROJECT_NAME__`. It assembles modules and provides H5, WeChat Mini Program, and App build entry points; reusable business implementations belong in modules.

## Directory Structure

```text
apps/uni-app
├── App.uvue                 # Root component and global styles
├── main.ts                  # Vue and Kratos uni-app x startup
├── manifest.json            # uni-app x platform configuration
├── module-manifest.ts       # Single module manifest
├── pages/                   # x pages
├── pagesMember/             # x subpackage pages
├── scripts/generate-pages.mjs
├── pages.json               # x page routes
├── index.html               # H5 HTML entry
├── package.json             # Host commands and runtime dependencies
└── vite.config.ts           # Module assembly and uni-app plugins
```

## Module Assembly

`module-manifest.ts` registers core and system by default, then loads local and published modules in declaration order. Keep business pages, APIs, and state in their owning modules; the host only maintains composition and platform configuration.

## Commands

Run from the workspace root:

```bash
pnpm dev:h5
pnpm dev:mp-weixin
pnpm build:h5
pnpm build:mp-weixin
pnpm --filter @liujitcn/kratos-uni-app dev:app
pnpm --filter @liujitcn/kratos-uni-app build:app
```
