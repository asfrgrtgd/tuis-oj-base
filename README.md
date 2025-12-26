# tuis-oj-base

Docker Compose ひとつで動くオンラインジャッジ基盤（API / Worker / go-judge / フロント）。C / C++ / Python / Java の提出・採点、管理者向け問題インポート、シンプルな Web UI を含みます。

## 主な機能
- 提出を go-judge で自動採点し結果を保存
- Web UI でログイン、問題閲覧、提出、結果確認、管理操作
- 管理者が ZIP で問題をインポート・更新
- 初期管理者とサンプル問題を自動投入（設定で無効化可）

## 構成
- `api`: Gin 製 REST API（セッション + CSRF、マイグレーション/初期管理者作成も担当）
- `worker`: Redis キューから submission_id を処理して go-judge を実行
- `frontend`: Vite + React の開発用 UI
- `go-judge`: 採点エンジン（ルート直下 Dockerfile でビルド）
- `migrations`: DB スキーマと初期問題
- `ドキュメント`: ドキュメント（アーキテクチャ・セットアップ・図ほか）
- `submission-files`, `secrets`, `logs`: 提出・シークレット・ログの保存先

## ローカルで動かす手順

### 前提条件
- **OS**: Linux または WSL2（Windows Subsystem for Linux）
  - Windows の場合は WSL2 をインストールしてください
  - macOS の場合は Docker Desktop をインストールしてください
- **Git**: バージョン管理ツール
- **Docker / Docker Compose**: `docker compose` コマンドが使えること

### 1. リポジトリを取得
```bash
git clone https://github.com/asfrgrtgd/tuis-oj-base.git
cd tuis-oj-base
```

### 2. 環境変数を設定
```bash
cp .env.example .env
cp frontend/.env.example frontend/.env
```
> ローカル開発ならデフォルトで OK。本番では `.env` 内の `SESSION_KEY` と `CSRF_SECRET` を必ず変更してください。

### 3. ファイルパーミッションの設定（Linux/WSL2 のみ）
API/Worker コンテナは `appuser`（uid:65532）で動作します。提出ファイルやログ、シークレットが保存されるディレクトリの所有者を事前に設定してください。

```bash
sudo chown -R 65532:65532 submission-files logs secrets
sudo chmod -R u+rwX submission-files logs secrets
```

> **注**: macOS の Docker Desktop では不要です。

### 4. コンテナをビルド・起動
```bash
docker compose up -d --build
```

### 5. DB マイグレーション（初回のみ）
```bash
POSTGRES_URL="postgres://tuisoj:tuisoj@db:5432/tuisoj?sslmode=disable"
docker compose run --rm --entrypoint migrate \
  -e POSTGRES_URL="$POSTGRES_URL" \
  api -path=/migrations -database "$POSTGRES_URL" up
```

### 6. 動作確認
| サービス | URL |
|---------|-----|
| Web UI (dev) | http://localhost:8080 |

### 7. 初期管理者でログイン
`BOOTSTRAP_ADMIN=true`（デフォルト）の場合、初回起動時に `admin` ユーザーが自動作成されます。
```bash
cat secrets/initial_admin_password.secret
```
このパスワードで http://localhost:8080 からログインできます。

詳細は `ドキュメント/環境構築と使い方.md` を参照。

## 使い方（概要）
- 受験者: ブラウザで UI にアクセスし、ログイン → 問題閲覧 → コード提出 → 結果確認。
- 管理者: 管理画面で問題 ZIP インポート、ユーザー CSV 追加、メトリクス確認。
詳しくは `document/使い方.md` を参照。
