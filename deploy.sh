#!/bin/bash

set -e

# 配置变量
PRIVATE_REGISTRY="192.168.31.54:5001"
IMAGE_NAME="gaamingzhang-blog"
IMAGE_TAG="latest"
FULL_IMAGE_NAME="${PRIVATE_REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}"

# 彩色输出函数
green() {
  echo -e "\033[32m$1\033[0m"
}

yellow() {
  echo -e "\033[33m$1\033[0m"
}

red() {
  echo -e "\033[31m$1\033[0m"
}

echo "=== 开始部署 Gaaming Zhang 博客到 Kubernetes ==="

# 1. 构建Docker镜像（amd64架构）
green "1. 构建amd64架构的Docker镜像..."
docker build --platform linux/amd64 -t ${IMAGE_NAME}:${IMAGE_TAG} .

# 添加私有仓库标签
green "2. 添加私有仓库标签..."
docker tag ${IMAGE_NAME}:${IMAGE_TAG} ${FULL_IMAGE_NAME}

# 3. 推送镜像到本地私有仓库
green "3. 推送镜像到本地私有仓库..."
docker push ${FULL_IMAGE_NAME}

# 4. 验证镜像推送成功
green "4. 验证镜像推送成功..."
if curl -s -o /dev/null http://${PRIVATE_REGISTRY}/v2/${IMAGE_NAME}/tags/list; then
  green "✓ 本地私有仓库可访问"
  IMAGE_TAGS=$(curl -s http://${PRIVATE_REGISTRY}/v2/${IMAGE_NAME}/tags/list | grep -o "latest")
  if [ -n "${IMAGE_TAGS}" ]; then
    green "✓ 镜像已成功推送到私有仓库"
  else
    red "✗ 镜像推送失败，请检查网络连接"
    exit 1
  fi
else
  red "✗ 本地私有仓库不可访问，请检查仓库状态"
  exit 1
fi

# 5. 部署到Kubernetes
green "5. 部署到Kubernetes..."
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/ingress.yaml

# 6. 查看部署状态
green "6. 查看部署状态..."
sleep 5
echo "--- 部署状态 ---"
kubectl get deployments
echo "--- 服务状态 ---"
kubectl get services
echo "--- Ingress状态 ---"
kubectl get ingresses

# 7. 等待Pod就绪
green "7. 等待Pod就绪..."
if kubectl wait --for=condition=ready pod -l app=gaamingzhang-blog --timeout=120s; then
  green "✓ Pod已就绪"
else
  yellow "⚠ Pod未在指定时间内就绪，检查Pod状态..."
  kubectl get pods -l app=gaamingzhang-blog
  kubectl describe pod -l app=gaamingzhang-blog | grep -A 10 Events
fi

# 8. 配置本地DNS
green "8. 配置本地DNS..."
INGRESS_IP=$(kubectl get ingress gaamingzhang-blog-ingress -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")

if [ -z "$INGRESS_IP" ]; then
  yellow "⚠ 警告: Ingress IP 未获取到，请检查集群网络配置"
  yellow "请手动在 /etc/hosts 文件中添加以下条目:"
  yellow "<INGRESS_IP> blog.local"
else
  green "✓ 获取到 Ingress IP: $INGRESS_IP"
  yellow "请在 /etc/hosts 文件中添加以下条目:"
  yellow "$INGRESS_IP blog.local"
  
  # 尝试自动添加到 /etc/hosts
  if [ -w "/etc/hosts" ]; then
    # 移除旧条目
    sudo sed -i '' '/blog.local/d' /etc/hosts 2>/dev/null || true
    # 添加新条目
    echo "$INGRESS_IP blog.local" | sudo tee -a /etc/hosts 2>/dev/null
    if [ $? -eq 0 ]; then
      green "✓ 已自动更新 /etc/hosts 文件"
    else
      yellow "⚠ 需要管理员权限更新 /etc/hosts 文件，请手动执行上述操作"
    fi
  else
    yellow "⚠ 需要管理员权限更新 /etc/hosts 文件，请手动执行上述操作"
  fi
fi

# 9. 测试访问
green "9. 测试访问..."
if command -v curl &> /dev/null; then
  if [ -n "$INGRESS_IP" ]; then
    echo "测试访问 blog.local:"
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://blog.local 2>/dev/null)
    if [ "$HTTP_CODE" -eq 200 ]; then
      green "✓ 访问成功！HTTP状态码: $HTTP_CODE"
    else
      yellow "⚠ 访问可能存在问题，HTTP状态码: $HTTP_CODE"
    fi
  else
    yellow "⚠ 无法测试访问，因为 Ingress IP 未获取到"
  fi
else
  yellow "⚠ curl 命令未找到，请手动测试访问 http://blog.local"
fi

# 10. 显示部署信息
green "10. 部署信息汇总..."
echo "=== 部署完成 ==="
echo "博客地址: http://blog.local"
echo "镜像地址: ${FULL_IMAGE_NAME}"
echo ""
green "如果遇到问题，请检查:"
echo "1. Kubernetes 集群状态"
echo "2. Ingress 控制器是否已安装"
echo "3. /etc/hosts 文件配置"
echo "4. 防火墙设置"
echo "5. 本地私有仓库状态"
echo ""
green "部署脚本执行完成！"
