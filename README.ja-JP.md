# kratos-admin

[简体中文](README.md) | [繁體中文](README.zh-TW.md) | [English](README.en-US.md) | [日本語](README.ja-JP.md)

`kratos-admin` は、Go + Kratos バックエンド、Vue 管理画面、uni-app アプリ基盤、React/Taro アプリ基盤、バージョン管理されたデータベースマイグレーション、モジュールスキャフォールドを含む、フロントエンドとバックエンドを分離した管理システムのリポジトリです。

## 実装済みの機能

- アカウントとパスワード、CAPTCHA、OAuth、TOTP/WebAuthn 多要素認証、JWT 更新、テナント、Casbin 権限。
- TOTP 多要素認証、ワンタイム復旧コード、`disabled`、`optional`、`all_required` のグローバルポリシー。
- オープン認可クライアント管理：クライアントをテナントに紐付け、資格情報から Bearer Token を取得し、operation interceptor と HTTP 暗号化/復号 Filter でテナント、状態、IP ホワイトリスト、JSON API ホワイトリスト、プロトコル暗号文を検証します。
- ユーザー、ロール、部門、職位、メニュー、辞書、設定、タスク、ファイル資産、ログ、地域、API、マイグレーション記録の管理。
- 管理画面ワークベンチでユーザー/ロール概要、ログイン傾向、ログイン結果、操作アクション分布統計を提供します。
- メッセージ分類、テナント内の対象指定/全員向けアプリ内メッセージ、受信トレイの既読/アーカイブ、Redis 配信復旧、Admin/uni-app/Taro メッセージセンター。
- Proto 駆動の HTTP、gRPC、OpenAPI、Agent Tool、MCP Tool、TypeScript RPC 生成。
- AI セッション、ストリーミングメッセージ、添付ファイル、ツール呼び出し、リトライ、再生成、分岐セッション。
- 管理画面のコード生成設定、プレビュー、生成進捗、復元。
- 実行ログ閲覧：リアルタイムコンソール SSE、履歴ログ検索、レベルとキーワードによる絞り込み、履歴元ファイルのダウンロード。
- ログイン元ポリシー（グローバルおよびテナント/ユーザー指定ルール）、パスワード複雑度ポリシー、ポリシーに基づく複数端末ログイン、独立したセッションタイムアウトと失効、自分のログイン履歴、プラットフォームのオンラインセッション管理、監査ログの非同期保存と保持期間整理、制御された MySQL バックアップ復元タスク。
- マウント可能な Go Core モジュール。バックエンドは `module.Module` を実装し、`Resources` で静的リソースを提供し、起動エントリーポイントから Core にプロトコルサービスを一括登録します。
- 管理画面、uni-app、Taro、バックエンドエラーカタログの言語集合を言語パッケージから自動検出し、動的メニュー、辞書、コード生成は登録済みの全言語に対応します。

このリポジトリには、EC、注文、決済、レコメンドなどの業務モジュールは含まれていません。

## ディレクトリ

| ディレクトリ | 説明 | ドキュメント |
| --- | --- | --- |
| `backend` | Kratos サービス、Proto、GORM、マイグレーション、Core ホスト構成。 | [backend/README.md](backend/README.md) |
| `frontend/admin` | デフォルトホスト、core、System、CLI を含む管理画面 workspace。 | [frontend/admin/README.md](frontend/admin/README.md) |
| `frontend/uni-app` | デフォルトホスト、core、system、CLI を含む uni-app workspace。 | [frontend/uni-app/README.md](frontend/uni-app/README.md) |
| `frontend/taro-app` | デフォルトホスト、core、UI、system、CLI を含む React/Taro workspace。 | [frontend/taro-app/README.md](frontend/taro-app/README.md) |
| `docs` | 現在のアーキテクチャ、運用手順、テーマ別ドキュメント。 | [docs/README.md](docs/README.md) |

オープン認可プロトコルの API 範囲、interceptor の境界、暗号化拡張ポイントについては [docs/开放授权协议设计.md](docs/开放授权协议设计.md) を参照してください。

## 環境

- Go `1.27.0`。
- Node.js `^20.19.0` または `>=22.12.0`。
- pnpm のバージョンは各 workspace の `packageManager` に従います。管理画面は `10.33.4`、uni-app と Taro は `10.13.1` です。
- MySQL、Redis、Consul、Vault。用途と設定入口は以下を参照してください。
- Docker デプロイには利用可能な Docker CLI と Docker daemon が必要です。
- TOTP のバインドを有効にする場合、`mfa.encryption_key` に明示値があればそれを使用し、空の場合は TOTP シークレットを保護する時点で `kratos-kit:mfa/encryption` により実行時キースサービスから派生します。WebAuthn を有効にする場合は `mfa.webauthn.rp_id` と `mfa.webauthn.rp_origins` も設定します。設定ファイルの機密値は `ENC[...]` で保存してください。
- Buf、protoc プラグイン、Wire、gorm-gen はコード再生成時だけ必要で、`make -C backend init` でインストールできます。

`make` または `make help` を実行するとリポジトリ全体のコマンドを確認できます。Backend と Frontend の完全なターゲットは、それぞれ `make -C backend help`、`make -C frontend help` で確認できます。

### ミドルウェア依存関係

| ミドルウェア | 用途 | 設定入口 |
| --- | --- | --- |
| MySQL | 業務データの永続化とデータベースマイグレーション。 | `backend/configs/data.yaml` |
| Redis | キャッシュ、分散ロック、キュー、メッセージ配信。 | `backend/configs/data.yaml` |
| Consul | サービス登録と検出。 | `backend/configs/registry.yaml` |
| Vault | アプリケーションのルートキー管理、設定復号、業務キー派生。 | `backend/configs/key.yaml` |

各環境の接続先とアクセスパラメータは、対応する `<name>.<env>.yaml` で設定します。起動前にミドルウェアへ接続でき、Vault が初期化・アンシール済みで、`VAULT_TOKEN` が必要なルートキーを読み取れることを確認してください。デプロイ、初期化、認証情報の管理は実行環境の責務であり、プロジェクトの起動とは連動しません。本番環境では安全な接続と最小権限の認証情報を使用してください。

## ローカル起動

まずデータベースを作成します。

```sql
CREATE DATABASE kratos_admin CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
```

フロントエンドの依存関係をインストールします。

```bash
make -C frontend init
```

依存関係のディレクトリを完全に再構築する場合：

```bash
make -C frontend reinstall
```

Vault が起動してアンシール済みであることを確認し、端末または IDE で有効な `VAULT_TOKEN` を設定してからバックエンドを起動します。

```bash
make -C backend run APP_ENV=dev
```

`run` は起動に必要な API、OpenAPI、Wire の生成物を先に更新します。生成物に変更がないことを確認できた場合は `make -C backend run-only APP_ENV=dev` で直接起動できます。基本設定は `<name>.yaml`、環境差分は `<name>.<env>.yaml` を使用し、現在の環境ファイルがない場合は基本設定へフォールバックします。完全なターゲット、実行順序、パラメータは [Backend Common Procedures](backend/README.md#常用流程) を参照してください。

すべてのフロントエンド開発環境（管理画面、uni-app/Taro H5、WeChat ミニプログラム）を起動します。

```bash
make -C frontend run
```

端末ごとに起動することもできます。常駐コマンドはそれぞれ別のターミナルで実行してください。

```bash
make -C frontend run-admin
cd frontend/uni-app && pnpm dev:h5
cd ../taro-app && pnpm dev:h5
```

| サービス | デフォルトアドレス |
| --- | --- |
| バックエンド HTTP | `http://localhost:7001` |
| バックエンド HTTPS（`APP_ENV=https`） | `https://localhost:7001` |
| バックエンド gRPC | `localhost:6001` |
| 管理画面 | `http://localhost:8848` |
| uni-app H5 | `http://localhost:5004` |
| Taro H5 | `http://localhost:5002` |

Taro の開発成果物は `frontend/taro-app/apps/taro-app/dist/dev/<platform>`、WeChat ミニプログラムの本番成果物は `dist/build/mp-weixin` にあります。H5 の本番成果物は引き続き `backend/data/taro-app` に出力されます。WeChat 開発者ツールはデフォルトで開発ディレクトリを使用し、リリース時は本番ディレクトリをインポートしてください。

uni-app と Taro H5 のデフォルトポートはそれぞれ `5004` と `5002` で、同時に起動できます。LAN から uni-app にアクセスする場合は `localhost` を開発マシンの LAN IP に置き換えてください。

LAN の HTTPS 連携では、リポジトリルートで共有証明書を生成し、バックエンドを HTTPS 環境で起動します。

```bash
bash scripts/generate-dev-cert.sh 192.168.1.100
make -C backend run-only APP_ENV=https
```

管理画面の開発プロキシはローカル環境ファイルで `VITE_PROXY` のバックエンドアドレスを `https://localhost:7001` に変更します。Taro と uni-app H5 はそれぞれの `.env.development-h5.local` で `VITE_APP_API_URL` を `https://localhost:7001` に変更します。3 端末の開発プロキシはいずれも自己署名証明書の検証をスキップします。

デフォルトのマイグレーションには開発アカウント `super / 112233` と `admin / 112233` が含まれます。デプロイ前にデフォルトパスワード、JWT キー、データベース、Redis の認証情報を変更してください。

## 生成とチェック

```bash
make gen
make check
make build
make -C backend fmt
```

`make gen` は Backend、Frontend、言語パッケージ、OpenAPI の順にリポジトリ全体の生成物を作成します。`make check` は Backend、3 つの frontend workspace、国際化の順にチェックします。ルートの `make build` はバックエンドバイナリと 3 つのフロントエンド H5 ホストをビルドします。すべてのフロントエンド（H5 + WeChat ミニプログラム）は `make -C frontend build`、H5 のみは `make -C frontend build-h5`、すべての npm リリースパッケージは `make -C frontend package` を使用します。

`make -C backend cli` は `kratos-kit/cmd/normalize-go-imports` をインストールし、`make -C backend fmt` はそのコマンドを実行して `goimports` で Backend の全 Go ファイルを整形します。コード生成タスクは `FMT_FILE_LIST`（Backend 相対パスを 1 行に 1 つ）でファイル一覧を受け取り、今回書き換えたファイルだけを整形します。`make -C backend api` は生成終了時にプロトコル生成物の Go import alias を統一し、全量生成とファイル単位の整形による不要な差分を防ぎます。

このツールの実装は `kratos-kit/cmd/normalize-go-imports` に統一され、Admin リポジトリには独立した Make ターゲットを保持しません。

バックエンドはデフォルトで `make -C backend build` により `linux/amd64` バイナリをビルドします。ルートの `make package` は `bin/server` と `configs` を含むバックエンドリリースアーカイブを作成し、同時にすべてのフロントエンド npm パッケージを生成します。対象プラットフォームは `GOOS`、`GOARCH` で上書きできます。

Docker イメージはリポジトリルートのコマンドでビルドします。

```bash
make docker-build IMAGE=kratos-admin TAG=latest
make docker-build-multiarch IMAGE=registry.example.com/kratos-admin TAG=latest
make docker-run IMAGE=kratos-admin TAG=latest APP_ENV=dev
make docker-stop IMAGE=kratos-admin TAG=latest
```

`docker-build` は `DOCKER_PLATFORM` で指定した単一プラットフォームをビルドし、デフォルトは `linux/amd64` です。`docker-build-multiarch` は Docker Buildx で `linux/amd64` と `linux/arm64` を同時にビルドし、デフォルトで `--push` によりイメージレジストリへプッシュします。`DOCKER_PLATFORMS` と `DOCKER_OUTPUT` でプラットフォームと出力方法を上書きできます。ビルドコマンドは Docker を確認してから管理画面、uni-app H5、Taro H5 をビルドし、バックエンドは Docker のマルチステージビルドで対象アーキテクチャ向けにコンパイルします。実行コマンドはホストの `7001/6001` ポートを公開し、`backend/data`、`backend/logs`、`backend/backups`、`backend/configs` をそれぞれ `/app/data`、`/app/logs`、`/app/backups`、`/app/configs` にマッピングします。イメージにはデフォルト設定と 3 端末分の静的リソースが含まれ、起動時はイメージ内の不足設定だけをホストの `backend/configs` に補充し、変更済み設定は上書きしません。その後、そのディレクトリを使用してサービスを起動します。静的サイトは既存のアップロードファイルを消去せずに `backend/data` へ補充され、Core は `oss.root_directory` に従ってローカルオブジェクトを `/data/` へ統一的にマッピングします。完全なビルドパラメータと実行例は本節を参照してください。

`I18N_LOCALES` はカンマ区切りの BCP 47 言語コード一覧です（デフォルトではバックエンド言語パッケージから主言語を除いて自動検出されます）。OpenAPI の対象言語を制御します。`make i18n` で OpenAPI の多言語 YAML を生成できます。オフライン生成には `I18N_OFFLINE=1 make i18n` を使用してください。

`backend/api/gen`、`backend/internal/data/gen`、各フロントエンドパッケージの `src/rpc`、OpenAPI、`wire_gen.go` は生成物であり、手動編集は禁止です。すべてのフロントエンド RPC の Buf 設定は `backend/api` に統一されています。管理画面は `make -C frontend ts-admin`、アプリ端末は `make -C frontend ts-uni-app` と `make -C frontend ts-taro-app`、3 端末は `make -C frontend ts`、全体はルートの `make gen` で生成します。

`make -C backend gen` は `backend/internal/module` の業務モジュール構成と `backend/internal/cmd/server` の独立起動 Wire 成果物を生成します。前者だけを更新する場合は `make -C backend public-wire`、カスタム構成ルートを更新する場合は `WIRE_DIR` に `wire.go` を含むディレクトリを指定します。

外部 Go プロジェクトは `github.com/liujitcn/kratos-admin/backend` が提供する名前付きモジュール、リソース、定期タスク、SSE ストリーム、キューコンシューマーを自身の貢献と統合し、`github.com/liujitcn/kratos-core` に渡してプロトコルサービスとアプリケーションライフサイクルを作成します。外部で生成されたコードは `backend/internal` を参照しません。

Admin の公開 `backend/adapter/core` と `backend/adapter/kit` のコンストラクターはデータベースクライアントだけを受け取り、必要なリポジトリを内部で作成します。それぞれ Core のストレージインターフェースと Kit の redact インターフェースを通じて Wire に参加します。redact resolver はアプリケーションインスタンス単位で注入され、リクエストコンテキストを通じて渡されます。ストレージコールバックは対応するデータベースに紐付けられ、グローバルなデフォルトインスタンスは使用しません。外部プロジェクトは通常の単一モジュール構成を維持でき、追加のインターフェース集合や特殊なホストモジュールは不要です。リポジトリ間のリリース順序は Kit redact と server/grpc、Core、Admin Backend です。

`make i18n` は `openapi.yaml` の生成後に `openapi.en-US.yaml`、`openapi.zh-TW.yaml`、`openapi.ja-JP.yaml` を同期生成します。デフォルトのリソースはバックエンドエラーカタログと管理画面 Core の言語パッケージから取得し、外部リソースは `OPENAPI_I18N_CONTENT="言語=パス"` で渡せます。未一致の文言はデフォルトで自動翻訳され、英語と日本語には Google V1、繁体字中国語には OpenCC を使用します。`I18N_AUTO_LOCALIZE=0` で自動翻訳を無効化できます。ネットワークがない場合は `I18N_OFFLINE=1 make i18n` を使用してください。

## 国際化

通常は `make i18n`（同期、生成、検証）だけを実行します。コミットまたは CI のチェックには `make i18n-check`（読み取り専用）を使用します。新しい言語は `make i18n-add I18N_LOCALE=de-DE` で下書きを生成し、手動確認後に `make i18n` を実行します。OpenAPI は既存の言語パッケージをデフォルトで自動検出し、`I18N_LOCALES` で対象言語を限定することもできます。

| 保守対象 | ソースファイル |
| --- | --- |
| 管理画面、uni-app、Taro の固定 UI 文言 | 各 core と業務モジュールの `src/locales/*.json` |
| バックエンドエラー通知、コード生成テンプレート文言 | `backend/internal/i18n/assets/*.json` |
| メニュー、辞書、設定、タスクなど動的リソースの翻訳 | `backend/migration/assets/v0.0.1/mysql/i18n.*.up.sql`、実行時は `base_i18n` に保存 |
| API ドキュメントのタイトル、説明、フィールド説明 | Proto の中国語説明と `scripts/local_openapi_i18n.py` のローカル用語マッピング |
| 多言語版が提供されているマイグレーション説明とプロジェクト文書 | 対応する `README.<locale>.md` などの文書 |

固定文言を変更する場合は、そのモジュールの全言語に同じ key とプレースホルダーを追加してください。`make i18n` は不足している UI 翻訳を自動補完せず、不足を検出すると失敗します。`src/locales/generated.ts` などの言語登録ファイルと `openapi.<locale>.yaml` は生成物であり、手動で保守しません。初期化 SQL の変更は、実行済みマイグレーションのデータベース翻訳を自動的に上書きしません。

完全な説明は [国際化言語拡張ガイド](docs/国际化语言扩展指南.md) を参照してください。

言語パッケージは、システムが表示できる言語集合を定義します。`base_language` テーブルは実行時の有効状態、名称、順序、主言語設定だけを管理します。管理画面の言語設定は `kratos-admin:locale`、uni-app と Taro は `kratos-app:locale` に保存します。すべての HTTP、リフレッシュトークン、fetch、SSE、uni.request、Taro.request は正規化された `Accept-Language` を送信します。固定文言は各 workspace の core/System JSON 言語パッケージで管理し、動的メニューと辞書はリクエスト言語に応じてバックエンドの翻訳テーブルから解決し、現在の言語の翻訳がない場合は主言語へフォールバックします。

新しい言語の追加に Go、TypeScript、モジュール登録コードの変更は必要ありません。`backend/internal/i18n/assets` と 3 つの workspace にある 6 つのフロントエンド言語パッケージディレクトリへ同名の JSON を追加し、`make i18n` を実行します。スクリプトは言語集合、言語キー、プレースホルダーを検証し、6 つのフロントエンド登録ファイル、Element Plus、Day.js のマッピングを生成します。言語名、順序、有効状態、主言語は `base_language` データベースレコードから提供されます。`common.language.*` はコンパイル時のオフライン表示と、生成される言語マイグレーションの初期名称に使用されます。完全なファイル一覧とマイグレーション手順は [国際化言語拡張ガイド](docs/国际化语言扩展指南.md) を参照してください。新規デプロイのデータベースに言語を追加する場合は、唯一の `v0.0.1` 初期化マイグレーションを直接更新します。既存データベースの有効状態はマイグレーションで上書きされません。

動的リソースの主言語は `base_language.is_primary` で設定します。メニュー、辞書、辞書項目、システム設定を作成または更新するとき、バックエンドはリクエストの `Accept-Language` に従って入力テキストを主言語へ変換し、主テーブルへ保存します。リクエスト言語が主言語でない場合は原文を対応する翻訳テーブルへ保存し、その他の有効な非主言語も翻訳テーブルだけに保存します。管理画面ではシステム設定名、メニュータイトル、辞書名、辞書項目ラベルの名前をクリックして翻訳ダイアログを開けます。テキスト/リッチテキスト設定値は実行時翻訳フォールバックに対応します。

## リリース

統合リリースでは次の 10 個の npm パッケージを更新・公開します。

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

`make tag` は最初に読み取り専用の `make i18n-check` を実行し、言語パッケージ、SQL 翻訳スクリプト、OpenAPI 多言語ドキュメントを検査します。生成物が同期していない場合はリリースを拒否し、リリース中に自動翻訳やファイル書き換えは行いません。その後、リリーススクリプトは現在のブランチがリモートのデフォルトブランチで `origin` と同期していることを要求し、バックエンドテストとフロントエンドのパッケージ化を実行し、`vX.Y.Z`、`backend/vX.Y.Z`、`npm/vX.Y.Z` を push します。`npm/vX.Y.Z` は `.github/workflows/publish-npm.yml` を起動し、npm Trusted Publishing で上記 10 パッケージを公開します。3 つのデフォルトホストはプライベートパッケージのため公開対象外です。ローカルでは利用可能な `git`、`gh`、GitHub ログイン状態が必要です。

ローカル npm 公開のみを行う場合：

```bash
pnpm login
make -C frontend publish
```

公開スクリプトは registry にすでに存在する同じバージョンのパッケージをスキップし、失敗後のリトライに対応します。プライベート registry は `NPM_REGISTRY`、`NPM_ACCESS`、`NPM_TAG` で上書きできます。

## ドキュメント

| テーマ | ドキュメント |
| --- | --- |
| 全体アーキテクチャ | [docs/系统总体设计.md](docs/系统总体设计.md) |
| 新機能の接続 | [docs/服务接入指南.md](docs/服务接入指南.md) |
| データベースマイグレーション | [docs/数据库与初始化数据设计.md](docs/数据库与初始化数据设计.md) |
| パラメータ検証 | [docs/接口参数校验设计.md](docs/接口参数校验设计.md) |
| ログインとパスワード | [docs/登录与密码加密流程.md](docs/登录与密码加密流程.md) |
| AI アシスタント | [docs/AI助手设计.md](docs/AI助手设计.md) |
| アプリ内メッセージ | [docs/站内信设计.md](docs/站内信设计.md) |
| 管理画面コンポーネント | [docs/前端组件清单.md](docs/前端组件清单.md) |
| 国際化設計 | [docs/国际化最终方案.md](docs/国际化最终方案.md) |
| セキュリティポリシーと運用タスク | [docs/安全策略与运维任务.md](docs/安全策略与运维任务.md) |
| 言語の追加 | [docs/国际化语言扩展指南.md](docs/国际化语言扩展指南.md) |

外部プロジェクトを作成するとき、3 端末の `packages/cli` は言語登録、ホストライフサイクル、チェック・ビルドツールを含む完全なフロントエンドを独立して生成します。
Go スキャフォールドは npm CLI だけを呼び出します。`--kratos-project` はバックエンドの静的出力に適応し、管理画面 CLI は共通フロントエンド Makefile とスクリプトを生成します。
3 端末はローカルの `system` をサポートし、一時的なモジュール名や生成後のファイル追記は必要ありません。CLI の更新は先に npm へ公開し、Go の正確なバージョン呼び出しで新機能を使用できるようにしてください。

Backend の `NewModules` と `NewStreams` はホストから注入された `*backend.CodeGenManager` を共有します。`backend.ProviderSet` が自動的に組み立てますが、これら 2 つの入口を手動で呼び出す場合も同じインスタンスを渡してください。内部依存の組み立てを変更した後は `make -C backend public-wire wire` を実行します。

システム管理の基礎管理は「システム設定」入口に統一し、通常設定とフォーム設定を管理します。フォームタイプは設定キーでモジュール登録済みフォームを読み込み、統一された検索・更新 API を再利用します。

データベースマイグレーションファイルはバックエンドバイナリに内蔵されます。Docker とバックエンドの圧縮アーカイブにはリリースアーカイブ用として引き続きこのディレクトリが含まれます。データベースにはマイグレーションファイルの参照とチェックサムだけを保存するため、履歴ファイルは保持してください。既存の本文レコードはアップグレード前にバックアップして変換する必要があります。詳しくは [Backend Migration Files and Records](backend/README.md#迁移文件与记录) を参照してください。

フロントエンドの統一ビルドは、タスク単位でグループ化した段階的な通常テキストログを使用し、ターミナルの色を無効にします。管理画面の自動インポート宣言は `make -C frontend types-admin` で明示的に生成し、通常のビルドではソースディレクトリ内のコンポーネント宣言を変更しません。

ルート、Backend、Frontend、CLI が生成する Makefile は、再帰ディレクトリの通知と成功タスクのキャッシュログを一律に無効化します。失敗したタスクは完全なエラーを出力します。pnpm、Turbo、Docker、リリーススクリプトは色なし環境を継承し、開発サービスは実際の実行ログを保持します。
