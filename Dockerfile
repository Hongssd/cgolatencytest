# 优化的 Dockerfile - 多阶段构建，最小化运行时镜像
FROM ubuntu:24.04 AS runtime

# 设置环境变量
ENV TZ=Asia/Shanghai
ENV DEBIAN_FRONTEND=noninteractive

# 安装必要的运行时依赖库
RUN apt-get update && apt-get install -y \
    libcurl4 \
    libssl3 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# 创建非 root 用户
RUN useradd -r -u 1001 -g root appuser

# 设置工作目录
WORKDIR /app

# 复制本地构建好的二进制文件
COPY main /app/main

# 设置文件权限
RUN chmod +x /app/main \
    && chown -R appuser:root /app

# 切换到非 root 用户
USER appuser

# 健康检查（可选）
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ps aux | grep main || exit 1

# 设置入口点
ENTRYPOINT ["./main"]
