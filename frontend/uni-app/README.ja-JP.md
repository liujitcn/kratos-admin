# frontend/uni-app

`frontend/uni-app` は独立した pnpm workspace で、実行可能な uni-app ホスト、公開可能なアプリ基盤、system 業務モジュール、プロジェクト CLI を提供します。技術スタックは `uni-app + Vue 3 + TypeScript + Vite + Pinia + Sass` で、H5 と WeChat ミニプログラムに対応します。

管理端末のレイヤー思想だけを再利用し、`frontend/admin` のソースや workspace には依存しません。ホーム、ログイン（TOTP/WebAuthn MFA を含む）、規約、WebView、個人センター、設定、プロフィール、AI アシスタント、アプリ内メッセージ受信トレイを提供し、ショップ、注文、決済、推薦は含みません。

## Workspace

```text
frontend/uni-app
├── apps/uni-app                   # @liujitcn/kratos-uni-app、既定ホスト
├── packages/core                  # @liujitcn/kratos-uni-app-core
├── packages/modules/system        # @liujitcn/kratos-uni-app-system
├── packages/cli                   # @liujitcn/kratos-uni-app-cli
├── scripts                        # package exports 境界検査
├── README.md
├── package.json
├── pnpm-workspace.yaml
└── turbo.json
```

依存方向は `apps/uni-app -> packages/modules/system -> packages/core` に固定します。core は認証、リクエスト、設定、Pinia、共通設定ページ、基盤ページ、状態ページ、動的ナビゲーション、ビルドプラグインを担当します。system は個人センター、プロフィール、設定 wrapper、AI を担当し、core の公開 exports だけを再利用します。ホストは入口、manifest、モジュール一覧、bootstrap、Vite 設定だけを管理します。workspace ルートには旧単体アプリの `src` や空テンプレートを残しません。

各パッケージの詳細は [ホスト](apps/uni-app/README.md)、[アプリ基盤](packages/core/README.md)、[system](packages/modules/system/README.md)、[CLI](packages/cli/README.md) を参照してください。

## ページの組み立て

ホストは `apps/uni-app/src/module-manifest.ts` で唯一のモジュール一覧を管理します。登録順が静的ビューの上書き優先度を決めます。各モジュールは `defineKratosAppModule()` を使用し、`pages`、`views`、`icons` を宣言します。`pages` はページ設定、`views` は安定 `viewKey` と物理ページの対応、`icons` は動的ナビゲーションのアイコンを表します。

ビルドプラグインは各モジュールの `src/views/**/*.vue` を走査し、`components` を除外します。後から登録されたモジュールは同じ物理ルートまたは `viewKey` を置き換えられます。ホストの `pages.json` は固定 bootstrap ページだけを保持し、開発・ビルド時に wrapper、ページ設定、静的資産を一時生成して統合します。通常終了時は復元し、強制終了後は `.kratos-uni-app-pages-state.json` から復旧します。同じホストで H5 とミニプログラムのプロセスを同時に実行する場合、ページ装配トランザクションを占有できるのは 1 プロセスです。

## 動的ナビゲーション

既定メニュー API は管理端末の `base-menu` サービスと同じです。

```text
GET /api/v1/app/base/menu
```

サービスパスは `/v1/app/base/menu` で、共通 `/api` base と結合します。`base_menu.id = 99000000` は非表示のモバイル固定ルートです。バックエンドが有効ページをフラットに返し、core が `parent_id` からメニュー木を作ります。`setAppNavigationAdapter()` で独自データソースも接続できます。

メニューの `path` は `app/` 接頭辞を使用し、`name` は `App` 接頭辞を使用します。タイトルとアイコンは `meta.title`、`meta.icon`、モバイル設定は `meta.app` に統一します。`meta.app.view_key` は登録済みでなければならず、任意のコンポーネントパスを API から受け付けません。アクセス方式は `PUBLIC`、`GUEST_ONLY`、`AUTHENTICATED` に対応します。ルート直下の二次ページは tab として扱い、ホームは `99010000`、マイページは `99090000`、`99020000`～`99080000` は予約領域です。

匿名状態とログイン状態でナビゲーションを別々にキャッシュします。新設定は全体を検証してから原子的に切り替え、リモート失敗時は現在の身份の最後の成功キャッシュ、キャッシュがなければローカル既定メニューを使用します。

## 国際化

uni-app の対応言語は core と System の JSON パックから自動検出し、モジュール登録時にキーとプレースホルダーを検証します。ログイン、ホーム、状態ページ、WebView、個人センター、設定、プロフィール、AI は `t(key)` を使用します。言語設定は `kratos-app:locale` に保存し、安定ルートと業務フィールドは変更しません。

すべての `uni.request`、ファイルアップロード、SSE は `Accept-Language` を送信します。動的メニューはバックエンドのタイトルを使用し、訳がない場合は主言語へフォールバックします。言語追加時はバックエンドと 3 workspace の言語パックを同期し、ルートで `make i18n-sync` を実行します。

ページ wrapper は自作 `KratosTabBar` を統一的に取り付けます。ホームとマイページは非表示のネイティブ tab として登録し、`switchTab` で WeChat の体験を維持します。通常ページは `navigateTo` を優先し、下位ページは親 tab を強調します。

## 開発とビルド

```bash
cd frontend/uni-app
pnpm install
pnpm dev:h5
pnpm dev:mp-weixin
pnpm build:h5
pnpm build:mp-weixin
```

H5 の既定アドレスは `http://localhost:5004` です。LAN HTTPS ではルートで `bash scripts/generate-dev-cert.sh 192.168.1.100` を実行し、`.env.development-h5.local` に `VITE_APP_HTTPS=true` を設定します。Backend が `APP_ENV=https` の場合は API URL を `https://localhost:7001` に変更します。H5 の生成物は `backend/data/uni-app`、WeChat 生成物は `apps/uni-app/dist/build/mp-weixin` に出力します。

## RPC 生成

uni-app の RPC 設定は `backend/api` にあります。

- `backend/api/buf.app.typescript.gen.yaml` は core RPC を生成します。
- `backend/api/buf.app.core.typescript.gen.yaml` は system RPC を生成します。

```bash
pnpm generate:rpc
# 同等のコマンド
make -C .. ts-uni-app
```

コマンドは `packages/core/src/rpc` と `packages/modules/system/src/rpc` を再生成します。RPC は生成物であり、手作業で変更しません。

## CLI

```bash
pnpm dlx @liujitcn/kratos-uni-app-cli create my-app
pnpm dlx @liujitcn/kratos-uni-app-cli create my-app --module orders
pnpm dlx @liujitcn/kratos-uni-app-cli create my-app --with @acme/pay
```

既定では system を含み、`--module` は workspace 内のローカルモジュールを作成し、`--with` は公開済みパッケージを追加します。既存ディレクトリは上書きしません。生成ホストには Vue 3 の `createSSRApp` 入口、モジュール一覧、bootstrap、manifest、Vite、TypeScript、H5 HTML が含まれます。

## パッケージと検証

公開 3 パッケージのバージョンは一致させます。

- `@liujitcn/kratos-uni-app-core`
- `@liujitcn/kratos-uni-app-system`
- `@liujitcn/kratos-uni-app-cli`

```bash
pnpm lint
pnpm tsc
pnpm test
pnpm check:exports
pnpm build:packages
```

`check:exports` は export target、パッケージ間ソース import、相対パス境界を検査します。`build:packages` は `dist/npm` に 3 つの tarball を生成します。完全な接続順序は [サービス統合ガイド](../../docs/服务接入指南.md) を参照してください。
