#!/bin/bash

# Istio快速部署脚本
# 用于在Kubernetes集群中部署Istio并配置金丝雀发布

set -e

echo "======================================"
echo "Istio部署与金丝雀发布配置脚本"
echo "======================================"

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 函数: 打印成功消息
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# 函数: 打印错误消息
print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# 函数: 打印警告消息
print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# 函数: 检查命令是否存在
check_command() {
    if ! command -v $1 &> /dev/null; then
        print_error "$1 未安装，请先安装 $1"
        exit 1
    fi
    print_success "$1 已安装"
}

# 函数: 等待Pod就绪
wait_for_pods() {
    local namespace=$1
    local label=$2
    local timeout=300
    
    echo "等待Pod就绪 (namespace: $namespace, label: $label)..."
    
    local start_time=$(date +%s)
    while true; do
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        
        if [ $elapsed -gt $timeout ]; then
            print_error "等待Pod就绪超时"
            return 1
        fi
        
        local ready_pods=$(kubectl get pods -n $namespace -l $label -o jsonpath='{.items[*].status.conditions[?(@.type=="Ready")].status}' | grep -o "True" | wc -l)
        local total_pods=$(kubectl get pods -n $namespace -l $label --no-headers | wc -l)
        
        if [ "$ready_pods" -eq "$total_pods" ] && [ "$total_pods" -gt 0 ]; then
            print_success "所有Pod已就绪 ($ready_pods/$total_pods)"
            return 0
        fi
        
        echo -n "."
        sleep 5
    done
}

# 步骤1: 检查前置条件
echo ""
echo "步骤1: 检查前置条件"
echo "--------------------------------------"
check_command kubectl
check_command curl

# 检查Kubernetes集群连接
if ! kubectl cluster-info &> /dev/null; then
    print_error "无法连接到Kubernetes集群"
    exit 1
fi
print_success "Kubernetes集群连接正常"

# 步骤2: 检查Istio是否已安装
echo ""
echo "步骤2: 检查Istio安装状态"
echo "--------------------------------------"
if kubectl get namespace istio-system &> /dev/null; then
    print_warning "Istio已安装，跳过安装步骤"
    
    # 检查Istio组件状态
    if kubectl get pods -n istio-system | grep -q "Running"; then
        print_success "Istio组件运行正常"
    else
        print_error "Istio组件状态异常"
        kubectl get pods -n istio-system
        exit 1
    fi
else
    echo "Istio未安装，开始安装..."
    
    # 步骤3: 下载Istio
    echo ""
    echo "步骤3: 下载Istio"
    echo "--------------------------------------"
    
    if ! command -v istioctl &> /dev/null; then
        echo "下载最新版本的Istio..."
        curl -L https://istio.io/downloadIstio | sh -
        
        # 进入Istio目录
        cd istio-*
        
        # 添加到PATH
        export PATH=$PWD/bin:$PATH
        
        # 复制到系统路径
        sudo cp bin/istioctl /usr/local/bin/
        
        cd ..
        
        print_success "Istio下载完成"
    else
        print_success "istioctl已安装"
    fi
    
    # 步骤4: 安装Istio
    echo ""
    echo "步骤4: 安装Istio"
    echo "--------------------------------------"
    
    echo "选择Istio安装配置:"
    echo "1) demo (适合开发和测试)"
    echo "2) minimal (适合生产环境)"
    echo "3) default (默认配置)"
    read -p "请选择 (1-3, 默认1): " profile_choice
    
    case $profile_choice in
        2) profile="minimal" ;;
        3) profile="default" ;;
        *) profile="demo" ;;
    esac
    
    echo "使用 $profile 配置文件安装Istio..."
    istioctl install --set profile=$profile -y
    
    print_success "Istio安装完成"
    
    # 等待Istio组件就绪
    wait_for_pods istio-system "app=istiod"
fi

# 步骤5: 启用自动边车注入
echo ""
echo "步骤5: 启用自动边车注入"
echo "--------------------------------------"

if kubectl get namespace default -o jsonpath='{.metadata.labels}' | grep -q "istio-injection"; then
    print_warning "default命名空间已启用边车注入"
else
    kubectl label namespace default istio-injection=enabled
    print_success "已为default命名空间启用边车注入"
fi

# 步骤6: 应用Istio配置
echo ""
echo "步骤6: 应用Istio配置"
echo "--------------------------------------"

if [ -d "k8s/istio" ]; then
    echo "应用Istio配置文件..."
    kubectl apply -f k8s/istio/
    print_success "Istio配置已应用"
else
    print_error "未找到k8s/istio目录"
    exit 1
fi

# 步骤7: 部署应用
echo ""
echo "步骤7: 部署应用"
echo "--------------------------------------"

echo "选择部署方式:"
echo "1) 部署稳定版本"
echo "2) 部署金丝雀版本"
echo "3) 部署稳定和金丝雀版本"
read -p "请选择 (1-3, 默认1): " deploy_choice

case $deploy_choice in
    2)
        echo "部署金丝雀版本..."
        kubectl apply -f k8s/deployment-canary.yaml
        wait_for_pods default "app=gaamingzhang-blog,version=canary"
        ;;
    3)
        echo "部署稳定和金丝雀版本..."
        kubectl apply -f k8s/deployment-stable.yaml
        kubectl apply -f k8s/deployment-canary.yaml
        wait_for_pods default "app=gaamingzhang-blog"
        ;;
    *)
        echo "部署稳定版本..."
        kubectl apply -f k8s/deployment-stable.yaml
        wait_for_pods default "app=gaamingzhang-blog,version=stable"
        ;;
esac

# 步骤8: 验证部署
echo ""
echo "步骤8: 验证部署"
echo "--------------------------------------"

echo "检查Pod状态:"
kubectl get pods -l app=gaamingzhang-blog

echo ""
echo "检查Istio配置:"
kubectl get destinationrules,virtualservices,gateways -n default

echo ""
echo "使用istioctl分析配置:"
istioctl analyze

# 步骤9: 获取访问信息
echo ""
echo "步骤9: 获取访问信息"
echo "--------------------------------------"

# 获取Ingress Gateway信息
INGRESS_HOST=$(kubectl get svc istio-ingressgateway -n istio-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")

if [ -z "$INGRESS_HOST" ] || [ "$INGRESS_HOST" == "" ]; then
    # 使用NodePort
    INGRESS_PORT=$(kubectl get svc istio-ingressgateway -n istio-system -o jsonpath='{.spec.ports[?(@.name=="http2")].nodePort}')
    INGRESS_HOST=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[0].address}')
    GATEWAY_URL="$INGRESS_HOST:$INGRESS_PORT"
else
    GATEWAY_URL="$INGRESS_HOST:80"
fi

print_success "访问地址: http://$GATEWAY_URL"
echo "Host头部: blog.local"

# 步骤10: 显示金丝雀发布命令
echo ""
echo "步骤10: 金丝雀发布命令"
echo "--------------------------------------"
echo "使用以下命令进行金丝雀发布:"
echo ""
echo "# 设置5%流量到金丝雀"
echo "./scripts/canary-release.sh 5"
echo ""
echo "# 设置10%流量到金丝雀"
echo "./scripts/canary-release.sh 10"
echo ""
echo "# 设置25%流量到金丝雀"
echo "./scripts/canary-release.sh 25"
echo ""
echo "# 设置50%流量到金丝雀"
echo "./scripts/canary-release.sh 50"
echo ""
echo "# 设置100%流量到金丝雀（完成发布）"
echo "./scripts/canary-release.sh 100"

echo ""
echo "======================================"
print_success "Istio部署与配置完成！"
echo "======================================"
