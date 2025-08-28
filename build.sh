#!/bin/bash

# 纯CGO项目构建脚本
set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 打印函数
print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 检查依赖
check_deps() {
    print_info "检查依赖..."
    local missing_deps=()
    
    for cmd in go gcc; do
        if ! command -v $cmd &> /dev/null; then
            missing_deps+=($cmd)
        fi
    done
    
    if [ ${#missing_deps[@]} -gt 0 ]; then
        print_error "缺少依赖: ${missing_deps[*]}"
        exit 1
    fi
    
    print_success "依赖检查通过"
}

# 检查Docker
check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker未安装或不在PATH中"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        print_error "Docker服务未运行或无权限访问"
        exit 1
    fi
    
    print_success "Docker检查通过"
}

# 检查Docker Compose
check_docker_compose() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker未安装或不在PATH中"
        exit 1
    fi
    
    # 检查是否有docker compose命令（新版本）
    if docker compose version &> /dev/null; then
        print_success "Docker Compose检查通过 (v2)"
        return 0
    fi
    
    # 检查是否有docker-compose命令（旧版本）
    if command -v docker-compose &> /dev/null; then
        print_success "Docker Compose检查通过 (v1)"
        return 0
    fi
    
    print_error "Docker Compose未安装或不在PATH中"
    print_info "请安装Docker Compose或确保docker compose命令可用"
    exit 1
}

# 构建Go应用
build_go() {
    print_info "构建Go应用..."
    
    if [ ! -f "go.mod" ]; then
        print_error "未找到go.mod文件"
        exit 1
    fi
    
    # 启用CGO并构建到根目录
    CGO_ENABLED=1 go build -o main .
    
    # 检查二进制文件是否生成
    if [ -f "main" ]; then
        print_success "Go应用构建成功，二进制文件: main"
        ls -lh main
    else
        print_error "Go应用构建失败，未生成main文件"
        exit 1
    fi
}

# 运行测试
run_tests() {
    print_info "运行测试..."
    
    if [ ! -f "go.mod" ]; then
        print_error "未找到go.mod文件"
        exit 1
    fi
    
    if CGO_ENABLED=1 go test -v ./...; then
        print_success "测试通过"
    else
        print_error "测试失败"
        exit 1
    fi
}

# 构建Docker镜像
build_docker() {
    local tag=${1:-gocpptest:latest}
    
    print_info "构建Docker镜像: $tag"
    
    if [ ! -f "Dockerfile" ]; then
        print_error "未找到Dockerfile文件"
        exit 1
    fi
    
    # 检查main二进制文件是否存在
    if [ ! -f "main" ]; then
        print_error "main二进制文件不存在，请先运行 ./build.sh build"
        exit 1
    fi
    
    print_info "使用本地构建的二进制文件..."
    
    docker build -t $tag .
    
    if [ $? -eq 0 ]; then
        # 验证镜像是否成功创建
        if docker image inspect $tag &> /dev/null; then
            print_success "Docker镜像构建成功: $tag"
            docker images $tag
        else
            print_error "Docker镜像构建失败，镜像未创建"
            exit 1
        fi
    else
        print_error "Docker镜像构建失败"
        exit 1
    fi
}

# 测试Docker镜像
test_docker() {
    local tag=${1:-gocpptest:latest}
    
    print_info "测试Docker镜像: $tag"
    
    # 检查镜像是否存在（更可靠的检查方法）
    if ! docker image inspect $tag &> /dev/null; then
        print_error "Docker镜像 $tag 不存在，请先构建"
        exit 1
    fi
    
    # 显示镜像信息
    print_info "镜像信息："
    docker images $tag
    
    # 运行容器进行测试
    print_info "运行容器进行测试..."
    if docker run --rm $tag; then
        print_success "Docker镜像测试通过"
    else
        print_error "Docker镜像测试失败"
        exit 1
    fi
}

# 一键Docker测试
docker_test() {
    print_info "开始一键Docker测试..."
    
    # 检查Docker
    check_docker
    
    # 先构建Go应用，确保有main二进制文件
    print_info "构建Go应用..."
    check_deps
    build_go
    
    # 构建镜像
    print_info "构建Docker镜像..."
    build_docker
    
    # 测试镜像
    print_info "测试Docker镜像..."
    test_docker
    
    print_success "Docker测试完成！"
}

# 一键Docker Compose测试
docker_compose_test() {
    print_info "开始一键Docker Compose测试..."
    
    # 检查Docker Compose
    check_docker_compose
    
    # 先构建Go应用，确保有main二进制文件
    print_info "构建Go应用..."
    check_deps
    build_go
    
    # 构建Docker Compose服务
    print_info "构建Docker Compose服务..."
    if docker compose build; then
        print_success "Docker Compose构建成功"
    else
        print_error "Docker Compose构建失败"
        exit 1
    fi
    
    # 启动服务
    print_info "启动Docker Compose服务..."
    if docker compose up -d; then
        print_success "Docker Compose服务启动成功"
    else
        print_error "Docker Compose服务启动失败"
        exit 1
    fi
    
    # 等待服务启动
    print_info "等待服务启动..."
    sleep 5
    
    # 检查服务状态
    print_info "检查服务状态..."
    if docker compose ps; then
        print_success "服务状态检查成功"
    else
        print_error "服务状态检查失败"
        exit 1
    fi
    
    # 查看服务日志
    print_info "查看服务日志..."
    docker compose logs --tail=20
    
    # 测试服务运行
    print_info "测试服务运行..."
    if docker compose exec -T http-latency-test echo "服务运行正常"; then
        print_success "服务运行测试通过"
    else
        print_warning "服务运行测试失败，但继续执行"
    fi
    
    # 停止服务
    print_info "停止Docker Compose服务..."
    if docker compose down; then
        print_success "Docker Compose服务停止成功"
    else
        print_warning "Docker Compose服务停止失败，但继续执行"
    fi
    
    print_success "Docker Compose测试完成！"
}

# 执行完整流程
run_all() {
    print_info "开始执行完整流程..."
    echo
    
    # 1. 清理
    print_info "步骤 1/5: 清理构建产物..."
    clean
    echo
    
    # 2. 检查依赖
    print_info "步骤 2/5: 检查依赖..."
    check_deps
    echo
    
    # 3. 运行测试
    print_info "步骤 3/5: 运行Go测试..."
    run_tests
    echo
    
    # 4. 重新构建
    print_info "步骤 4/5: 重新构建项目..."
    build_go
    echo
    
    # 5. Docker构建和测试
    print_info "步骤 5/5: Docker构建和测试..."
    check_docker
    build_docker
    test_docker
    docker_compose_test
    echo
    
    print_success "完整流程执行完成！"
    print_info "项目已成功构建、测试并打包到Docker镜像中"
}

# 清理
clean() {
    print_info "清理构建产物..."
    
    rm -f main
    go clean -cache -testcache 2>/dev/null || true
    
    # 清理Docker镜像
    if command -v docker &> /dev/null; then
        docker rmi gocpptest:latest 2>/dev/null || true
        print_info "Docker镜像已清理"
    fi
    
    print_success "清理完成"
}

# 显示帮助
show_help() {
    echo "CGO项目构建脚本使用方法："
    echo "  $0 build       # 构建项目"
    echo "  $0 test        # 运行测试"
    echo "  $0 docker      # 构建Docker镜像"
    echo "  $0 docker-test # 一键Docker测试（构建+测试）"
    echo "  $0 docker-compose-test # 🚀 一键Docker Compose测试（构建+启动+测试+清理）"
    echo "  $0 clean       # 清理构建产物"
    echo "  $0 rebuild     # 重新构建"
    echo "  $0 all         # 🚀 执行完整流程（清理+测试+构建+Docker）"
    echo "  $0 help        # 显示帮助"
    echo ""
    echo "推荐使用："
    echo "  $0 all         # 一键完成所有流程"
    echo "  $0 docker-compose-test # 🚀 Docker Compose完整测试"
    echo "  $0 docker-test # 仅Docker测试"
}

# 主函数
main() {
    case ${1:-help} in
        "build")
            check_deps
            build_go
            print_success "构建完成"
            ;;
        "test")
            run_tests
            ;;
        "docker")
            check_deps
            build_go
            check_docker
            build_docker
            ;;
        "docker-test")
            docker_test
            ;;
        "clean")
            clean
            ;;
        "rebuild")
            clean
            check_deps
            build_go
            print_success "重新构建完成"
            ;;
        "all")
            run_all
            ;;
        "help"|*)
            show_help
            ;;
    esac
}

main "$@"
