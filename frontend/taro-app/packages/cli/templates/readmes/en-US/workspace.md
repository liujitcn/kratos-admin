# __PROJECT_NAME__

`__PROJECT_NAME__` is an independent pnpm workspace created by `@liujitcn/kratos-taro-app-cli`. It uses Taro, React, and TypeScript, and supports H5 and WeChat Mini Programs.

## Directory Structure

```text
__PROJECT_NAME__
├── apps/taro-app             # Private Taro host
├── packages/modules          # Local workspace business modules
├── package.json              # Shared commands and development dependencies
├── pnpm-workspace.yaml       # Workspace package patterns
└── tsconfig.json             # Shared TypeScript configuration
```

The host contains only the entry point, module manifest, and platform build configuration. Runtime foundations, UI theme, and default business pages are provided by `@liujitcn/kratos-taro-app-core`, `@liujitcn/kratos-taro-app-ui`, and `@liujitcn/kratos-taro-app-system`.

## Development

```bash
pnpm install
pnpm dev:h5
pnpm dev:mp-weixin
pnpm build:h5
pnpm build:mp-weixin
pnpm tsc
```

Development output is under `apps/taro-app/dist/dev/<platform>` and production output under `apps/taro-app/dist/build/<platform>` (`h5` or `mp-weixin`). For a Kratos-integrated project, the H5 production build is written to `backend/web/taro-app`. WeChat Developer Tools should use `dist/dev/mp-weixin`; import `dist/build/mp-weixin` for release.

The module assembly entry is `apps/taro-app/src/module-manifest.ts`. Declaration order determines static view override priority; update each module's `src/pages.ts` and view mapping when adding pages.

Environment files are at the workspace root and use the same names as uni-app: `VITE_APP_PORT`, `VITE_APP_BASE_PATH`, `VITE_APP_BASE_API`, `VITE_APP_API_URL`, `VITE_APP_STATIC_API`, and `VITE_APP_STATIC_URL`. H5 overlays the matching `*-h5` file on the base mode file.

To access H5 over a LAN IP, run `bash scripts/generate-dev-cert.sh 192.168.1.100` from the repository root, then set `VITE_APP_HTTPS=true` in `.env.development-h5.local`. If the backend uses HTTPS, also set `VITE_APP_API_URL=https://localhost:7001`.
