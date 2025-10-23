FROM golang:1.22

WORKDIR /app

# 安装依赖
RUN apt-get update && apt-get install -y \
    git cmake g++ wget zlib1g-dev openssl libssl-dev \
    && rm -rf /var/lib/apt/lists/*

# 下载并编译 TDLib
RUN git clone https://github.com/tdlib/td.git /td && \
    mkdir /td/build && cd /td/build && \
    cmake -DCMAKE_BUILD_TYPE=Release .. && \
    cmake --build . --target install

# 设置 CGO 环境变量，让 Go 能找到 TDLib
ENV CGO_CFLAGS="-I/usr/local/include"
ENV CGO_LDFLAGS="-L/usr/local/lib -ltdjson"

# 拷贝代码
COPY . .

# 下载依赖
RUN go mod tidy

# 构建 Go 可执行文件
RUN go build -o news_bot main.go

CMD ["./news_bot"]