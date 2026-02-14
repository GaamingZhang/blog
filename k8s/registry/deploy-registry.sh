#!/bin/bash
set -e

REGISTRY_NAMESPACE="container-registry"
REGISTRY_DIR="$(dirname "$0")"
REGISTRY_NODE="kubernetesworker000"

echo "=== 部署 Docker Registry 到 Kubernetes ==="

echo ""
echo "步骤 1: 创建命名空间"
kubectl apply -f ${REGISTRY_DIR}/namespace.yaml

echo ""
echo "步骤 2: 创建存储配置"
echo "Registry 将部署在节点: ${REGISTRY_NODE}"
echo ""
echo "请在该节点上执行以下命令创建存储目录:"
echo ""
echo "  ssh ${REGISTRY_NODE} 'sudo mkdir -p /mnt/data/registry && sudo chmod 777 /mnt/data/registry'"
echo ""
read -p "是否已完成存储目录创建? (y/n): " confirm
if [[ "$confirm" != "y" ]]; then
    echo "请先完成存储配置"
    exit 1
fi
kubectl apply -f ${REGISTRY_DIR}/pv.yaml

echo ""
echo "步骤 3: 创建认证配置"
echo "请运行以下命令创建 htpasswd 认证:"
echo ""
echo "  # 安装 htpasswd 工具 (如果未安装)"
echo "  # Ubuntu/Debian: sudo apt-get install apache2-utils"
echo "  # macOS: brew install httpd"
echo ""
echo "  # 创建 htpasswd 文件"
echo "  htpasswd -c auth.htpasswd admin"
echo ""
echo "  # 创建 Kubernetes Secret"
echo "  kubectl create secret generic registry-auth \\"
echo "    --from-file=HTPASSWD=auth.htpasswd \\"
echo "    -n ${REGISTRY_NAMESPACE}"
echo ""
read -p "是否已完成认证配置? (y/n): " confirm
if [[ "$confirm" != "y" ]]; then
    echo "请先完成认证配置"
    exit 1
fi

echo ""
echo "步骤 4: 创建 TLS 证书"
echo "请运行以下命令创建自签名证书:"
echo ""
echo "  # 生成私钥和证书"
echo "  openssl req -x509 -newkey rsa:4096 -keyout tls.key -out tls.crt \\"
echo "    -days 365 -nodes -subj '/CN=registry.local'"
echo ""
echo "  # 创建 Kubernetes Secret"
echo "  kubectl create secret tls registry-tls \\"
echo "    --cert=tls.crt --key=tls.key \\"
echo "    -n ${REGISTRY_NAMESPACE}"
echo ""
read -p "是否已完成 TLS 配置? (y/n): " confirm
if [[ "$confirm" != "y" ]]; then
    echo "请先完成 TLS 配置"
    exit 1
fi

echo ""
echo "步骤 5: 部署 Registry"
kubectl apply -f ${REGISTRY_DIR}/deployment.yaml
kubectl apply -f ${REGISTRY_DIR}/service.yaml
kubectl apply -f ${REGISTRY_DIR}/ingress.yaml

echo ""
echo "步骤 6: 等待部署完成"
kubectl rollout status deployment/docker-registry -n ${REGISTRY_NAMESPACE}

echo ""
echo "=== Registry 部署完成 ==="
echo ""
echo "部署节点: ${REGISTRY_NODE}"
echo ""
echo "访问方式:"
echo "  - ClusterIP: docker-registry.container-registry.svc.cluster.local:5000"
echo "  - NodePort: https://<任意节点IP>:30500"
echo "  - Ingress: https://registry.local (需要配置 /etc/hosts)"
echo ""
echo "配置 Docker 客户端 (在 Jenkins 节点上):"
echo ""
echo "  1. 添加 insecure-registry (如果使用自签名证书)"
echo "     编辑 /etc/docker/daemon.json:"
echo '     {"insecure-registries": ["registry.local", "<node-ip>:30500"]}'
echo "     然后重启 Docker: sudo systemctl restart docker"
echo ""
echo "  2. 登录 Registry:"
echo "     docker login <node-ip>:30500 -u admin"
echo ""
echo "  3. 推送镜像:"
echo "     docker tag myimage <node-ip>:30500/myimage:latest"
echo "     docker push <node-ip>:30500/myimage:latest"
echo ""
echo "创建 Jenkins 凭证:"
echo "  在 Jenkins 中创建 'registry-credentials' 凭证"
echo "  类型: Username with password"
echo "  Username: admin"
echo "  Password: <你设置的密码>"
