# kratos-admin

[简体中文](README.md) | [繁體中文](README.zh-TW.md) | [English](README.en-US.md) | [日本語](README.ja-JP.md)

`kratos-admin` 是一個前後端分離的管理系統儲存庫，包含 Go + Kratos 後端、Vue 管理後台、uni-app 應用底座、React/Taro 應用底座、版本化資料庫遷移與模組腳手架。

## 已實作能力

- 帳號密碼、驗證碼、OAuth、TOTP/WebAuthn 多因素驗證、JWT 更新、租戶與 Casbin 權限。
- TOTP 多因素驗證、一次性復原碼，以及 `disabled`、`optional`、`all_required` 全域策略。
- 開放授權用戶端管理：用戶端依租戶綁定，憑據換取 Bearer Token，並由 operation 攔截器與 HTTP 加解密 Filter 驗證租戶、狀態、IP 白名單、JSON API 白名單及協定密文。
- 使用者、角色、部門、職位、選單、字典、設定、工作、檔案資產、日誌、地區、API 與遷移記錄管理。
- 後台工作台提供使用者/角色概覽、登入趨勢、登入結果及操作動作分布統計。
- 訊息分類、租戶內定向/全員站內信、收件匣已讀/封存、Redis 投遞復原與 Admin/uni-app/Taro 訊息中心。
- Proto 驅動的 HTTP、gRPC、OpenAPI、Agent Tool、MCP Tool 與 TypeScript RPC 產生。
- AI 會話、串流訊息、附件、工具呼叫、重試、重新產生與分支會話。
- 管理端程式碼產生設定、預覽、產生進度與還原。
- 執行日誌瀏覽：即時主控台 SSE、歷史日誌查詢、層級與關鍵字篩選，以及歷史原始檔案下載。
- 登入來源策略（全域及租戶/使用者定向規則）、密碼複雜度策略、按策略啟用多裝置登入、獨立工作階段逾時與撤銷、本人登入記錄、平台線上工作階段管理、稽核日誌非同步落庫與保留清理，以及受控 MySQL 備份還原工作。
- 可掛載的 Go Core 模組；後端實作 `module.Module`，透過 `Resources` 提供靜態資源，並由啟動入口交給 Core 統一註冊協定服務。
- 管理端、uni-app、Taro 與後端錯誤目錄的語言集合由語言包自動發現；動態選單、字典與程式碼產生同步支援所有已註冊語言。

儲存庫不包含商城、訂單、支付或推薦等業務模組。

## 目錄

| 目錄 | 說明 | 文件 |
| --- | --- | --- |
| `backend` | Kratos 服務、Proto、GORM、遷移與 Core 宿主組合。 | [backend/README.md](backend/README.md) |
| `frontend/admin` | 管理端 workspace，包含預設宿主、core、System 與 CLI。 | [frontend/admin/README.md](frontend/admin/README.md) |
| `frontend/uni-app` | uni-app workspace，包含預設宿主、core、system 與 CLI。 | [frontend/uni-app/README.md](frontend/uni-app/README.md) |
| `frontend/taro-app` | React/Taro workspace，包含預設宿主、core、UI、system 與 CLI。 | [frontend/taro-app/README.md](frontend/taro-app/README.md) |
| `docs` | 目前架構、操作流程與專題說明。 | [docs/README.md](docs/README.md) |

開放授權協定的介面範圍、攔截器邊界與加密擴充點，請參閱 [docs/開放授權協定設計.md](docs/開放授權協定設計.md)。

## 環境

- Go `1.27.0`。
- Node.js `^20.19.0` 或 `>=22.12.0`。
- pnpm 版本依各 workspace 的 `packageManager` 為準：管理後台 `10.33.4`，uni-app 與 Taro 應用端 `10.13.1`。
- MySQL、Redis、Consul 與 Vault；用途與設定入口見下方說明。
- Docker 部署需要可用的 Docker CLI 與 Docker daemon。
- 啟用 TOTP 綁定時，`mfa.encryption_key` 有明確值則使用該值，留空時在實際保護 TOTP 金鑰時按 `kratos-kit:mfa/encryption` 從執行時金鑰服務派生；啟用 WebAuthn 時還需設定 `mfa.webauthn.rp_id` 與 `mfa.webauthn.rp_origins`。設定檔中的敏感值應使用 `ENC[...]` 儲存。
- Buf、protoc 插件、Wire 與 gorm-gen 僅在重新產生程式碼時需要，可透過 `make -C backend init` 安裝。

直接執行 `make` 或 `make help` 查看儲存庫命令；Backend 與 Frontend 的完整目標分別使用 `make -C backend help`、`make -C frontend help` 查看。

### 依賴中介軟體

| 中介軟體 | 用途 | 設定入口 |
| --- | --- | --- |
| MySQL | 業務資料持久化與資料庫遷移。 | `backend/configs/data.yaml` |
| Redis | 快取、分散式鎖、佇列與訊息投遞。 | `backend/configs/data.yaml` |
| Consul | 服務註冊與發現。 | `backend/configs/full/registry.yaml` |
| Vault | 應用根金鑰管理、設定解密與業務金鑰派生。 | `backend/configs/key.yaml` |

各環境透過對應的 `<name>.<env>.yaml` 設定連線位址與存取參數。啟動前應確保中介軟體可存取，Vault 已初始化並解封，且 `VAULT_TOKEN` 具有讀取所需根金鑰的權限；部署、初始化與憑據維護由執行環境負責，不與專案啟動聯動。生產環境應使用安全連線與最小權限憑據。

## 本機啟動

先建立資料庫：

```sql
CREATE DATABASE kratos_admin CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
```

安裝前端依賴：

```bash
make -C frontend init
```

如果依賴目錄需要完全重建：

```bash
make -C frontend reinstall
```

確認 Vault 已啟動並解封，在終端機或 IDE 中自行設定有效的 `VAULT_TOKEN` 後啟動後端：

```bash
make -C backend run-minimal
```

`run-minimal` 使用 `backend/configs`。需要完整欄位時使用 `make -C backend run-full`；確認產物未變更時可使用 `make -C backend run-only` 直接啟動最小設定。完整目標、執行順序與參數請參閱 [Backend 常用流程](backend/README.md#常用流程)。

啟動全部前端開發環境（管理後台、uni-app/Taro H5 與微信小程式）：

```bash
make -C frontend run
```

也可以依端啟動（每個常駐命令都應在獨立終端機執行）：

```bash
make -C frontend run-admin
cd frontend/uni-app && pnpm dev:h5
cd ../taro-app && pnpm dev:h5
```

| 服務 | 預設位址 |
| --- | --- |
| 後端 HTTP | `http://localhost:7001` |
| 後端 HTTPS（`APP_ENV=https`） | `https://localhost:7001` |
| 後端 gRPC | `localhost:6001` |
| 管理後台 | `http://localhost:8848` |
| uni-app H5 | `http://localhost:5004` |
| Taro H5 | `http://localhost:5002` |

Taro 開發產物位於 `frontend/taro-app/apps/taro-app/dist/dev/<平台>`，微信小程式生產產物位於 `dist/build/mp-weixin`；H5 生產產物仍輸出到 `backend/web/taro-app`。微信開發者工具預設使用開發目錄，發佈時匯入生產目錄。

uni-app 與 Taro H5 預設分別使用 `5004` 與 `5002`，可以同時啟動。區域網路裝置存取 uni-app 時，將 `localhost` 替換為開發機區域網路 IP。

區域網路 HTTPS 聯調時，在儲存庫根目錄產生共用憑證，並讓後端使用 HTTPS 環境：

```bash
bash scripts/generate-dev-cert.sh 192.168.1.100
make -C backend run-only APP_ENV=https
```

管理端開發代理需要在本機環境檔案中將 `VITE_PROXY` 的後端位址改為 `https://localhost:7001`；Taro 與 uni-app H5 需要在各自 `.env.development-h5.local` 中將 `VITE_APP_API_URL` 改為 `https://localhost:7001`。三端開發代理都會略過自簽憑證驗證。

預設遷移提供開發帳號 `super / 112233` 與 `admin / 112233`。部署前必須修改預設密碼、JWT 金鑰、資料庫與 Redis 憑據。

## 產生與檢查

```bash
make gen
make check
make build
make -C backend fmt
```

`make gen` 依 Backend、Frontend、語言包與 OpenAPI 的順序產生全倉產物；`make check` 依 Backend、三個前端 workspace 與國際化的順序執行檢查。根目錄 `make build` 建置後端二進位檔與三個前端 H5 宿主；只建置全部前端（H5 + 微信小程式）可使用 `make -C frontend build`，僅建置 H5 使用 `make -C frontend build-h5`，產生全部 npm 發佈包使用 `make -C frontend package`。

`make -C backend cli` 會安裝 `kratos-kit/cmd/normalize-go-imports`，`make -C backend fmt` 再執行該命令並使用 `goimports` 格式化 Backend 全部 Go 檔案。程式碼產生任務透過 `FMT_FILE_LIST` 傳入檔案清單（每行一個 Backend 相對路徑），僅格式化本次改寫檔案。`make -C backend api` 在產生結束時統一規範化協定產物的 Go import 別名，避免全量產生與依檔案格式化之間反覆產生無關差異。

該工具實作統一位於 `kratos-kit/cmd/normalize-go-imports`，不再於 Admin 儲存庫保留獨立 Make 目標。

後端預設透過 `make -C backend build` 建置 `linux/amd64` 二進位檔；儲存庫根目錄的 `make package` 會產生包含 `bin/server` 與 `configs` 的後端發佈壓縮包，並同時產生全部前端 npm 套件。目標平台可使用 `GOOS`、`GOARCH` 覆蓋。

Docker 映像檔透過儲存庫根目錄命令建置：

```bash
make docker-build IMAGE=kratos-admin TAG=latest
make docker-build-multiarch IMAGE=registry.example.com/kratos-admin TAG=latest
make docker-run IMAGE=kratos-admin TAG=latest
make docker-stop IMAGE=kratos-admin TAG=latest
```

`docker-build` 使用 Docker Buildx 建置 `DOCKER_PLATFORM` 指定的單一平台，並透過 `--load` 載入本機 Docker 映像檔庫，預設為 `linux/amd64`；可直接使用 `docker run` 或 `docker image ls` 檢查。`docker-build-multiarch` 使用 Docker Buildx 同時建置 `linux/amd64` 與 `linux/arm64`，預設使用 Docker media types 並停用 provenance 附件後推送到映像檔倉庫，以相容 SWR 基礎版；可使用 `DOCKER_PLATFORMS` 與 `DOCKER_OUTPUT` 覆蓋平台及輸出方式。傳統本機映像檔庫無法一次載入多平台映像檔；如需將單一平台載入本機，可執行 `make docker-build-multiarch DOCKER_PLATFORMS=linux/amd64 DOCKER_OUTPUT=--load`。建置命令先檢查 Docker，再重新建置管理後台、uni-app H5、Taro H5，後端程式由 Docker 多階段建置按目標架構編譯。三個 H5 建置會並行執行；Dockerfile 會重用 Go 模組與編譯快取。執行命令發佈主機 `7001/6001` 埠，將 `backend/data`、`backend/logs`、`backend/backups` 與 `backend/configs` 分別對映到容器的 `/app/data`、`/app/logs`、`/app/backups` 與 `/app/configs`。映像檔內包含預設 `configs` 與三端靜態資源；容器啟動時僅將映像檔中的缺少設定補充到主機的 `backend/configs`，不會覆蓋主機已修改的設定，再使用該目錄啟動服務。靜態站點啟動時補充到 `backend/data`，既有上傳檔案不會被清除；Core 根據 `oss.root_directory` 將本機物件統一對映到 `/data/`。如需離線歸檔，請按單一平台使用 `--output type=docker,dest=kratos-admin-amd64.tar` 輸出 Docker tar；多平台映像檔無法輸出成單一 Docker tar。完整建置參數與執行範例見本節。

`I18N_LOCALES` 使用逗號分隔的 BCP 47 語言代碼清單（預設從後端語言包自動發現，排除主語言），控制 OpenAPI 的目標語言。`make i18n` 產生 OpenAPI 多語言 YAML。離線產生使用 `I18N_OFFLINE=1 make i18n`。

`backend/api/gen`、`backend/internal/data/gen`、各前端套件的 `src/rpc`、OpenAPI 及 `wire_gen.go` 都是產生產物，不得手動修改。所有前端 RPC 的 Buf 設定統一位於 `backend/api`，管理端透過 `make -C frontend ts-admin` 產生，應用端分別透過 `make -C frontend ts-uni-app` 與 `make -C frontend ts-taro-app` 產生；需要一次產生三端時執行 `make -C frontend ts`，全倉產生使用根目錄 `make gen`。

`make -C backend gen` 會產生 `backend/internal/module` 的業務模組組裝與 `backend/internal/cmd/server` 的獨立啟動 Wire 產物；單獨更新前者使用 `make -C backend public-wire`，單獨更新自訂組合根可透過 `WIRE_DIR` 指定包含 `wire.go` 的目錄。

外部 Go 專案將 `github.com/liujitcn/kratos-admin/backend` 的具名模組、資源、排程工作、SSE 串流與佇列消費者貢獻，與自身貢獻合併後，再交給 `github.com/liujitcn/kratos-core` 建立協定服務及應用生命週期；外部產生程式碼不會引用 `backend/internal`。

Admin 公開的 `backend/adapter/core` 與 `backend/adapter/kit` 建構函式只接收資料庫用戶端，並在內部建立所需儲存庫，分別透過 Core 儲存介面與 Kit 脫敏介面參與 Wire。脫敏解析器按應用程式實例注入並隨請求上下文傳遞，儲存回呼綁定對應資料庫，不使用全域預設實例。外部專案保持普通單模組結構，無需額外的介面集合或特殊宿主模組；跨儲存庫發佈順序為 Kit redact 與 server/grpc、Core、Admin Backend。

`make i18n` 會在產生 `openapi.yaml` 後同步產生 `openapi.en-US.yaml`、`openapi.zh-TW.yaml` 與 `openapi.ja-JP.yaml`。預設資源來自後端錯誤目錄與管理端 Core 語言包；外部資源可透過 `OPENAPI_I18N_CONTENT="語言=路徑"` 傳入。未命中的文案預設自動翻譯：英文與日文使用 Google V1，繁體中文使用 OpenCC；設定 `I18N_AUTO_LOCALIZE=0` 可關閉自動翻譯。無網路環境使用 `I18N_OFFLINE=1 make i18n`。

## 國際化

日常只需 `make i18n`（同步、產生並校驗），提交或 CI 檢查使用 `make i18n-check`（唯讀）。新增語言時使用 `make i18n-add I18N_LOCALE=de-DE` 產生草稿，人工複核後再執行 `make i18n`；OpenAPI 預設自動發現已有語言包，也可透過 `I18N_LOCALES` 限定目標語言。

| 需要維護的內容 | 源檔案 |
| --- | --- |
| 管理端、uni-app、Taro 的固定介面文案 | 各端 core 與業務模組的 `src/locales/*.json` |
| 後端錯誤提示、程式碼產生範本文案 | `backend/internal/i18n/assets/*.json` |
| 選單、字典、設定、工作等動態資源譯文 | `backend/migration/assets/v0.0.1/mysql/i18n.*.up.sql`，執行時儲存於 `base_i18n` |
| API 文件標題、說明與欄位描述 | Proto 中文說明及 `scripts/local_openapi_i18n.py` 本地術語對映 |
| 已提供多語言版本的遷移說明與專案文件 | 對應的 `README.<locale>.md` 等文件 |

修改固定文案時須補齊該模組各語言的相同 key 與佔位符。`make i18n` 不會自動補齊缺少的介面譯文；同步發現缺失會直接失敗。`src/locales/generated.ts` 等語言註冊檔案和 `openapi.<locale>.yaml` 是產生產物，不手動維護。初始化 SQL 的變化不會自動覆蓋已執行遷移的資料庫譯文。

完整說明請參閱 [國際化語言擴充指南](docs/國際化語言擴展指南.md)。

語言包定義系統能夠呈現的語言集合，`base_language` 表只負責執行時啟用狀態、名稱、排序與主語言設定。管理端語言偏好儲存為 `kratos-admin:locale`，uni-app 與 Taro 儲存為 `kratos-app:locale`；所有 HTTP、更新權杖、fetch、SSE、uni.request 與 Taro.request 請求都會傳送規範化的 `Accept-Language`。固定文案由各 workspace 的 core/System JSON 語言包維護，動態選單與字典由後端翻譯表按請求語言解析，缺少目前語言譯文時回退主語言。

新增語言不需要修改 Go、TypeScript 或模組註冊程式碼：在 `backend/internal/i18n/assets` 與三個 workspace 的六個前端語言包目錄中增加同名 JSON，然後執行 `make i18n`。腳本會校驗語言集合、語言鍵與佔位符，並產生六個前端註冊檔案、Element Plus 與 Day.js 對映。語言名稱、排序、啟用狀態與主語言由 `base_language` 資料庫記錄提供；`common.language.*` 用於編譯期離線顯示與產生語言遷移的初始名稱。新增語言的完整檔案清單與遷移流程請參閱 [國際化語言擴充指南](docs/國際化語言擴展指南.md)。需要把語言加入新部署資料庫時，直接更新唯一的 `v0.0.1` 初始化遷移；已有資料庫的啟用狀態不會被遷移覆蓋。

動態資源的主語言由 `base_language.is_primary` 設定。建立或更新選單、字典、字典項目與系統設定時，後端按請求 `Accept-Language` 將輸入文字轉換為主語言寫入主表；請求語言不是主語言時，原文寫入對應翻譯表，其他已啟用非主語言也只儲存在翻譯表。系統設定名稱、選單標題、字典名稱與字典項目標籤支援在管理端點擊名稱開啟翻譯彈窗，文字/富文字設定值支援執行時翻譯回退。

## 發佈

統一發佈會更新並發佈 10 個 npm 套件：

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

`make tag` 會先執行唯讀的 `make i18n-check`，檢查語言包、SQL 翻譯腳本與 OpenAPI 多語言文件；產生物未同步時會直接拒絕發佈，不會在發佈過程中自動翻譯或改寫檔案。隨後發佈腳本要求目前分支為遠端預設分支且與 `origin` 同步，執行後端測試與前端打包，然後推送 `vX.Y.Z`、`backend/vX.Y.Z`、`npm/vX.Y.Z`。`npm/vX.Y.Z` 觸發 `.github/workflows/publish-npm.yml`，透過 npm Trusted Publishing 發佈以上 10 個套件；三個預設宿主均為私有套件，不參與發佈。本機需要可用的 `git`、`gh` 與 GitHub 登入狀態。

只進行本機 npm 發佈時：

```bash
pnpm login
make -C frontend publish
```

發佈腳本會跳過 registry 中已存在的同版本套件，支援失敗後重試。私有 registry 可透過 `NPM_REGISTRY`、`NPM_ACCESS` 與 `NPM_TAG` 覆蓋。

## 文件

| 主題 | 文件 |
| --- | --- |
| 整體架構 | [docs/系統總體設計.md](docs/系統總體設計.md) |
| 新能力接入 | [docs/服務接入指南.md](docs/服務接入指南.md) |
| 資料庫遷移 | [docs/資料庫與初始化資料設計.md](docs/資料庫與初始化資料設計.md) |
| 參數驗證 | [docs/介面參數校驗設計.md](docs/介面參數校驗設計.md) |
| 登入與密碼 | [docs/登入與密碼加密流程.md](docs/登入與密碼加密流程.md) |
| AI 助手 | [docs/AI助手設計.md](docs/AI助手設計.md) |
| 站內信 | [docs/站內信設計.md](docs/站內信設計.md) |
| 管理端元件 | [docs/前端元件清單.md](docs/前端元件清單.md) |
| 國際化設計 | [docs/國際化最終方案.md](docs/國際化最終方案.md) |
| 安全策略與維運工作 | [docs/安全策略與運維任務.md](docs/安全策略與運維任務.md) |
| 新增語言 | [docs/國際化語言擴展指南.md](docs/國際化語言擴展指南.md) |

建立外部專案時，三端 `packages/cli` 獨立產生完整前端，包含語言註冊、宿主生命週期與檢查建置工具。
Go 腳手架只呼叫 npm CLI；`--kratos-project` 適配後端靜態輸出，管理端 CLI 產生共用前端 Makefile 與腳本。
三端支援本地 `system`，不需要臨時模組名稱或產生後補寫檔案。CLI 更新需先發佈到 npm，Go 的精確版本呼叫才能使用新能力。

Backend 的 `NewModules` 與 `NewStreams` 共用宿主注入的 `*backend.CodeGenManager`；透過 `backend.ProviderSet` 自動裝配，手動呼叫這兩個入口時也需傳入同一實例。修改內部依賴裝配後執行 `make -C backend public-wire wire`。

系統管理的基礎管理統一使用「系統設定」入口維護普通設定與表單設定；表單類型按設定 key 載入模組註冊的表單，重用統一查詢與更新介面。

資料庫遷移檔案內建於後端二進位檔；Docker 與後端壓縮包仍包含該目錄用於發佈歸檔。資料庫只儲存遷移檔案引用及校驗值，歷史檔案需要保留；既有正文記錄需在升級前備份轉換。詳見 [後端遷移檔案與記錄](backend/README.md#遷移檔案與記錄)。

前端統一建置使用分階段、按任務分組的普通文字日誌，關閉終端機顏色。管理端自動匯入宣告透過 `make -C frontend types-admin` 明確產生，普通建置不再修改原始碼目錄中的元件宣告。

根目錄、Backend、Frontend 及 CLI 產生的 Makefile 統一關閉遞迴目錄提示與成功任務的快取日誌；失敗任務仍輸出完整錯誤。pnpm、Turbo、Docker 與發佈腳本繼承無顏色環境，開發服務保留實際執行日誌。
