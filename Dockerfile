FROM golang:1.22

WORKDIR /app

# 安装 TDLib 依赖
RUN apt-get update && apt-get install -y \
    git cmake g++ wget zlib1g-dev openssl libssl-dev \
    && rm -rf /var/lib/apt/lists/*

# 安装 go-tdlib
RUN go install github.com/zelenin/go-tdlib/cmd/tdjson@latest

# 拷贝代码
COPY . .

# 下载依赖
RUN go mod tidy

# 构建可执行文件
RUN go build -o news_bot main.go

# 容器启动命令
CMD ["./news_bot"]