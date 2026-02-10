# --- ビルドステージ ---
FROM golang:1.23-alpine AS builder

WORKDIR /app

# 依存関係のダウンロード
COPY go.mod go.sum ./
RUN go mod download

# ソースコードのコピーとビルド
COPY . .
# main.goがある場所を指定してビルド（バイナリ名をserverとする）
RUN go build -o server main.go

# --- 実行ステージ ---
FROM alpine:latest

WORKDIR /app

# ビルドしたバイナリをコピー
COPY --from=builder /app/server .

# ポートの公開（NeoShowcaseの設定と合わせる）
EXPOSE 8080

# 実行
CMD ["./server"]