# frontend/admin

管理端末は pnpm workspace を使用し、「薄いホスト + core 基盤 + 任意の業務モジュール + 開発ツール」で構成します。ホストは組み立てと起動だけを担当し、ページ、リクエスト、RPC 型、業務依存は対応するモジュールパッケージに所属します。System モジュールはメッセージ管理、メッセージ分類、個人受信トレイを提供し、トップツールに未読数を表示します。Core レイアウトはアバターメニューから画面ロックを提供し、ロックパスワードの要約は現在のロックセッション中だけ保持します。

依存方向は `app -> business module -> core` に固定します。core は業務モジュールに依存せず、業務モジュール同士も原則として相互参照しません。再利用が必要な場合は、相手の `package.json#exports` が公開する Interface だけを使用します。

## ディレクトリの責務

```text
frontend/admin
├── apps/admin                        # 現在の全 module を組み立てる既定ホスト
├── packages/core                     # @liujitcn/kratos-admin-core 基盤
│   ├── src/api/{base/v1,system/admin/v1} # Proto 完全パスで整理した基盤 API
│   ├── src/components                # 共通コンポーネント
│   ├── src/layouts                   # レイアウト
│   ├── src/modules                   # モジュール登録 interface
│   ├── src/rpc                       # ログイン、メニュー、基盤機能の Proto 型
│   └── src/views                     # ログインと静的エラーページ
├── packages/modules/system           # @liujitcn/kratos-admin-system
│   └── src/{api,config,rpc,utils,views} # Proto 階層で管理する API と RPC
├── packages/cli                      # @liujitcn/kratos-admin-cli
│   └── templates/business-workspace  # 完全な pnpm workspace テンプレート
├── internal/vite-config              # 現在のホストのビルド設定
├── internal/tsconfig                 # 共通 TypeScript 設定
├── internal/lint-config              # 共通 Oxlint 設定
├── scripts/build-package.mjs         # core と業務モジュールの npm ビルダー
├── package.json                       # workspace コマンドと共通開発依存
├── pnpm-workspace.yaml
├── tsconfig.json                      # workspace TypeScript パス設定
└── turbo.json
```

`apps/admin/src/module-manifest.ts` は既定ホストの module 設定の唯一の情報源で、実行時ローダー、Vite の走査対象、事前ビルド依存を宣言します。`apps/admin/src/modules.ts` は現在のホストの全 module を読み込み、既定エクスポートします。core の業務ビューにはログインページだけを保持し、403、404、500、Pending は core の既定静的実装を使用します。個人センター、AI アシスタント、システム管理画面は `systemAdminModule` が提供します。個人センターと AI アシスタントのルートはバックエンドメニューから動的登録され、ホストに静的業務ルートを宣言しません。

## ルートファイル

| パス | 役割 |
| --- | --- |
| `apps/` | 実行可能な管理端末ホストの集合。既定ホストは `admin` |
| `packages/core/` | 公開可能な管理端末ランタイム、レイアウト、コンポーネント、基盤ページ |
| `packages/modules/` | 任意の業務モジュール。API とページをモジュール内で管理 |
| `packages/cli/` | 公開可能な業務 workspace CLI とテンプレート |
| `internal/` | 現在のソースリポジトリ専用の Vite、TypeScript、lint 設定 |
| `scripts/build-package.mjs` | npm ソースコピーと TypeScript 宣言を生成し、core 内部 alias を変換 |
| `package.json` | workspace コマンド、ツール依存、Node、pnpm バージョン |
| `pnpm-lock.yaml` | workspace 全体の依存解決結果 |
| `pnpm-workspace.yaml` | workspace パッケージの範囲 |
| `tsconfig.json` | 共通設定とソースパスマッピング |
| `turbo.json` | 開発、ビルド、公開ビルド、型検査のタスク関係 |
| `AGENTS.md` | 管理端末の協業とコード制約 |

`package.json` を含む各サブディレクトリには同階層の `README.md` があり、パッケージ内のファイルとディレクトリの責務を説明します。

管理端末は `@liujitcn/kratos-admin-core`、`@liujitcn/kratos-admin-system`、`@liujitcn/kratos-admin-cli` を公開します。既定ホスト `@liujitcn/kratos-admin` は非公開で、npm のビルド・公開一覧には含めません。

## 開発とビルド

```bash
cd frontend/admin
pnpm install
pnpm dev
pnpm test
pnpm type:check
pnpm lint:oxlint
pnpm build
pnpm build:package
```

`frontend/admin` から上位 Makefile の共通フローも実行できます。

```bash
make -C .. run-admin
make -C .. check-admin
make -C .. build-admin
make -C .. package-admin
```

既定ホストは `http://localhost:8848` です。環境変数は `apps/admin/.env*` にあり、開発 API プロキシと本番ビルド出力はホストの Vite 設定が管理します。本番ビルドは `backend/data/admin` に出力します。

管理端末のログインパスワードは安全なコンテキストでは Web Crypto で暗号化します。LAN の HTTP アドレスでアクセスし、ブラウザに Web Crypto がない場合は純粋な JavaScript 実装へフォールバックし、バックエンドのパスワード暗号文プロトコルは維持します。このフォールバックは実行互換性だけを解決し、HTTP では Token の漏えいや能動的改ざんが起こり得るため、本番では HTTPS を使用してください。

LAN IP で HTTPS 開発サービスを有効にする場合は、リポジトリルートで証明書を生成します。

```bash
cd ../..
bash scripts/generate-dev-cert.sh 192.168.1.100
```

IP は実際の LAN IP に置き換えます。スクリプトはルートの `certs` に共有証明書を生成します。`apps/admin/.env.development.local` に `VITE_HTTPS=true` を追加し、`pnpm dev` を再起動して `https://192.168.1.100:8848` にアクセスします。初回はブラウザで自己署名証明書を信頼し、他の端末からアクセスする場合も各端末で証明書を信頼する必要があります。Backend、Taro、uni-app は同じ証明書ディレクトリを再利用できます。

管理端末の認証状態はブラウザの Cookie-only 方式です。refresh token は Path を限定した HttpOnly Cookie に保存し、access token はページメモリだけに保存して `localStorage` や `sessionStorage` には書き込みません。アプリ起動時に旧バージョンの永続 access token を削除し、機密情報を含まない有効期限 Cookie で静かなセッション復元が必要か判断します。

## 国際化

管理端末の対応言語は core と System の JSON 言語パックから自動検出し、モジュール登録時に言語キーとプレースホルダーを検証します。ログインページとトップツールは同じ locale store を共有し、言語切替でページを再読み込みせず、現在のルート、クエリ、未送信フォームを保持します。

言語設定は `kratos-admin:locale` に保存します。Axios、refresh token、fetch、SSE、Swagger のリクエストは統一して `Accept-Language` を送信します。動的メニューと辞書はバックエンドが locale に従って返し、現在の言語の訳がない場合は主言語へフォールバックします。新しい言語を追加する場合は、バックエンドと 3 つの workspace の言語パックを同期してから、ルートで `make i18n-sync` を実行します。登録ファイルと Day.js マッピングは生成物です。詳細は [国際化言語拡張ガイド](../../docs/国际化语言扩展指南.md) を参照してください。

API は `api/base/v1`、`api/system/admin/v1` のように Proto 完全パスで整理し、`src/api` にはサービスファイルと同名のリクエストラッパーだけを置きます。実行時設定と内部補助実装はそれぞれ `src/config`、`src/utils` に置きます。RPC は `rpc/base/v1`、`rpc/system/admin/v1` のように完全な Proto 階層を保持します。RPC 型は実際の利用者に所属させ、core はログイン、メニュー、ユーザー情報、起動時機能を保持し、System はシステム管理、個人センター、AI と依存型を自己完結させます。Proto 変更後はルートで `make -C frontend ts-admin` を実行し、core と System の RPC を再生成します。3 端すべてを一度に生成する場合は `make -C frontend ts` を実行します。サーバー契約の細分化が未完了の場合、生成ファイルに現在のパッケージが呼び出さないメソッドが一時的に含まれることがありますが、生成ファイルは手書きしません。

core 内部のソースは `@/*`、業務モジュールは `@liujitcn/kratos-admin-core/*` と自身のパッケージ名を使用します。モジュール間のページ遷移は Vue Router を使用し、ディレクトリを越える相対 import によるコード再利用は禁止します。

## モジュール Interface

core は `bootstrapAdminApp`、`defineAdminModule`、`setAdminDocumentTitle`、ビュー登録表、トップツール、ユーザーメニュー、ルート拡張を公開します。ブラウザタイトルは公共設定 API の `sysName` を優先し、取得に失敗した場合はホスト環境変数をフォールバックにします。業務モジュールの入口は名前とページローダーを宣言します。

```ts
import type { Component } from "vue";
import { defineAdminModule } from "@liujitcn/kratos-admin-core";

const views = import.meta.glob<{ default: Component }>("./views/**/*.vue");

export const orderAdminModule = defineAdminModule({
  name: "order",
  views
});
```

業務ページは module 接頭辞付きパスだけを登録します。`views/list/index.vue` は `order/list/index` として解決され、接頭辞のない別名は提供しません。バックエンドメニューの `component` は完全な module パスを指定します。異なる module に同名の `views` があっても相互に上書きしません。ホストは `src/module-manifest.ts` で module を宣言し、`src/modules.ts` で読み込んで全 module を既定エクスポートします。

### 静的ページの置換

core は `ADMIN_STATIC_VIEWS` で固定ビューキーを公開します。

| 属性 | ビューキー |
| --- | --- |
| `LOGIN` | `login/index` |
| `FORBIDDEN` | `error/403` |
| `NOT_FOUND` | `error/404` |
| `SERVER_ERROR` | `error/500` |
| `PENDING` | `error/pending` |

業務モジュールは `staticViews` で固定ビューキーへ明示的にマッピングできます。後から登録した module が先の実装を置き換えます。通常の `views` は常に module 名で分離され、置換可能なのは `staticViews` だけです。core の npm Interface はモジュール接続、コンポーネント許可リスト、`request`、`navigation`、`table`、`security`、`stores/runtime` の安定入口だけを公開します。業務モジュールは core の内部ソースパスへ依存しません。

## 業務プロジェクトの作成

CLI が生成する業務プロジェクトも pnpm workspace で、独立ホストと公開可能な業務モジュールパッケージを含みます。

```bash
pnpm dlx @liujitcn/kratos-admin-cli create shop-admin --module shop
pnpm dlx @liujitcn/kratos-admin-cli create shop-admin --module shop,order

# 現在のリポジトリで開発
pnpm module:create ../shop-admin --module shop
pnpm module:create ../shop-admin --module shop,order
pnpm module:create ../shop-admin --module shop --module order
```

生成結果は次の構成です。

```text
shop-admin
├── apps/admin
│   └── README.md
├── packages/modules/shop
│   └── README.md
├── packages/modules/order
│   └── README.md
├── scripts/build-package.mjs
├── package.json
├── pnpm-workspace.yaml
├── README.md
├── tsconfig.json
└── turbo.json
```

CLI は既定で `@liujitcn/kratos-admin-system` を読み込み、その後 `--module` の順に自前 module を作成します。`--module` は複数回指定でき、カンマ区切りも使用できます。`--with` は公開済みの追加 module をホストへ加えるだけで、ソースは作成せず、業務 module 間の暗黙依存も作りません。CLI は既存ディレクトリの上書きを拒否します。
