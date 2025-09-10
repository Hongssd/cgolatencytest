# 运行时环境 - 包含最新版本 libcurl
FROM ubuntu:24.04 AS runtime

# 设置环境变量
ENV TZ=Asia/Shanghai
ENV DEBIAN_FRONTEND=noninteractive
ENV CURL_VERSION=8.11.0

# 安装构建依赖（用于编译libcurl）和运行时依赖
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

# 复制本地构建好的二进制文件
COPY main /app/main
COPY config.yml /app/config.yml
COPY config/default.yml /app/default.yml

# 设置文件权限
RUN chmod +x /app/main \
    && chown -R appuser:root /app

# 切换到非 root 用户
USER appuser

# 健康检查（可选）
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ps aux | grep main || exit 1

# 设置入口点
ENTRYPOINT ["./main","-config","config.yml"]
