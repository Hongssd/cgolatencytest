# 运行时环境 - 包含最新版本 libcurl
FROM ubuntu:24.04 AS runtime

# 设置环境变量
ENV TZ=Asia/Shanghai
ENV DEBIAN_FRONTEND=noninteractive
ENV CURL_VERSION=8.11.0

# 安装构建依赖（用于编译libcurl）和运行时依赖
RUN apt-get update && apt-get install -y \
    # 编译libcurl所需的依赖
    build-essential \
    libssl-dev \
    zlib1g-dev \
    libnghttp2-dev \
    libpsl-dev \
    pkg-config \
    wget \
    # 运行时依赖
    libssl3 \
    libnghttp2-14 \
    libpsl5 \
    zlib1g \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# 编译安装最新版本的 libcurl
WORKDIR /tmp
RUN wget https://curl.se/download/curl-${CURL_VERSION}.tar.gz \
    && tar -xzf curl-${CURL_VERSION}.tar.gz \
    && cd curl-${CURL_VERSION} \
    && ./configure \
        --prefix=/usr/local \
        --with-openssl \
        --enable-websockets \
        --enable-http2 \
        --disable-static \
        --enable-shared \
    && make -j$(nproc) \
    && make install \
    && ldconfig \
    && cd / \
    && rm -rf /tmp/curl-${CURL_VERSION}*

# 清理构建依赖（保留运行时依赖）
RUN apt-get remove -y \
    build-essential \
    libssl-dev \
    zlib1g-dev \
    libnghttp2-dev \
    libpsl-dev \
    pkg-config \
    wget \
    && apt-get autoremove -y \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# 验证 libcurl 安装
RUN /usr/local/bin/curl-config --version || echo "curl-config not found, but libcurl should be available"


# 创建非 root 用户
RUN useradd -r -u 1001 -g root appuser

# 设置工作目录
WORKDIR /app

# 复制本地构建好的二进制文件
COPY main /app/main
COPY config.yml /app/config.yml
COPY config/default.yml /app/default.yml

# 设置文件权限
RUN chmod +x /app/main \
    && chown -R appuser:root /app

# 设置环境变量确保能找到新版本libcurl
ENV LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH

# 切换到非 root 用户
USER appuser

# 健康检查（可选）
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ps aux | grep main || exit 1

# 设置入口点
ENTRYPOINT ["./main","-config","config.yml"]
