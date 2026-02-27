#!/bin/bash

# 金丝雀发布回滚脚本
# 用于快速将所有流量切回稳定版本

set -e

echo "======================================"
echo "金丝雀发布回滚脚本"
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

# 确认回滚
echo -e "${YELLOW}警告: 此操作将把所有流量切回稳定版本${NC}"
read -p "确认要回滚吗? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
    echo "回滚已取消"
    exit 0
fi

# 步骤1: 将所有流量切回稳定版本
echo ""
echo "步骤1: 将所有流量切回稳定版本"
echo "--------------------------------------"

cat <<EOF | kubectl apply -f -
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: gaamingzhang-blog-external
  namespace: default
spec:
  hosts:
  - "blog.local"
  - "*"
  gateways:
  - gaamingzhang-blog-gateway
  http:
  - route:
    - destination:
        host: gaamingzhang-blog-service
        subset: stable
      weight: 100
    - destination:
        host: gaamingzhang-blog-service
        subset: canary
      weight: 0
EOF

print_success "流量已切回稳定版本"

# 步骤2: 等待配置生效
echo ""
echo "步骤2: 等待配置生效"
echo "--------------------------------------"
sleep 5

# 步骤3: 验证稳定版本
echo ""
echo "步骤3: 验证稳定版本"
echo "--------------------------------------"

echo "检查稳定版本Pod状态:"
kubectl get pods -l app=gaamingzhang-blog,version=stable

echo ""
echo "检查VirtualService配置:"
kubectl get virtualservice gaamingzhang-blog-external -n default -o yaml | grep -A 10 "http:"

# 步骤4: 询问是否删除金丝雀版本
echo ""
read -p "是否要删除金丝雀版本? (yes/no, 默认no): " delete_canary

if [ "$delete_canary" == "yes" ]; then
    echo ""
    echo "步骤4: 删除金丝雀版本"
    echo "--------------------------------------"
    
    kubectl delete deployment gaamingzhang-blog-canary -n default
    print_success "金丝雀版本已删除"
else
    print_warning "金丝雀版本保留，但不会接收流量"
fi

# 步骤5: 显示当前状态
echo ""
echo "步骤5: 当前状态"
echo "--------------------------------------"

echo "Pod状态:"
kubectl get pods -l app=gaamingzhang-blog

echo ""
echo "流量分配:"
echo "- 稳定版本: 100%"
echo "- 金丝雀版本: 0%"

echo ""
echo "======================================"
print_success "回滚完成！"
echo "======================================"
