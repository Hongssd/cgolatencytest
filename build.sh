#!/bin/bash

# 容器化CGO项目构建脚本
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

# 耗时统计函数
start_timer() {
    start_time=$(date +%s.%N)
}

end_timer() {
    local operation_name="$1"
    end_time=$(date +%s.%N)
    duration=$(echo "$end_time - $start_time" | bc -l)
    print_success "$operation_name 完成，耗时: ${duration}s"
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

# 构建Docker镜像（多阶段构建）
build_docker_image() {
    local tag=${1:-cgolatencytest:latest}
    local include_test=${2:-false}
    
    print_info "构建Docker镜像: $tag"
    
    if [ ! -f "Dockerfile" ]; then
        print_error "未找到Dockerfile文件"
        exit 1
    fi
    
    if [ ! -f "go.mod" ]; then
        print_error "未找到go.mod文件"
        exit 1
    fi
    
    start_timer
    
    # 清理之前的镜像
    print_info "清理之前的镜像..."
    docker rmi $tag 2>/dev/null || true
    
    # 构建镜像（多阶段构建）
    print_info "开始多阶段构建..."
    docker build -t $tag .
    
    if [ $? -eq 0 ]; then
        # 验证镜像是否成功创建
        if docker image inspect $tag &> /dev/null; then
            end_timer "Docker镜像构建"
            print_success "Docker镜像构建成功: $tag"
            print_info "镜像信息："
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
    local tag=${1:-cgolatencytest:latest}
    
    print_info "测试Docker镜像: $tag"
    
    # 检查镜像是否存在
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
    
    # 构建镜像
    print_info "构建Docker镜像..."
    build_docker_image
    
    # 测试镜像
    print_info "测试Docker镜像..."
    test_docker
    
    print_success "Docker测试完成！"
}

# 一键Docker Compose启动
docker_compose_start() {
    print_info "开始一键Docker Compose启动..."
    
    # 检查Docker Compose
    check_docker_compose
    
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
    
    print_success "Docker Compose服务启动完成！"
    print_info "服务正在后台运行，可以使用以下命令管理："
    print_info "  查看日志: docker compose logs -f"
    print_info "  停止服务: docker compose down"
    print_info "  查看状态: docker compose ps"
}

# 执行完整流程
run_all() {
    print_info "开始执行完整流程..."
    echo
    
    # 1. 清理
    print_info "步骤 1/3: 清理构建产物..."
    clean
    echo
    
    # 2. 构建和测试
    print_info "步骤 2/3: 构建和测试..."
    check_docker
    build_docker_image
    test_docker
    echo
    
    # 3. Docker Compose启动
    print_info "步骤 3/3: Docker Compose启动..."
    docker_compose_start
    echo
    
    print_success "完整流程执行完成！"
    print_info "项目已成功在容器中构建、测试并打包到Docker镜像中"
}

# 清理
clean() {
    print_info "清理构建产物..."
    
    rm -f main
    
    # 清理Docker镜像
    if command -v docker &> /dev/null; then
        docker rmi cgolatencytest:latest 2>/dev/null || true
        print_info "Docker镜像已清理"
    fi
    
    print_success "清理完成"
}

# 显示帮助
show_help() {
    echo "容器化CGO项目构建脚本使用方法："
    echo ""
    echo "构建相关："
    echo "  $0 build       # 构建Docker镜像（多阶段构建）"
    echo "  $0 clean       # 清理构建产物"
    echo ""
    echo "Docker相关："
    echo "  $0 docker      # 构建Docker镜像"
    echo "  $0 docker-test # 一键Docker测试（构建+测试）"
    echo "  $0 docker-compose-start # Docker Compose启动"
    echo ""
    echo "完整流程："
    echo "  $0 all         # 执行完整流程（清理+构建+测试+Docker）"
    echo ""
    echo "其他："
    echo "  $0 help        # 显示帮助"
    echo ""
    echo "推荐使用："
    echo "  $0 all         # 一键完成所有流程"
    echo "  $0 build       # 仅构建项目"
    echo "  $0 docker-test # Docker测试"
}

# 主函数
main() {
    case ${1:-help} in
        "build")
            check_docker
            build_docker_image
            print_success "构建完成"
            ;;
        "docker")
            check_docker
            build_docker_image
            ;;
        "docker-test")
            docker_test
            ;;
        "docker-compose-start")
            docker_compose_start
            ;;
        "clean")
            clean
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