<!-- prettier-ignore -->
# __APP_PACKAGE__

The admin host for `__PROJECT_NAME__`. This private package contains no business logic. It combines `@liujitcn/kratos-admin-system`, custom modules (__MODULE_PACKAGES__), and any selected additional modules.

## Files and directories

```text
apps/admin
├── src
│   ├── main.ts
│   ├── module-manifest.ts
│   ├── modules.ts
│   └── vite-env.d.ts
├── .env
├── .env.development
├── .env.production
├── favicon.svg
├── index.html
├── package.json
├── README.md
├── tsconfig.json
└── vite.config.ts
```

| Path | Purpose |
| --- | --- |
| `src/main.ts` | Passes all default-exported modules to core to start the Vue app. |
| `src/module-manifest.ts` | Declares module loaders, package names, and optional pre-bundled dependencies. |
| `src/modules.ts` | Loads and default-exports the modules enabled for this host. |
| `src/vite-env.d.ts` | Includes Vite client and core global types. |
| `.env` | Shared app title, port, and other environment variables. |
| `.env.development` | Development API URL and proxy configuration. |
| `.env.production` | Production API URL and build configuration. |
| `favicon.svg` | Browser tab icon for the admin app. |
| `index.html` | Vite HTML entry point and app mount node. |
| `package.json` | Declares host commands and core, current module, and extra module dependencies. |
| `README.md` | Documents this host's files and development workflow. |
| `tsconfig.json` | TypeScript configuration for the host. |
| `vite.config.ts` | Combines core configuration with build settings derived from the module manifest. |

## Module composition

`src/module-manifest.ts` is the only source of host module configuration. Each manifest entry declares its npm package, runtime loader, and optional pre-bundled dependencies. `src/modules.ts` loads and default-exports all modules, while `vite.config.ts` derives module scanning settings from the manifest. To add or remove a module, update the host dependency and this manifest.

Registration order only affects explicit `staticViews` replacements: a later module replaces an earlier module's matching static view key. Ordinary business pages are isolated by `<module>/<view>` and cannot replace pages in other modules by sharing a name.

Use Vue Router for navigation between modules. Cross-module code may only import public npm subpaths; the host must not import module `src` directories.

## Commands

```bash
pnpm --filter __APP_PACKAGE__ dev
pnpm --filter __APP_PACKAGE__ type:check
pnpm --filter __APP_PACKAGE__ build
```

For LAN access, put a shared certificate in the workspace root `certs` directory and set `VITE_HTTPS=true` in `.env.development.local`. You can also specify certificate paths with `VITE_HTTPS_KEY` and `VITE_HTTPS_CERT`.
