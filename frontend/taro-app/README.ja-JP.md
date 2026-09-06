# frontend/taro-app

`frontend/taro-app` は独立した pnpm workspace で、React 18 と Taro 4 により `frontend/uni-app` と同じアプリ機能・画面方針を実装します。H5 と WeChat ミニプログラムに対応し、ホーム、ログイン（TOTP/WebAuthn MFA を含む）、規約、WebView、個人センター、設定、プロフィール、AI アシスタント、アプリ内メッセージ受信トレイを提供します。ショップ、注文、決済、推薦機能は含みません。

## Workspace

```text
frontend/taro-app
├── apps/taro-app                 # 非公開 Taro ホスト
├── packages/core                 # ランタイム、基盤ページ、ビルド runner
├── packages/ui                   # NutUI テーマとアイコン適合
├── packages/modules/system       # 個人センター、設定、プロフィール、AI
├── packages/cli                  # 独立 workspace スキャフォールド
├── scripts                       # package exports 境界検査
├── package.json                  # 共通コマンドと開発依存
├── pnpm-workspace.yaml           # workspace 範囲
└── turbo.json                    # workspace タスク関係
```

依存方向は `apps/taro-app -> packages/modules/* -> packages/ui -> packages/core` に固定します。core は UI や業務モジュールに依存せず、system は core と UI の公開 exports だけを再利用し、ホストはモジュールとプラットフォーム設定を組み立てます。

詳細は [非公開ホスト](apps/taro-app/README.md)、[アプリ基盤](packages/core/README.md)、[UI 基盤](packages/ui/README.md)、[system モジュール](packages/modules/system/README.md)、[CLI](packages/cli/README.md) を参照してください。

## ページの組み立て

`apps/taro-app/src/module-manifest.ts` が唯一のモジュール一覧です。登録順がページ、安定 `viewKey`、アイコンの上書き優先度を決めます。モジュールは `defineKratosTaroModule()` で実行時機能を宣言し、`defineKratosTaroBuildModule()` でビルド時のページ記述を提供します。

ホストがコミットするのは固定の `pages/bootstrap` ページだけです。core runner は開発・ビルド開始前に各モジュールの `src/views/**/*.tsx` を走査し、`components` を除外して page wrapper と設定を生成します。自作 `KratosTabBar` を統一して取り付け、`pages*` ルートからメインパッケージと分包を分け、各モジュールの `src/static` を宿主へ統合します。宿主に既存するファイルを優先し、同名モジュールは後に登録されたものを優先します。ビルド終了後は元の `app.config.ts` を復元し、一時ファイルを削除します。

ホームとマイページは非表示のネイティブ tab ルートとして登録し、`switchTab` で WeChat のネイティブ体験を維持します。通常ページは `navigateTo` を優先し、下位ページは親階層に所属して対応する tab を強調します。異常終了後は次回コマンドが `.kratos-taro-app-pages-state.json` に基づき復元します。H5 と WeChat の開発プロセスは同じ装配トランザクションを保持でき、全プロセス終了後に宿主ファイルを復元します。

## 開発とビルド

```bash
cd frontend/taro-app
pnpm install
pnpm dev:h5
pnpm dev:mp-weixin
pnpm build:h5
pnpm build:mp-weixin
```

上位 Makefile の入口は次のとおりです。

```bash
make -C .. run-taro-app
make -C .. check-taro-app
make -C .. build-taro-app
make -C .. package-taro-app
```

H5 の既定アドレスは `http://localhost:5002` で、`/api` と `/events` は `http://localhost:7001` へプロキシします。LAN の HTTPS 開発ではルートで `bash scripts/generate-dev-cert.sh 192.168.1.100` を実行し、`.env.development-h5.local` に `VITE_APP_HTTPS=true` を設定します。Backend が `APP_ENV=https` の場合は `VITE_APP_API_URL=https://localhost:7001` に変更します。H5 の本番生成物は `backend/data/taro-app`、WeChat 生成物は `apps/taro-app/dist/mp-weixin` に出力します。

設計稿幅は 750 です。uni-app から移行する `rpx` は Taro の設計稿 `px` として記述し、物理ピクセルを維持する元の `px` は Taro に変換させません。固定パスの静的資産は `src/static` に配置し、画面で直接表示する画像は所属パッケージの `static/*` export から静的 import します。

## 国際化

Taro の対応言語は core と System の JSON 言語パックから自動検出し、モジュール登録時にキーとプレースホルダーを検証します。ログイン、ホーム、状態ページ、WebView、個人センター、設定、プロフィール、AI は `t(key)` を使用します。言語設定は `kratos-app:locale` に保存し、安定ルートと業務フィールドは変更しません。

すべての `Taro.request`、ファイルアップロード、SSE リクエストは `Accept-Language` を送信します。動的メニューはバックエンドが解析したタイトルを使用し、訳がない場合は主言語へフォールバックします。新しい言語を追加する場合はバックエンドと 3 workspace の言語パックを同期し、ルートで `make i18n-sync` を実行します。

## RPC 生成

Taro の RPC テンプレートは `backend/api` にあります。

- `buf.taro-app.typescript.gen.yaml` は core RPC を生成します。
- `buf.taro-app.core.typescript.gen.yaml` は system RPC を生成します。

```bash
pnpm generate:rpc
# 同等のコマンド
make -C .. ts-taro-app
```

RPC は生成物であり、手作業で変更しません。

## CLI

```bash
pnpm dlx @liujitcn/kratos-taro-app-cli create my-app
pnpm dlx @liujitcn/kratos-taro-app-cli create shop-app --module shop,order
pnpm dlx @liujitcn/kratos-taro-app-cli create my-app --with @acme/customer-module
```

生成プロジェクトは同じ React/Taro 技術スタック、モジュール一覧、runner トランザクション、H5/WeChat ビルド方式を使用します。ローカルモジュールは `pages`、実行時入口、ビルド時入口を個別に管理できます。

## パッケージと検証

公開 4 パッケージのバージョンは一致させます。

- `@liujitcn/kratos-taro-app-core`
- `@liujitcn/kratos-taro-app-ui`
- `@liujitcn/kratos-taro-app-system`
- `@liujitcn/kratos-taro-app-cli`

```bash
pnpm lint
pnpm tsc
pnpm test
pnpm check:exports
pnpm build:packages
pnpm build:h5
pnpm build:mp-weixin
```

`check:exports` は公開 target、バージョン一致、パッケージ間 import 境界を検証します。`test` はモジュール優先度、ナビゲーション、runner トランザクション、CLI、AI SSE を検証し、`build:packages` は `dist/npm` に 4 つの tarball を生成します。
