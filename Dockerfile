# 多阶段构建 - 第一阶段：构建环境
FROM ubuntu:24.04 AS builder

# 设置环境变量
ENV TZ=Asia/Shanghai
ENV DEBIAN_FRONTEND=noninteractive
ENV CURL_VERSION=8.11.0
ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64

# 安装Go和构建依赖
RUN apt-get update && apt-get install -y \
    # Go语言环境
    wget \
    # 构建工具
    build-essential \
    pkg-config \
    # libcurl构建依赖
    libssl-dev \
    zlib1g-dev \
    libnghttp2-dev \
    libpsl-dev \
    libidn2-dev \
    # 其他构建依赖
    ca-certificates \
    git \
    && rm -rf /var/lib/apt/lists/*

# 安装Go 1.25
RUN wget https://go.dev/dl/go1.25.0.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go1.25.0.linux-amd64.tar.gz \
    && rm go1.25.0.linux-amd64.tar.gz

# 设置Go环境变量
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"
ENV GOCACHE="/go/cache"
ENV CGO_ENABLED=1
ENV GOPROXY=https://goproxy.cn,direct
ENV GO_PROXY=https://goproxy.cn,direct

# 设置工作目录
WORKDIR /workspace

# 复制源代码
COPY . .

# 下载Go依赖
RUN go mod download

# 验证依赖
RUN go mod verify

# 构建应用
RUN go build -o main .

# 验证构建结果
RUN ls -la main && \
    echo "构建完成！" && \
    echo "二进制文件: /workspace/main" && \
    echo "文件大小: $(du -h /workspace/main | cut -f1)" && \
    echo "文件类型: $(file /workspace/main)"

# 第二阶段：运行时环境
FROM ubuntu:24.04 AS runtime

# 设置环境变量
ENV TZ=Asia/Shanghai
ENV DEBIAN_FRONTEND=noninteractive
ENV CURL_VERSION=8.11.0

# 安装运行时依赖
RUN apt-get update && apt-get install -y \
    # 运行时依赖
    libssl3 \
    libnghttp2-14 \
    libpsl5 \
    zlib1g \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# 创建非 root 用户
RUN useradd -r -u 1001 -g root appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件和配置文件
COPY --from=builder /workspace/main /app/main
COPY --from=builder /workspace/config.yml /app/config.yml
COPY --from=builder /workspace/config/default.yml /app/default.yml

# 创建logs目录
RUN mkdir -p /app/logs

# 设置文件权限
RUN chmod +x /app/main \
    && chown -R appuser:root /app

# 切换到非 root 用户
USER appuser

# 健康检查（可选）
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ps aux | grep main || exit 1

# 设置入口点，同时输出到控制台和日志文件
ENTRYPOINT ["sh", "-c", "./main -config config.yml 2>&1 | tee logs/log"]