# ==================================
# 開発ステージ
# ==================================
FROM node:20-alpine@sha256:643e7036aa985317ebfee460005e322aa550c6b6883000d01daefb58689a58e2 AS development
WORKDIR /app

# package.json と package-lock.json をコピー
COPY package*.json ./

# 依存関係をインストール
RUN npm ci

# ソースコードをコピー
COPY . .

# ポート 8080 を公開
EXPOSE 8080

# 開発サーバーを起動
CMD ["npm", "run", "dev", "--", "--host"]

# ==================================
# ビルドステージ
# ==================================
FROM node:20-alpine@sha256:643e7036aa985317ebfee460005e322aa550c6b6883000d01daefb58689a58e2 AS builder
WORKDIR /app

# package.json と package-lock.json をコピー
COPY package*.json ./

# 依存関係をインストール
RUN npm ci

# ソースコードをコピー
COPY . .

# 本番ビルドを実行
RUN npm run build

# ==================================
# 配布用ステージ（静的ファイルのみ格納）
# ==================================
FROM alpine:3.20@sha256:765942a4039992336de8dd5db680586e1a206607dd06170ff0a37267a9e01958 AS production
WORKDIR /app

# dist をコピーしてボリューム(/usr/share/nginx/html)へ流し込むだけの軽量コンテナ
COPY --from=builder /app/dist /dist
CMD ["sh", "-c", "cp -r /dist/. /usr/share/nginx/html && exec tail -f /dev/null"]
