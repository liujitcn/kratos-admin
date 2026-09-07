# backend

Backend は、メッセージ分類、アプリ内メッセージ管理、ユーザー受信トレイ、Redis 配信復旧、管理画面ワークベンチ統計、ファイル資産メタデータ、ログイン元ポリシー、セッション失効、監査イベントの非同期保存、ログ保管期間の整理、制御されたデータベースバックアップを提供します。セキュリティ、メッセージ、オープン認可の初期データは、すべて `v0.0.1` の初期化マイグレーションで提供します。

`backend` は API 契約、Go 生成インターフェース、Service 実装、Biz ビジネス層、タスクスケジューリング、HTTP/gRPC/MCP/AI の登録、必要なデータアクセス処理を保持します。プロセスのエントリーポイントは `internal/cmd/server` です。ルートパッケージは `ProviderSet`、`NewModuleResources`、`NewModules`、`NewTasks`、`NewStreams`、`NewQueueConsumers` を通じて、外部 Core ホストから再利用できる公開境界を提供します。`internal/adapter/core` は Admin が生成したデータベースアクセス機能を `kratos-core/data` の Store/Writer 契約へ適合させ、`internal/module` はモジュール実装だけを保持します。AI Runtime は `internal/biz` に実装され、外部公開の再利用入口は `pkg/agent` です。業務モジュールは `pkg/notification.Publish` でアプリ内メッセージを発行し、内部トランザクションと Dispatch 復旧処理が最終配信を担当します。オープン認可クライアントは単一テーブルの JSON operation ホワイトリストを使用してテナントに紐付けられ、公開エンドポイントはクライアント用 Bearer Token を発行します。HTTP middleware はテナント、状態、IP ホワイトリスト、API 範囲を検証し、HTTP 暗号化 Filter はリクエストのバインド前にクライアントデータを復号し、成功レスポンス後に暗号化します。ログイン認証は TOTP 多要素認証、ワンタイム復旧コード、グローバルおよびテナント/ユーザー単位のログイン元ポリシーに対応します。

## ディレクトリ

```text
backend
├── internal/cmd/server             # Admin の独立起動入口と Wire 合成ルート
├── api
│   ├── proto                         # Proto 契約
│   └── gen/go                        # Buf が生成する Go、HTTP、gRPC、ツールコード
├── internal/biz                      # 業務 Case、DTO、コード生成、補助ドメインコード
├── internal/adapter/core             # Core Store/Writer 契約への Admin 永続化アダプター
├── bootstrap.go                      # ProviderSet、モジュール、タスク、SSE、キュー、リソース入口
├── internal/module                   # Admin と kratos-core の内部モジュール適合とリソース実装
│   ├── module.go                     # Core Module プロトコル登録
│   ├── resources.go                  # module.Module 静的リソース
│   ├── init.go                        # Admin モジュール ProviderSet
│   └── wire.go / wire_gen.go          # 公開入口の内部依存組み立て
├── pkg/agent                         # 再利用可能な AI Runtime、モデル、ツール API
├── pkg/runtimeconfig                 # 実行時設定 Proto の登録、検証、キャッシュ API
├── pkg/notification                  # アプリ内メッセージ発行 API
├── internal/task                     # 非同期タスクと定期タスク実行器
├── internal/server                   # サービス interceptor と API モジュール登録適合
│   └── middleware/{oauth,logstream}  # 業務別の HTTP/gRPC サービス interceptor
├── internal/data/gen                 # GORM が生成するモデル、クエリ、リポジトリ
├── internal/service                  # Proto Service 実装
├── internal/const                    # 業務定数
├── internal/i18n/assets              # 業務言語リソース
├── data                              # フロントエンド H5 生成物と OSS アップロード対象
├── logs                              # 実行ログとログ保存フォールバックファイル
├── backups                           # ローカルバックアップ作業ディレクトリ
├── codegen/restore                   # コード生成復元スナップショット
└── migration                         # コード生成が使用するマイグレーションリソース
```

## 基本フロー

`backend` ディレクトリに移動したら、まずヘルプで全ターゲット、既定値、上書き例を確認します。

```bash
make help
```

初回開発時は Buf、protoc、Wire、gorm-gen、goimports、検査ツールをインストールします。

```bash
make init
```

通常の起動方法です。

```bash
make run
```

`make run` は「protobuf Go -> OpenAPI -> 独立入口 Wire -> サービス起動」の順で必要な生成物を更新します。生成物に変更がないことを確認できた場合は、生成を省略して直接起動できます。

```bash
make run-only
```

バックエンドは MySQL、Redis、Consul、Vault に依存し、接続パラメーターは `configs/data.yaml`、`configs/registry.yaml`、`configs/key.yaml` とそれぞれの環境別ファイルで設定します。起動前に各ミドルウェアへ接続でき、Vault がアンシール済みであることを確認し、ルートキーの読み取り権限を持つ `VAULT_TOKEN` を端末または IDE に設定してください。同じアプリケーションの各ノードは共通のルートキー参照と `scope` を使用します。Vault に接続できない、シールされている、キーが存在しない、または token が無効な場合は起動に失敗し、ローカルファイルへはフォールバックしません。ミドルウェアのデプロイ、初期化、認証情報の管理は実行環境の責務であり、アプリケーション起動とは連動しません。

既定の設定ディレクトリは `./configs`、既定の実行環境は `dev` です。基本設定は `<name>.yaml`、環境差分は `<name>.<env>.yaml` を使用します。環境ファイルが存在する場合は基本設定の後に読み込み、存在しない場合は基本設定へフォールバックします。

セッションのライフサイクルとアップロードのセキュリティスキャンは `authn.session`、`oss.upload_security` を使用します。監査ログの保管は「システム管理 → バックアップ管理 → データアーカイブ」でテーブル単位に管理し、データベースバックアップは「システム管理 → バックアップ管理 → データバックアップ」でデータソース単位に管理します。ログ保存フォールバック設定は非表示設定 `baseLogFallback` を使用します。バックアップ整合性キーと暗号化キーは、タスク実行時にそれぞれ `kratos-admin:backup/integrity`、`kratos-admin:backup/encryption` から実行時キーサービス経由で派生します。通常のシステム設定は「システム設定」ページで管理します。通常の HTTP リクエストは `server.http.timeout` と `server.http.max_body_bytes` のみを使用し、`/events`、`/mcp`、AI メッセージストリームは通常のリクエストタイムアウトを自動的にスキップします。

ローカルファイルストレージのディスクルートは `configs/oss.yaml` の `oss.root_directory` だけで設定し、Core はそのディレクトリを `/data/` にマッピングします。アップロード対象は `業務種別/ファイル分類/年/月/日/ファイル名` で階層化し、データベースには OSS オブジェクトパスを保存します。`backend/data` には三端の H5 生成物とアップロード対象だけを保持し、ログ、バックアップ、コード生成復元スナップショットはそれぞれ `backend/logs`、`backend/backups`、`backend/codegen/restore` に保存します。

多要素認証方式はシステム設定 `securityMfaMethod` で選択し、現在は `totp` と `webauthn` に対応しています。実行時 MFA パラメータは `mfa.yaml` または環境上書きファイル `mfa.dev.yaml` の `mfa` ノードから読み込みます。`mfa.encryption_key` に明示値がある場合はそれを優先し、空の場合は TOTP キーを実際に暗号化・復号する時点で `kratos-kit:mfa/encryption` により実行時キーサービスから派生します。管理端末とアプリ端末で TOTP を無効化するには現在のパスワードと動的パスワードまたは復旧コードが必要です。WebAuthn を無効化するには現在のパスワードと Passkey または復旧コードの検証が必要です。本番環境の実キーをリポジトリやデータベースに保存しないでください。完全なフィールド定義は `kratos-kit/api/proto/config/v1/mfa.proto` に従います。

```bash
make run-only CONF=/path/to/configs
make run-only APP_ENV=prod
make run-only RUN_ARGS='--help'
```

たとえば `APP_ENV=dev` は `data.yaml` の後に `data.dev.yaml` を読み込み、`data.prod.yaml` は無視します。ローカル開発設定は `*.dev.yaml` に統一して保存し、これらのファイルは既定では Git に含めません。

ローカルで HTTPS により HTTP サービスを起動する場合は、リポジトリルートでフロントエンドとバックエンドが共有する開発証明書を生成してから、`https` 実行環境を使用します。

```bash
bash scripts/generate-dev-cert.sh 192.168.1.100
make -C backend run-only APP_ENV=https
```

`APP_ENV=https` は `configs/server.https.yaml` を読み込み、リポジトリルートの `certs/dev-cert.pem` と `certs/dev-key.pem` を使用して、HTTP サービス `:7001` を HTTPS として提供します。アクセス先は `https://localhost:7001` または `https://192.168.1.100:7001` です。この環境上書き設定はコンテナリリースのパスへ直接再利用できません。本番環境では証明書をデプロイディレクトリへマウントし、対応する設定に実際のパスを指定してください。

独立入口は `kratoscore.ProviderSet` と内部モジュール ProviderSet を注入します。Core は HTTP、gRPC、MCP、SSE、キュー、定期タスクのランタイムを統一的に作成・管理します。Admin は 6 種類の完全な監査ログモデルを登録し、自動マイグレーションを担当します。Core は API/ポリシーログを非同期で書き込み、Admin はログイン、操作、データアクセス、権限ログを非同期で書き込みます。

定期タスクは実行前にタスク番号で Redis 分散ロックを取得します。ロックを別インスタンスが保持している場合、定期実行はスキップされ、手動実行はロック競合エラーを返します。Redis ロックの初期化に失敗した場合は警告を記録し、プロセス内メモリロックへフォールバックします。このモードは単一インスタンス専用です。複数インスタンスで運用する場合は、すべてのインスタンスが同じ Redis に接続し、Redis ロックモードで動作することを確認してください。

## 変更後の実行

| 変更対象 | コマンド | 説明 |
| --- | --- | --- |
| Proto 契約 | `make api openapi` | Backend の protobuf Go と OpenAPI ソースを生成します。フロントエンド TypeScript RPC はリポジトリルートの `make -C ../frontend ts` を使用し、`ts-admin`、`ts-uni-app`、`ts-taro-app` も個別に実行できます。 |
| データベース構造 | `make gorm-gen` | 開発データベースを更新してから、`GORM_GEN_CONFIG`、`GORM_GEN_DATABASE`、`GORM_TABLE` に従って生成します。 |
| ProviderSet またはコンストラクタ引数 | `make public-wire wire` | 公開入口の内部組み立てと独立サービス入口をそれぞれ更新します。 |
| README または docs | `make -C .. project-docs` | リポジトリルートから Markdown を収集し、各言語のドキュメントディレクトリを生成します。 |
| 言語パックまたは国際化リソース | `make -C .. i18n` | 国際化はリポジトリルートの共通コマンドで管理します。 |
| Go import エイリアス | `make cli fmt` | `cli` で `kratos-kit/cmd/normalize-go-imports` をインストールし、`fmt` で実行して `goimports` により整形します。 |
| 複数のバックエンド生成元を同時に変更 | `make gen` | GORM、インターフェース、OpenAPI、Wire、整形を順に実行します。開発データベースへ接続できる必要があります。 |

生成物は上記コマンドで更新し、手作業で変更しないでください。

Go import エイリアス正規化コマンドは `kratos-kit/cmd/normalize-go-imports` が提供します。Admin リポジトリにはローカルコピーを保持しません。まずコマンドをインストールしてから整形します。

```bash
make cli
make fmt
```

## 検査

コミット前に Backend の検査を実行します。

```bash
make check
```

`make check` は `make lint` と `make test` を順に実行し、自動整形は行いません。国際化を含む全体検査はリポジトリルートで `make check` を実行してください。整形が必要な場合は先に次を実行します。

```bash
make fmt
```

個別に実行することもできます。

```bash
make lint
make test
```

## ビルド

既定では `linux/amd64`、`CGO_ENABLED=0` の実行ファイルをビルドします。

```bash
make build
```

生成物は `bin/server` です。別のプラットフォームはパラメータで上書きできます。

```bash
make build GOOS=darwin GOARCH=arm64 BINARY=bin/server-darwin-arm64
```

リリースアーカイブ、Docker イメージ、三端の静的リソースはリポジトリルートの手順です。ルートで `make package` または `make docker-build` を実行し、パラメータと実行例はルート README を参照してください。

## 主なパラメータ

| パラメータ | 既定値 | 用途 |
| --- | --- | --- |
| `CONF` | `./configs` | サービス実行設定ディレクトリ。 |
| `APP_ENV` | `dev` | `<name>.<env>.yaml` の環境上書きを選択。 |
| `RUN_ARGS` | 空 | サービスコマンドへ追加する引数。 |
| `CGO_ENABLED` | `0` | Go ビルド時に CGO を有効にするか。 |
| `GOOS` / `GOARCH` | `linux` / `amd64` | ビルド対象プラットフォーム。 |
| `BINARY` | `bin/server` | 実行ファイル出力パス。 |
| `ARCHIVE` | `dist/backend-<os>-<arch>.tar.gz` | Backend バイナリの圧縮パス。ルート `make package` から使用します。 |
| `PUBLIC_WIRE_DIR` | `internal/module` | 公開入口で使用する内部 `wire.go` のディレクトリ。 |
| `WIRE_DIR` | `internal/cmd/server` | 独立入口の `wire.go` ディレクトリ。 |
| `GORM_GEN_CONFIG` | `configs/data.dev.yaml` | GORM 生成に使用するデータソース設定。 |
| `GORM_GEN_DATABASE` | 空 | 任意のデータベース名。既定では設定ファイルから読み込みます。 |
| `GORM_TABLE` | 内蔵テーブル一覧 | GORM 生成対象テーブルをカンマ区切りで指定。 |

## 外部ホストからの再利用

外部 Go プロジェクトが Backend を再利用する場合、自身の Wire 合成ルートに `kratoscore.ProviderSet`、`backend.ProviderSet`、ホスト側の ProviderSet を統合して追加します。

```go
func NewApp(ctx *bootstrap.Context) (*kratos.App, func(), error) {
	panic(wire.Build(
		kratoscore.ProviderSet,
		backend.ProviderSet,
		mergeProviderSet,
	))
}
```

ルートパッケージは `AdminResources`、`AdminModules`、`AdminTasks`、`AdminStreams`、`AdminConsumers` から名前付きの貢献を出力します。ホストの ProviderSet は、それらを他の業務モジュールの貢献とともに Core の最終集合へ明示的に追加します。公開コンストラクタは Core の公開型だけを使用し、外部生成された `wire_gen.go` は `backend/internal` に依存しません。

`backend.NewModules` はホストプロセス単位の実行ログ収集器を初期化します。外部プロジェクトが上記の方法で Backend を接続すると、自身や他の登録済みモジュールが stdout/stderr に出力したログも実行ログのリアルタイムコンソールへ入り、履歴ログファイルはホストのログ設定に従って読み込まれます。

`configs/auth.yaml` の JWT キーに明示値がある場合はそれを優先します。空の場合はサーバーとクライアントが `kratos-kit:authn/jwt` を使用して実行時キーサービスから派生します。設定ファイルの機密値は `ENC[...]` で保存し、平文のキーをコミットしないでください。

管理画面のブラウザは Cookie-only リフレッシュトークン方式を使用します。リフレッシュトークンは Path を限定した HttpOnly Cookie にのみ保存し、アクセストークンはページメモリにのみ保存します。uni-app と Taro はこの方式識別子を送信せず、既存のトークン送信方式を使い続けます。

外部モジュールから AI を利用する場合は `pkg/agent.NewRuntime` で Runtime を作成し、`RuntimeConfig.AdminTools/AppTools` または `Runtime.RegisterTool` で Eino `InvokableTool` を登録します。単純な構造化ツールには `pkg/agent.InferTool` によるパラメータ schema の自動生成を優先します。コメント審査やコンテンツ抽出などの固定フローは `NewChatClient`、`NewStructuredRunner`、`SchemaFor`、マルチモーダル Part コンストラクタを組み合わせて実装でき、`internal` パッケージを参照する必要はありません。権限制御が必要な場合は `ToolAccessChecker` を実装し、権限システムへ接続しない場合は `Checker` を `nil` に保ちます。

外部モジュールから実行時設定を接続する場合は、自身の Proto に設定メッセージを定義し、`pkg/runtimeconfig.Register` でキー、既定値、機密フィールドを登録します。Admin 起動時に `base_config` の非表示設定を統一的に初期化して Redis を更新します。設定 JSON は ProtoJSON のエンコード/デコードと Protovalidate 検証を使用し、検証ルール ID はそのまま国際化メッセージキーとして利用できます。共通の起動設定は `kratos-kit/api` の `config.v1.Bootstrap`、たとえば `authn.session`、`oss.upload_security`、`logger`、`data` を優先的に再利用します。
