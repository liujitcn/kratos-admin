# kratos-admin

[简体中文](README.md) | [繁體中文](README.zh-TW.md) | [English](README.en-US.md) | [日本語](README.ja-JP.md)

`kratos-admin` is a full-stack separated administration system repository containing a Go + Kratos backend, Vue administration console, uni-app application foundation, React/Taro application foundation, versioned database migrations, and module scaffolding.

## Implemented Capabilities

- Account/password, captcha, OAuth, TOTP/WebAuthn multi-factor authentication, JWT refresh, tenant, and Casbin authorization.
- TOTP multi-factor authentication, one-time recovery codes, and the global `disabled`, `optional`, and `all_required` policies.
- Open authorization client management: clients are bound to tenants, credentials exchange for Bearer Tokens, and operation interceptors plus the HTTP encryption/decryption Filter validate tenants, status, IP allowlists, JSON API allowlists, and protocol ciphertext.
- Management of users, roles, departments, positions, menus, dictionaries, configurations, jobs, file assets, logs, regions, APIs, and migration records.
- The administration workbench provides user/role overviews, login trends, login results, and operation distribution statistics.
- Message categories, tenant-targeted/all-tenant internal messages, inbox read/archive operations, Redis delivery recovery, and Admin/uni-app/Taro message centers.
- Proto-driven generation of HTTP, gRPC, OpenAPI, Agent Tool, MCP Tool, and TypeScript RPC code.
- AI sessions, streaming messages, attachments, tool calls, retries, regeneration, and branched conversations.
- Administration-side code generation configuration, preview, generation progress, and restoration.
- Runtime log browsing: live console SSE, historical log queries, level and keyword filters, and original historical file downloads.
- Login-source policies (global and tenant/user-targeted rules), password complexity policies, policy-controlled multi-device login, independent session timeout and revocation, personal login records, platform online-session management, asynchronous audit-log persistence and retention cleanup, and controlled MySQL backup and restore jobs.
- Mountable Go Core modules; the backend implements `module.Module`, provides static assets through `Resources`, and lets the startup entry register protocol services through Core.
- Language sets for the administration console, uni-app, Taro, and backend error catalog are discovered automatically from language packages; dynamic menus, dictionaries, and code generation support all registered languages.

The repository does not include commerce, order, payment, or recommendation modules.

## Directory Layout

| Directory | Description | Documentation |
| --- | --- | --- |
| `backend` | Kratos services, Proto, GORM, migrations, and Core host composition. | [backend/README.md](backend/README.md) |
| `frontend/admin` | Administration workspace containing the default host, core, System, and CLI. | [frontend/admin/README.md](frontend/admin/README.md) |
| `frontend/uni-app` | uni-app workspace containing the default host, core, system, and CLI. | [frontend/uni-app/README.md](frontend/uni-app/README.md) |
| `frontend/taro-app` | React/Taro workspace containing the default host, core, UI, system, and CLI. | [frontend/taro-app/README.md](frontend/taro-app/README.md) |
| `docs` | Current architecture, operating procedures, and topic documentation. | [docs/README.md](docs/README.md) |

See [docs/开放授权协议设计.md](docs/开放授权协议设计.md) for the scope of the open authorization protocol, interceptor boundaries, and encryption extension points.

## Environment

- Go `1.27.0`.
- Node.js `^20.19.0` or `>=22.12.0`.
- The pnpm version follows each workspace's `packageManager`: `10.33.4` for the administration console, and `10.13.1` for uni-app and Taro.
- MySQL, Redis, Consul, and Vault; see the sections below for their purposes and configuration entry points.
- Docker deployment requires an available Docker CLI and Docker daemon.
- When enabling TOTP enrollment, an explicit `mfa.encryption_key` takes precedence; when empty, the key is derived from the runtime key service using `kratos-kit:mfa/encryption` when protecting the TOTP secret. Enabling WebAuthn also requires `mfa.webauthn.rp_id` and `mfa.webauthn.rp_origins`. Sensitive configuration values should be stored using `ENC[...]`.
- Buf, protoc plugins, Wire, and gorm-gen are only required when regenerating code and can be installed with `make -C backend init`.

Run `make` or `make help` to view repository-level commands. Run `make -C backend help` and `make -C frontend help` for the complete Backend and Frontend targets.

### Middleware Dependencies

| Middleware | Purpose | Configuration |
| --- | --- | --- |
| MySQL | Business-data persistence and database migrations. | `backend/configs/data.yaml` |
| Redis | Caching, distributed locks, queues, and message delivery. | `backend/configs/data.yaml` |
| Consul | Service registration and discovery. | `backend/configs/full/registry.yaml` |
| Vault | Application root-key management, configuration decryption, and business-key derivation. | `backend/configs/key.yaml` |

Each environment configures connection addresses and access parameters through the corresponding `<name>.<env>.yaml` file. Before startup, ensure that the middleware is reachable, Vault is initialized and unsealed, and `VAULT_TOKEN` has permission to read the required root keys. Deployment, initialization, and credential management belong to the runtime environment and are not coupled to application startup. Production environments should use secure connections and least-privilege credentials.

## Local Startup

Create the database first:

```sql
CREATE DATABASE kratos_admin CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
```

Install frontend dependencies:

```bash
make -C frontend init
```

To completely rebuild dependency directories:

```bash
make -C frontend reinstall
```

After confirming that Vault is running and unsealed, configure a valid `VAULT_TOKEN` in your terminal or IDE and start the backend:

```bash
make -C backend run-minimal
```

`run-minimal` uses `backend/configs`. Use `make -C backend run-full` when all configuration fields are needed; use `make -C backend run-only` to start the minimal configuration without regenerating artifacts. See [Backend Common Procedures](backend/README.md#常用流程) for complete targets, order, and parameters.

Start all frontend development environments (administration console, uni-app/Taro H5, and WeChat Mini Program):

```bash
make -C frontend run
```

You can also start each target separately (run every long-lived command in its own terminal):

```bash
make -C frontend run-admin
cd frontend/uni-app && pnpm dev:h5
cd ../taro-app && pnpm dev:h5
```

| Service | Default address |
| --- | --- |
| Backend HTTP | `http://localhost:7001` |
| Backend HTTPS (`APP_ENV=https`) | `https://localhost:7001` |
| Backend gRPC | `localhost:6001` |
| Administration console | `http://localhost:8848` |
| uni-app H5 | `http://localhost:5004` |
| Taro H5 | `http://localhost:5002` |

Taro development output is located at `frontend/taro-app/apps/taro-app/dist/dev/<platform>`, WeChat Mini Program production output is located at `dist/build/mp-weixin`, and H5 production output remains in `backend/web/taro-app`. WeChat DevTools uses the development directory by default; import the production directory for releases.

uni-app and Taro H5 default to ports `5004` and `5002` respectively and can run at the same time. For LAN access to uni-app, replace `localhost` with the development machine's LAN IP.

For LAN HTTPS integration, generate a shared certificate in the repository root and run the backend in the HTTPS environment:

```bash
bash scripts/generate-dev-cert.sh 192.168.1.100
make -C backend run-only APP_ENV=https
```

The administration proxy must set `VITE_PROXY` in a local environment file to `https://localhost:7001`; Taro and uni-app H5 must set `VITE_APP_API_URL` in their respective `.env.development-h5.local` files to `https://localhost:7001`. All three development proxies skip self-signed certificate verification.

The default migration provides development accounts `super / 112233` and `admin / 112233`. Change the default passwords, JWT key, database credentials, and Redis credentials before deployment.

## Generation and Checks

```bash
make gen
make check
make build
make -C backend fmt
```

`make gen` generates repository artifacts in Backend, Frontend, language packages, and OpenAPI order; `make check` checks Backend, the three frontend workspaces, and internationalization in order. The root `make build` builds the backend binary and three frontend H5 hosts; use `make -C frontend build` to build all frontends (H5 + WeChat Mini Program), `make -C frontend build-h5` for H5 only, and `make -C frontend package` to generate all npm release packages.

`make -C backend cli` installs `kratos-kit/cmd/normalize-go-imports`; `make -C backend fmt` runs it and formats all Backend Go files with `goimports`. Code-generation tasks receive a file list through `FMT_FILE_LIST` (one Backend-relative path per line) and format only the files rewritten in that task. `make -C backend api` normalizes Go import aliases for protocol artifacts after generation to prevent unrelated diffs between full generation and per-file formatting.

The shared implementation of this tool is `kratos-kit/cmd/normalize-go-imports`; the Admin repository no longer keeps a separate Make target for it.

The backend defaults to building a `linux/amd64` binary through `make -C backend build`; the root `make package` creates a backend release archive containing `bin/server` and `configs`, and also generates all frontend npm packages. Override the target platform with `GOOS` and `GOARCH`.

Build Docker images from the repository root:

```bash
make docker-build IMAGE=kratos-admin TAG=latest
make docker-build-multiarch IMAGE=registry.example.com/kratos-admin TAG=latest
make docker-run IMAGE=kratos-admin TAG=latest
make docker-stop IMAGE=kratos-admin TAG=latest
```

`docker-build` uses Docker Buildx to build the single platform specified by `DOCKER_PLATFORM` and load it into the local Docker image store with `--load`, defaulting to `linux/amd64`; inspect it with `docker run` or `docker image ls`. `docker-build-multiarch` uses Docker Buildx to build `linux/amd64` and `linux/arm64` simultaneously, using Docker media types with provenance disabled before pushing to the image registry by default for SWR Basic compatibility; override platforms and output with `DOCKER_PLATFORMS` and `DOCKER_OUTPUT`. The classic local image store cannot load a multi-platform image in one operation; to load one platform locally, run `make docker-build-multiarch DOCKER_PLATFORMS=linux/amd64 DOCKER_OUTPUT=--load`. The build command checks Docker first, then rebuilds the administration console, uni-app H5, and Taro H5, and compiles the backend for the target architecture through a Docker multi-stage build. The three H5 builds run in parallel; the Dockerfile reuses Go modules and the compilation cache. The run command publishes host ports `7001/6001`, mapping `backend/data`, `backend/logs`, `backend/backups`, and `backend/configs` to `/app/data`, `/app/logs`, `/app/backups`, and `/app/configs`. The image contains default configs and static assets for all three clients; startup only supplements missing image configs into the host's `backend/configs` and does not overwrite modified host configs, then starts the service with that directory. Static sites are supplemented into `backend/data` without clearing existing uploads; Core maps local objects uniformly to `/data/` according to `oss.root_directory`. For offline import, export a single-platform Docker tar with `--output type=docker,dest=kratos-admin-amd64.tar`; a multi-platform image cannot be exported as one Docker tar. See this section for complete build parameters and examples.

`I18N_LOCALES` is a comma-separated list of BCP 47 language codes (discovered from backend language packages by default, excluding the primary language) and controls OpenAPI target languages. `make i18n` generates multilingual OpenAPI YAML. For offline generation, use `I18N_OFFLINE=1 make i18n`.

`backend/api/gen`, `backend/internal/data/gen`, `src/rpc` in each frontend package, OpenAPI, and `wire_gen.go` are generated artifacts and must not be edited manually. Buf configurations for all frontend RPCs are centralized in `backend/api`; generate the administration client with `make -C frontend ts-admin`, the application clients with `make -C frontend ts-uni-app` and `make -C frontend ts-taro-app`, all three with `make -C frontend ts`, and the entire repository with the root `make gen`.

`make -C backend gen` generates business-module assembly in `backend/internal/module` and the independent startup Wire artifact in `backend/internal/cmd/server`; refresh the former alone with `make -C backend public-wire`, or specify a directory containing `wire.go` through `WIRE_DIR` for a custom composition root.

External Go projects merge the named modules, resources, scheduled jobs, SSE streams, and queue consumers contributed by `github.com/liujitcn/kratos-admin/backend` with their own contributions, then pass them to `github.com/liujitcn/kratos-core` to create protocol services and the application lifecycle. External generated code does not reference `backend/internal`.

Admin's public `backend/adapter/core` and `backend/adapter/kit` constructors accept only the database client and create the required repositories internally. They participate in Wire through the Core storage interfaces and Kit redaction interfaces respectively. The redaction resolver is injected per application instance and passed through request contexts; storage callbacks bind to the corresponding database and do not use a global default instance. External projects retain a normal single-module structure and need no additional interface set or special host module. Cross-repository release order is Kit redact and server/grpc, Core, then Admin Backend.

`make i18n` generates `openapi.en-US.yaml`, `openapi.zh-TW.yaml`, and `openapi.ja-JP.yaml` after generating `openapi.yaml`. Default resources come from the backend error catalog and the administration Core language packages; external resources can be supplied through `OPENAPI_I18N_CONTENT="language=path"`. Unmatched text is automatically translated by default: English and Japanese use Google V1, and Traditional Chinese uses OpenCC. Set `I18N_AUTO_LOCALIZE=0` to disable automatic translation. Use `I18N_OFFLINE=1 make i18n` without network access.

## Internationalization

For daily work, run `make i18n` (sync, generate, and validate). For commits or CI checks, run `make i18n-check` (read-only). Add a language with `make i18n-add I18N_LOCALE=de-DE`, review the draft manually, and then run `make i18n`; OpenAPI discovers existing language packages by default, or you can limit target languages with `I18N_LOCALES`.

| Content to maintain | Source files |
| --- | --- |
| Fixed UI text for the administration console, uni-app, and Taro | `src/locales/*.json` in each core and business module |
| Backend error messages and code-generation template text | `backend/internal/i18n/assets/*.json` |
| Dynamic translations for menus, dictionaries, configurations, and jobs | `backend/migration/assets/v0.0.1/mysql/i18n.*.up.sql`, stored in `base_i18n` at runtime |
| API documentation titles, descriptions, and field descriptions | Proto Chinese descriptions and local terminology mappings in `scripts/local_openapi_i18n.py` |
| Migration notes and project documentation with available translations | Corresponding `README.<locale>.md` and other documents |

When changing fixed text, add the same keys and placeholders for every language in that module. `make i18n` does not automatically fill missing UI translations; synchronization fails when missing entries are detected. `src/locales/generated.ts` and similar language-registration files, as well as `openapi.<locale>.yaml`, are generated artifacts and must not be maintained manually. Changes to initialization SQL do not automatically overwrite translations in already-migrated databases.

See [Internationalization Language Extension Guide](docs/国际化语言扩展指南.md) for the complete description.

Language packages define the set of languages the system can render. The `base_language` table only manages runtime enablement, names, ordering, and primary-language configuration. The administration locale preference is stored as `kratos-admin:locale`; uni-app and Taro use `kratos-app:locale`. All HTTP, refresh-token, fetch, SSE, uni.request, and Taro.request requests send a normalized `Accept-Language`. Fixed text is maintained by Core/System JSON language packages in each workspace; dynamic menus and dictionaries are resolved by backend translation tables for the request language and fall back to the primary language when translations are missing.

Adding a language requires no changes to Go, TypeScript, or module registration code: add same-named JSON files under `backend/internal/i18n/assets` and the six frontend language-package directories in the three workspaces, then run `make i18n`. The script validates language sets, language keys, and placeholders, and generates six frontend registration files plus Element Plus and Day.js mappings. Language name, ordering, enabled state, and primary-language state come from `base_language` records; `common.language.*` is used for compile-time offline display and initial names in generated language migrations. See the [Internationalization Language Extension Guide](docs/国际化语言扩展指南.md) for the complete file list and migration flow. To add the language to a new deployment database, update the single `v0.0.1` initialization migration directly; existing database enablement is not overwritten by migrations.

The primary language for dynamic resources is configured by `base_language.is_primary`. When creating or updating menus, dictionaries, dictionary items, and system configurations, the backend converts input text to the primary language according to `Accept-Language` and writes it to the primary table. When the request language is not primary, the original text is written to the corresponding translation table; other enabled non-primary languages are also stored only in translation tables. The administration console supports opening a translation dialog by clicking names for system configurations, menu titles, dictionary names, and dictionary-item labels. Text and rich-text configuration values support runtime translation fallback.

## Release

The unified release updates and publishes 10 npm packages:

- `@liujitcn/kratos-admin-core`
- `@liujitcn/kratos-admin-system`
- `@liujitcn/kratos-admin-cli`
- `@liujitcn/kratos-uni-app-core`
- `@liujitcn/kratos-uni-app-system`
- `@liujitcn/kratos-uni-app-cli`
- `@liujitcn/kratos-taro-app-core`
- `@liujitcn/kratos-taro-app-ui`
- `@liujitcn/kratos-taro-app-system`
- `@liujitcn/kratos-taro-app-cli`

```bash
make tag VERSION=0.0.30
```

`make tag` first runs the read-only `make i18n-check` to inspect language packages, SQL translation scripts, and multilingual OpenAPI documents. It refuses to release when generated artifacts are out of sync and does not translate or rewrite files during release. The release script then requires the current branch to be the remote default branch and synchronized with `origin`, runs backend tests and frontend packaging, and pushes `vX.Y.Z`, `backend/vX.Y.Z`, and `npm/vX.Y.Z`. `npm/vX.Y.Z` triggers `.github/workflows/publish-npm.yml` and publishes the 10 packages above through npm Trusted Publishing; the three default hosts are private packages and are not published. A usable `git`, `gh`, and GitHub login session are required locally.

For local npm publishing only:

```bash
pnpm login
make -C frontend publish
```

The publishing script skips package versions already present in the registry and supports retries after failures. Override a private registry with `NPM_REGISTRY`, `NPM_ACCESS`, and `NPM_TAG`.

## Documentation

| Topic | Document |
| --- | --- |
| Overall architecture | [docs/系统总体设计.md](docs/系统总体设计.md) |
| Integrating new capabilities | [docs/服务接入指南.md](docs/服务接入指南.md) |
| Database migrations | [docs/数据库与初始化数据设计.md](docs/数据库与初始化数据设计.md) |
| Parameter validation | [docs/接口参数校验设计.md](docs/接口参数校验设计.md) |
| Login and passwords | [docs/登录与密码加密流程.md](docs/登录与密码加密流程.md) |
| AI assistant | [docs/AI助手设计.md](docs/AI助手设计.md) |
| Internal messaging | [docs/站内信设计.md](docs/站内信设计.md) |
| Administration components | [docs/前端组件清单.md](docs/前端组件清单.md) |
| Internationalization design | [docs/国际化最终方案.md](docs/国际化最终方案.md) |
| Security policies and operations jobs | [docs/安全策略与运维任务.md](docs/安全策略与运维任务.md) |
| Adding languages | [docs/国际化语言扩展指南.md](docs/国际化语言扩展指南.md) |

When creating external projects, the `packages/cli` packages for all three clients independently generate complete frontends, including language registration, host lifecycle, and check/build tools.
The Go scaffolding only calls the npm CLI; `--kratos-project` adapts backend static output, while the administration CLI generates the shared frontend Makefile and scripts.
All three clients support a local `system` module, with no temporary module names or post-generation file patching. CLI updates must be published to npm first so that the Go code can call the exact version.

Backend's `NewModules` and `NewStreams` share the `*backend.CodeGenManager` injected by the host; `backend.ProviderSet` assembles it automatically, and manual calls to these two entry points must pass the same instance. After changing internal dependency assembly, run `make -C backend public-wire wire`.

System administration's foundational management is unified under the “System Configuration” entry, which maintains ordinary and form configurations. Form types load module-registered forms by configuration key and reuse the unified query and update interfaces.

Migration files are embedded in the backend binary; Docker and backend archives still include the directory for release packaging. The database stores only migration-file references and checksums, so historical files must be retained; existing body records should be backed up and converted before upgrades. See [Backend Migration Files and Records](backend/README.md#迁移文件与记录).

Frontend builds use staged, task-grouped plain-text logs with terminal colors disabled. Administration auto-import declarations are generated explicitly with `make -C frontend types-admin`; ordinary builds no longer modify component declarations in the source directory.

Makefiles generated at the root, Backend, Frontend, and CLI levels all disable recursive-directory notices and successful-task cache logs; failed tasks still print complete errors. pnpm, Turbo, Docker, and release scripts inherit the no-color environment, while development services retain their actual runtime logs.
