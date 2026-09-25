# __PROJECT_NAME__

`__PROJECT_NAME__` is an independent pnpm workspace created by `@liujitcn/kratos-uni-app-cli`. It supports H5 and WeChat Mini Programs.

## Directory Structure

```text
__PROJECT_NAME__
├── apps/uni-app          # Private uni-app host
├── packages/modules     # Local workspace business modules
├── package.json         # Shared commands and development dependencies
├── pnpm-workspace.yaml  # Workspace package patterns
└── tsconfig.json        # Shared TypeScript configuration
```

The host contains only the entry point, manifest, module list, and startup wiring. Core capabilities are provided by `@liujitcn/kratos-uni-app-core`; default business pages are provided by `@liujitcn/kratos-uni-app-system`. Local modules integrate through their public package entry points and must not use cross-package relative imports.

## Development

```bash
pnpm install
pnpm dev:h5
pnpm dev:mp-weixin
pnpm build:h5
pnpm build:mp-weixin
```

To access H5 over a LAN IP, run `bash scripts/generate-dev-cert.sh 192.168.1.100` from the repository root, then set `VITE_APP_HTTPS=true` in `.env.development-h5.local`. If the backend uses HTTPS, also set `VITE_APP_API_URL=https://localhost:7001`.

The module assembly entry is `apps/uni-app/src/module-manifest.ts`. Keep it in sync when adding, removing, or changing modules, and document page and API responsibilities in each module README.

Before a production build, set the real API and static asset URLs in `.env.production`. That file is intentionally blank by default and should not contain localhost addresses.
