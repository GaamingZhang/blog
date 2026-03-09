# Kubernetes多集群直接部署命令清单（修正版）

## 快速部署命令（按顺序执行）

### 1. 所有节点系统准备

#### 1.1 清理现有集群（所有节点：30, 31, 40, 41, 42, 43）

```bash
# 在所有节点执行
kubeadm reset -f
rm -rf /etc/kubernetes/ /var/lib/kubelet/ /var/lib/dockershim/ /var/lib/etcd/ /var/lib/cni/ /etc/cni/ /root/.kube/config
iptables -F && iptables -t nat -F && iptables -t mangle -F && iptables -X
systemctl restart containerd
```

#### 1.2 设置主机名（每个节点单独执行）

```bash
# 192.168.31.30
hostnamectl set-hostname cluster1-master

# 192.168.31.31
hostnamectl set-hostname cluster2-master

# 192.168.31.40
hostnamectl set-hostname cluster1-worker1

# 192.168.31.41
hostnamectl set-hostname cluster1-worker2

# 192.168.31.42
hostnamectl set-hostname cluster2-worker1

# 192.168.31.43
hostnamectl set-hostname cluster2-worker2
```

#### 1.3 配置hosts（所有节点）

```bash
cat >> /etc/hosts << 'EOF'
# 集群1
192.168.31.30 cluster1-master
192.168.31.40 cluster1-worker1
192.168.31.41 cluster1-worker2
192.168.31.100 cluster1-vip

# 集群2
192.168.31.31 cluster2-master
192.168.31.42 cluster2-worker1
192.168.31.43 cluster2-worker2
192.168.31.101 cluster2-vip
EOF
```

#### 1.4 系统基础配置（所有节点）

```bash
# 关闭swap
swapoff -a
sed -i '/swap/d' /etc/fstab

# 加载内核模块
cat > /etc/modules-load.d/k8s.conf << 'EOF'
overlay
br_netfilter
EOF

modprobe overlay
modprobe br_netfilter

# 配置内核参数
cat > /etc/sysctl.d/k8s.conf << 'EOF'
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF

sysctl --system
```

#### 1.5 安装containerd（所有节点）

```bash
# 安装依赖
apt-get update
apt-get install -y ca-certificates curl gnupg lsb-release

# 添加Docker仓库
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg

echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

# 安装containerd
sudo apt-get update
sudo apt-get install -y containerd.io

# 配置containerd
mkdir -p /etc/containerd
sudo containerd config default > /etc/containerd/config.toml
sudo sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml

# 重启containerd
sudo systemctl restart containerd
sudo systemctl enable containerd
```

#### 1.6 安装Kubernetes组件（所有节点）

```bash
# 添加Kubernetes仓库
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.31/deb/Release.key | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
chmod 644 /etc/apt/keyrings/kubernetes-apt-keyring.gpg

echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.31/deb/ /' | tee /etc/apt/sources.list.d/kubernetes.list

# 安装Kubernetes
apt-get update
apt-get install -y kubelet=1.31.3-1.1 kubeadm=1.31.3-1.1 kubectl=1.31.3-1.1
apt-mark hold kubelet kubeadm kubectl

systemctl enable kubelet
```

---

### 2. 集群1部署（192.168.31.30）

#### 2.1 初始化集群1 Master（仅在192.168.31.30执行）

**重要说明**：使用本地IP初始化，不使用VIP，避免鸡生蛋问题。

```bash
# 创建配置文件（使用本地IP，但证书包含VIP）
cat > kubeadm-config-cluster1.yaml << 'EOF'
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
kubernetesVersion: v1.31.3
controlPlaneEndpoint: "192.168.31.30:6443"  # 使用本地IP
networking:
  podSubnet: "10.244.1.0/16"
  serviceSubnet: "10.96.1.0/12"
clusterName: cluster1
apiServer:
  certSANs:
  - "192.168.31.100"  # VIP地址
  - "192.168.31.30"   # 本地IP
  - "cluster1-vip"    # VIP主机名
  - "cluster1-master" # Master主机名
  - "kubernetes"      # Kubernetes服务名
  - "kubernetes.default.svc"
  - "kubernetes.default.svc.cluster.local"
  - "10.96.1.1"       # Kubernetes ClusterIP
---
apiVersion: kubeadm.k8s.io/v1beta3
kind: InitConfiguration
localAPIEndpoint:
  advertiseAddress: 192.168.31.30
  bindPort: 6443
nodeRegistration:
  criSocket: unix:///var/run/containerd/containerd.sock
  imagePullPolicy: IfNotPresent
  name: cluster1-master
  taints: null
EOF

# 预下载镜像
sudo kubeadm config images pull --config kubeadm-config-cluster1.yaml

# 初始化集群
sudo kubeadm init --config kubeadm-config-cluster1.yaml --upload-certs

# 配置kubectl
mkdir -p $HOME/.kube
cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
chown $(id -u):$(id -g) $HOME/.kube/config

# 验证
kubectl get nodes
```

#### 2.2 安装Calico网络（仅在192.168.31.30执行）

**重要**：先安装Calico让节点变为Ready状态，避免Pod调度问题。

```bash
# 下载Calico
curl https://raw.githubusercontent.com/projectcalico/calico/v3.29.1/manifests/calico.yaml -O

# 修改Pod CIDR
sed -i 's|# - name: CALICO_IPV4POOL_CIDR|- name: CALICO_IPV4POOL_CIDR|' calico.yaml
sed -i 's|#   value: "192.168.0.0/16"|  value: "10.244.1.0/16"|' calico.yaml

# 安装Calico
kubectl apply -f calico.yaml

# 等待就绪
kubectl wait --for=condition=ready pod -l k8s-app=calico-node -n kube-system --timeout=300s

# 验证节点状态（应该变为Ready）
kubectl get nodes
```

#### 2.3 安装kube-vip（仅在192.168.31.30执行）

**重要**：在Calico安装完成后部署kube-vip，提供VIP功能。

```bash
# 创建RBAC
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-vip
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: system:kube-vip-role
rules:
  - apiGroups: [""]
    resources: ["services", "services/status", "nodes"]
    verbs: ["list", "get", "watch", "update"]
  - apiGroups: ["coordination.k8s.io"]
    resources: ["leases"]
    verbs: ["list", "get", "watch", "update", "create"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:kube-vip-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:kube-vip-role
subjects:
- kind: ServiceAccount
  name: kube-vip
  namespace: kube-system
EOF

# 创建kube-vip Pod
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: kube-vip
  namespace: kube-system
spec:
  containers:
  - args:
    - manager
    env:
    - name: vip_arp
      value: "true"
    - name: port
      value: "6443"
    - name: vip_interface
      value: "enp2s0"
    - name: address
      value: "192.168.31.100"
    - name: cp_enable
      value: "true"
    - name: vip_leaderelection
      value: "false"
    image: ghcr.io/kube-vip/kube-vip:v0.8.0
    name: kube-vip
    securityContext:
      capabilities:
        add:
        - NET_ADMIN
        - NET_RAW
    volumeMounts:
    - mountPath: /etc/kubernetes/admin.conf
      name: kubeconfig
  hostNetwork: true
  tolerations:
  - key: node-role.kubernetes.io/control-plane
    effect: NoSchedule
  - key: node-role.kubernetes.io/master
    effect: NoSchedule
  volumes:
  - hostPath:
      path: /etc/kubernetes/admin.conf
    name: kubeconfig
EOF

# 等待kube-vip启动
sleep 10

# 验证VIP
ip addr show enp2s0 | grep 192.168.31.100

# 验证kube-vip Pod
kubectl get pods -n kube-system | grep kube-vip
```

#### 2.4 更新kubeconfig使用VIP（可选，推荐）

```bash
# 更新kubeconfig使用VIP
kubectl config set-cluster kubernetes --server=https://192.168.31.100:6443

# 验证连接
kubectl cluster-info
kubectl get nodes
```

#### 2.5 Worker节点加入集群1（在192.168.31.40和192.168.31.41执行）

```bash
# 在Master节点获取join命令
kubeadm token create --print-join-command

# 在Worker节点执行输出的join命令
# 例如：
kubeadm join 192.168.31.100:6443 --token <token> --discovery-token-ca-cert-hash sha256:<hash>
```

---

### 3. 集群2部署（192.168.31.31）

#### 3.1 初始化集群2 Master（仅在192.168.31.31执行）

```bash
# 创建配置文件（使用本地IP，但证书包含VIP）
cat > kubeadm-config-cluster2.yaml << 'EOF'
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
kubernetesVersion: v1.31.3
controlPlaneEndpoint: "192.168.31.31:6443"  # 使用本地IP
networking:
  podSubnet: "10.244.2.0/16"
  serviceSubnet: "10.96.2.0/12"
clusterName: cluster2
apiServer:
  certSANs:
  - "192.168.31.101"  # VIP地址
  - "192.168.31.31"   # 本地IP
  - "cluster2-vip"    # VIP主机名
  - "cluster2-master" # Master主机名
  - "kubernetes"      # Kubernetes服务名
  - "kubernetes.default.svc"
  - "kubernetes.default.svc.cluster.local"
  - "10.96.2.1"       # Kubernetes ClusterIP
---
apiVersion: kubeadm.k8s.io/v1beta3
kind: InitConfiguration
localAPIEndpoint:
  advertiseAddress: 192.168.31.31
  bindPort: 6443
nodeRegistration:
  criSocket: unix:///var/run/containerd/containerd.sock
  imagePullPolicy: IfNotPresent
  name: cluster2-master
  taints: null
EOF

# 预下载镜像
kubeadm config images pull --config kubeadm-config-cluster2.yaml

# 初始化集群
kubeadm init --config kubeadm-config-cluster2.yaml --upload-certs

# 配置kubectl
mkdir -p $HOME/.kube
cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
chown $(id -u):$(id -g) $HOME/.kube/config

# 验证
kubectl get nodes
```

#### 3.2 安装Calico网络（仅在192.168.31.31执行）

**重要**：先安装Calico让节点变为Ready状态，避免Pod调度问题。

```bash
# 下载Calico
curl https://raw.githubusercontent.com/projectcalico/calico/v3.29.1/manifests/calico.yaml -O

# 修改Pod CIDR
sed -i 's|# - name: CALICO_IPV4POOL_CIDR|- name: CALICO_IPV4POOL_CIDR|' calico.yaml
sed -i 's|#   value: "192.168.0.0/16"|  value: "10.244.2.0/16"|' calico.yaml

# 安装Calico
kubectl apply -f calico.yaml

# 等待就绪
kubectl wait --for=condition=ready pod -l k8s-app=calico-node -n kube-system --timeout=300s

# 验证节点状态（应该变为Ready）
kubectl get nodes
```

#### 3.3 安装kube-vip（仅在192.168.31.31执行）

**重要**：在Calico安装完成后部署kube-vip，提供VIP功能。

```bash
# 创建RBAC
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-vip
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: system:kube-vip-role
rules:
  - apiGroups: [""]
    resources: ["services", "services/status", "nodes"]
    verbs: ["list", "get", "watch", "update"]
  - apiGroups: ["coordination.k8s.io"]
    resources: ["leases"]
    verbs: ["list", "get", "watch", "update", "create"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:kube-vip-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:kube-vip-role
subjects:
- kind: ServiceAccount
  name: kube-vip
  namespace: kube-system
EOF

# 创建kube-vip Pod（修改VIP为192.168.31.101）
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: kube-vip
  namespace: kube-system
spec:
  containers:
  - args:
    - manager
    env:
    - name: vip_arp
      value: "true"
    - name: port
      value: "6443"
    - name: vip_interface
      value: "enp2s0"
    - name: address
      value: "192.168.31.101"
    - name: cp_enable
      value: "true"
    - name: vip_leaderelection
      value: "false"
    image: ghcr.io/kube-vip/kube-vip:v0.8.0
    name: kube-vip
    securityContext:
      capabilities:
        add:
        - NET_ADMIN
        - NET_RAW
    volumeMounts:
    - mountPath: /etc/kubernetes/admin.conf
      name: kubeconfig
  hostNetwork: true
  tolerations:
  - key: node-role.kubernetes.io/control-plane
    effect: NoSchedule
  - key: node-role.kubernetes.io/master
    effect: NoSchedule
  volumes:
  - hostPath:
      path: /etc/kubernetes/admin.conf
    name: kubeconfig
EOF

# 等待kube-vip启动
sleep 10

# 验证VIP
ip addr show enp2s0 | grep 192.168.31.101

# 验证kube-vip Pod
kubectl get pods -n kube-system | grep kube-vip
```

#### 3.4 更新kubeconfig使用VIP（可选，推荐）

```bash
# 更新kubeconfig使用VIP
kubectl config set-cluster kubernetes --server=https://192.168.31.101:6443

# 验证连接
kubectl cluster-info
kubectl get nodes
```

#### 3.5 Worker节点加入集群2（在192.168.31.42和192.168.31.43执行）

```bash
# 在Master节点获取join命令
kubeadm token create --print-join-command

# 在Worker节点执行
kubeadm join 192.168.31.101:6443 --token <token> --discovery-token-ca-cert-hash sha256:<hash>
```

---

### 4. 基础设施组件部署

#### 4.0 部署本地存储类（所有集群）

**重要**：Harbor等应用需要持久存储，必须先部署StorageClass。

```bash
# 1. 创建本地存储目录（所有节点执行）
sudo mkdir -p /opt/local-path-provisioner
sudo chmod 777 /opt/local-path-provisioner

# 2. 预先拉取busybox镜像（所有节点执行）
sudo crictl pull docker.io/library/busybox:1.36

# 3. 部署local-path-provisioner（仅在Master节点执行）
kubectl apply -f - <<EOF
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: local-path
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
provisioner: rancher.io/local-path
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-path-config
  namespace: kube-system
data:
  config.json: |-
    {
      "nodePathMap":[
        {
          "node":"DEFAULT_PATH_FOR_NON_LISTED_NODES",
          "paths":["/opt/local-path-provisioner"]
        }
      ]
    }
  setup: |-
    #!/bin/sh
    mkdir -p "$VOL_DIR"
    chmod 777 "$VOL_DIR"
  teardown: |-
    #!/bin/sh
    rm -rf "$VOL_DIR"
  helperPod.yaml: |-
    apiVersion: v1
    kind: Pod
    metadata:
      name: helper-pod
    spec:
      priorityClassName: system-node-critical
      tolerations:
      - operator: Exists
      containers:
      - name: helper-pod
        image: busybox:1.36
        command:
        - sh
        - -c
        - sleep 3600
        securityContext:
          privileged: true
        volumeMounts:
        - name: hostpath
          mountPath: /host
      volumes:
      - name: hostpath
        hostPath:
          path: /
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: local-path-provisioner
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: local-path-provisioner
  template:
    metadata:
      labels:
        app: local-path-provisioner
    spec:
      serviceAccountName: local-path-provisioner-service-account
      containers:
      - name: local-path-provisioner
        image: rancher/local-path-provisioner:v0.0.24
        imagePullPolicy: IfNotPresent
        command:
        - local-path-provisioner
        - --debug
        - start
        - --config
        - /etc/config/config.json
        volumeMounts:
        - name: config-volume
          mountPath: /etc/config/
        env:
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
      volumes:
      - name: config-volume
        configMap:
          name: local-path-config
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: local-path-provisioner-service-account
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: local-path-provisioner-role
rules:
- apiGroups: [""]
  resources: ["nodes", "persistentvolumeclaims", "configmaps"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["endpoints", "persistentvolumes", "pods"]
  verbs: ["*"]
- apiGroups: [""]
  resources: ["events"]
  verbs: ["create", "patch"]
- apiGroups: ["storage.k8s.io"]
  resources: ["storageclasses"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: local-path-provisioner-bind
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: local-path-provisioner-role
subjects:
- kind: ServiceAccount
  name: local-path-provisioner-service-account
  namespace: kube-system
EOF

# 4. 等待provisioner启动
kubectl wait --for=condition=ready pod -l app=local-path-provisioner -n kube-system --timeout=120s

# 5. 验证StorageClass
kubectl get storageclass

# 应该看到：
# NAME         PROVISIONER              RECLAIMPOLICY   VOLUMEBINDINGMODE      AGE
# local-path   rancher.io/local-path    Delete          WaitForFirstConsumer   10s
# (default)
```

**或者使用官方YAML部署（推荐）**：

```bash
# 1. 删除旧配置
kubectl delete deployment local-path-provisioner -n kube-system 2>/dev/null || true
kubectl delete configmap local-path-config -n kube-system 2>/dev/null || true

# 2. 使用官方YAML部署
kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/v0.0.24/deploy/local-path-storage.yaml

# 3. 修改storageclass为默认
kubectl patch storageclass local-path \
  -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'

# 4. 等待provisioner启动
kubectl wait --for=condition=ready pod -l app=local-path-provisioner -n local-path-storage --timeout=120s

# 5. 验证StorageClass
kubectl get storageclass
```

#### 4.1 部署Harbor（集群1：192.168.31.30）

```bash
# 安装Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# 部署Harbor
helm repo add harbor https://helm.goharbor.io
helm repo update
kubectl create namespace harbor

helm install harbor harbor/harbor \
  --namespace harbor \
  --set expose.type=nodePort \
  --set expose.nodePort.ports.http.nodePort=30002 \
  --set expose.tls.enabled=false \
  --set externalURL=http://192.168.31.30:30002 \
  --set harborAdminPassword="Harbor12345" \
  --set persistence.persistentVolumeClaim.registry.size=50Gi

# 等待就绪
kubectl wait --for=condition=ready pod -l app=harbor -n harbor --timeout=600s
```

#### 4.2 部署Harbor（集群2：192.168.31.31）

```bash
# 安装Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# 部署Harbor
helm repo add harbor https://helm.goharbor.io
helm repo update
kubectl create namespace harbor

helm install harbor harbor/harbor \
  --namespace harbor \
  --set expose.type=nodePort \
  --set expose.nodePort.ports.http.nodePort=30002 \
  --set expose.tls.enabled=false \
  --set externalURL=http://192.168.31.31:30002 \
  --set harborAdminPassword="Harbor12345" \
  --set persistence.persistentVolumeClaim.registry.size=50Gi

kubectl wait --for=condition=ready pod -l app=harbor -n harbor --timeout=600s
```

#### 4.3 部署Istio（集群1：192.168.31.30）

```bash
# 下载Istio
cd /tmp
curl -L https://istio.io/downloadIstio | sh -
cd istio-*
export PATH=$PWD/bin:$PATH

# 安装Istio
cat > istio-cluster1.yaml << 'EOF'
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  profile: default
  values:
    global:
      meshID: mesh1
      multiCluster:
        clusterName: cluster1
      network: network1
EOF

istioctl install -f istio-cluster1.yaml -y

# 降低istiod资源请求（避免内存不足导致调度失败）
kubectl set resources deployment istiod -n istio-system \
  --requests=cpu=100m,memory=128Mi \
  --limits=cpu=1,memory=512Mi

# 等待istiod就绪
kubectl wait --for=condition=ready pod -l app=istiod -n istio-system --timeout=120s

kubectl get pods -n istio-system
```

#### 4.4 部署Istio（集群2：192.168.31.31）

```bash
# 下载Istio
cd /tmp
curl -L https://istio.io/downloadIstio | sh -
cd istio-*
export PATH=$PWD/bin:$PATH

# 安装Istio
cat > istio-cluster2.yaml << 'EOF'
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  profile: default
  values:
    global:
      meshID: mesh1
      multiCluster:
        clusterName: cluster2
      network: network1
EOF

istioctl install -f istio-cluster2.yaml -y

# 降低istiod资源请求（避免内存不足导致调度失败）
kubectl set resources deployment istiod -n istio-system \
  --requests=cpu=100m,memory=128Mi \
  --limits=cpu=1,memory=512Mi

# 等待istiod就绪
kubectl wait --for=condition=ready pod -l app=istiod -n istio-system --timeout=120s

kubectl get pods -n istio-system
```

#### 4.5 部署ArgoCD（集群1：192.168.31.30）

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
kubectl wait --for=condition=available --timeout=600s deployment/argocd-server -n argocd

# 获取密码
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
echo

# 端口转发
kubectl port-forward svc/argocd-server -n argocd 8080:443 --address 0.0.0.0 &
```

#### 4.6 部署Ingress-Nginx（集群1和集群2）

```bash
# 集群1（192.168.31.30）
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/cloud/deploy.yaml
kubectl wait --namespace ingress-nginx --for=condition=ready pod --selector=app.kubernetes.io/component=controller --timeout=120s

# 集群2（192.168.31.31）
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/cloud/deploy.yaml
kubectl wait --namespace ingress-nginx --for=condition=ready pod --selector=app.kubernetes.io/component=controller --timeout=120s
```

---

### 5. 双环境配置

#### 5.1 创建Namespace（集群1和集群2）

```bash
# 集群1（192.168.31.30）
kubectl create namespace gaamingblog-prod
kubectl create namespace gaamingblog-canary
kubectl label namespace gaamingblog-prod environment=production istio-injection=enabled
kubectl label namespace gaamingblog-canary environment=canary istio-injection=enabled

# 集群2（192.168.31.31）
kubectl create namespace gaamingblog-prod
kubectl create namespace gaamingblog-canary
kubectl label namespace gaamingblog-prod environment=production istio-injection=enabled
kubectl label namespace gaamingblog-canary environment=canary istio-injection=enabled
```

#### 5.2 创建Secret（集群1：192.168.31.30）

```bash
# 数据库Secret - 生产环境
kubectl create secret generic gaamingblog-prod-db-secret \
  --from-literal=db-host=mysql-service.default.svc.cluster.local \
  --from-literal=db-port=3306 \
  --from-literal=db-name=gaamingblog_prod \
  --from-literal=db-user=gaamingblog \
  --from-literal=db-password='JiamingBlog@2024#Prod' \
  --namespace=gaamingblog-prod

# 数据库Secret - 开发环境
kubectl create secret generic gaamingblog-canary-db-secret \
  --from-literal=db-host=mysql-service.default.svc.cluster.local \
  --from-literal=db-port=3306 \
  --from-literal=db-name=gaamingblog_canary \
  --from-literal=db-user=gaamingblog \
  --from-literal=db-password='JiamingBlog@2024#Prod' \
  --namespace=gaamingblog-canary

# Harbor Registry Secret - 生产环境
kubectl create secret docker-registry harbor-registry-secret \
  --docker-server=192.168.31.30:30003 \
  --docker-username=admin \
  --docker-password=Harbor12345 \
  --namespace=gaamingblog-prod

# Harbor Registry Secret - 开发环境
kubectl create secret docker-registry harbor-registry-secret \
  --docker-server=192.168.31.30:30003 \
  --docker-username=admin \
  --docker-password=Harbor12345 \
  --namespace=gaamingblog-canary
```

#### 5.3 创建Secret（集群2：192.168.31.31）

```bash
# 数据库Secret - 生产环境
kubectl create secret generic gaamingblog-prod-db-secret \
  --from-literal=db-host=mysql-service.default.svc.cluster.local \
  --from-literal=db-port=3306 \
  --from-literal=db-name=gaamingblog_prod \
  --from-literal=db-user=gaamingblog \
  --from-literal=db-password='JiamingBlog@2024#Prod' \
  --namespace=gaamingblog-prod

# 数据库Secret - 开发环境
kubectl create secret generic gaamingblog-canary-db-secret \
  --from-literal=db-host=mysql-service.default.svc.cluster.local \
  --from-literal=db-port=3306 \
  --from-literal=db-name=gaamingblog_canary \
  --from-literal=db-user=gaamingblog \
  --from-literal=db-password='JiamingBlog@2024#Prod' \
  --namespace=gaamingblog-canary

# Harbor Registry Secret - 生产环境
kubectl create secret docker-registry harbor-registry-secret \
  --docker-server=192.168.31.31:30003 \
  --docker-username=admin \
  --docker-password=Harbor12345 \
  --namespace=gaamingblog-prod

# Harbor Registry Secret - 开发环境
kubectl create secret docker-registry harbor-registry-secret \
  --docker-server=192.168.31.31:30003 \
  --docker-username=admin \
  --docker-password=Harbor12345 \
  --namespace=gaamingblog-canary
```

---

### 6. 验证部署

#### 6.1 验证集群1（192.168.31.30）

```bash
kubectl get nodes -o wide
kubectl get pods -A
kubectl get svc -A
kubectl get namespaces | grep gaamingblog
```

#### 6.2 验证集群2（192.168.31.31）

```bash
kubectl get nodes -o wide
kubectl get pods -A
kubectl get svc -A
kubectl get namespaces | grep gaamingblog
```

---

### 7. 访问地址

- **集群1 API**: https://192.168.31.100:6443 (VIP) 或 https://192.168.31.30:6443 (本地)
- **集群2 API**: https://192.168.31.101:6443 (VIP) 或 https://192.168.31.31:6443 (本地)
- **Harbor集群1**: https://192.168.31.30:30003 (admin/Harbor12345)
- **Harbor集群2**: https://192.168.31.31:30003 (admin/Harbor12345)
- **ArgoCD**: https://192.168.31.30:8080 (admin/获取的密码)

---

### 8. 预计时间

- **系统准备**：30分钟
- **集群1部署**：1小时
- **集群2部署**：1小时
- **基础设施组件**：2小时
- **双环境配置**：30分钟
- **总计**：约5小时

---

## 关键修正说明

### 问题1：VIP配置的鸡生蛋问题

**原问题**：
- kubeadm init 时指定 `controlPlaneEndpoint: "192.168.31.100:6443"` (VIP)
- 但此时 kube-vip 还没部署，VIP 不存在
- 导致初始化失败

**解决方案**：
1. **使用本地IP初始化**：`controlPlaneEndpoint: "192.168.31.30:6443"`
2. **在证书中添加VIP**：通过 `apiServer.certSANs` 预先包含VIP地址
3. **部署kube-vip**：创建VIP地址
4. **更新kubeconfig**：修改为使用VIP地址

### 问题2：证书验证错误

**原问题**：
- 初始化时使用本地IP `192.168.31.30`
- API Server证书只包含本地IP和ClusterIP
- kubeconfig配置使用VIP `192.168.31.100`，导致证书验证失败
- 错误：`x509: certificate is valid for 10.96.0.1, 192.168.31.30, not 192.168.31.100`

**解决方案**：
- 在 `ClusterConfiguration` 中添加 `apiServer.certSANs` 字段
- 预先将VIP地址、主机名等添加到证书的Subject Alternative Names中
- 这样证书同时包含本地IP和VIP，两种方式都可以访问

### 问题3：Pod调度失败（节点NotReady）

**原问题**：
- kube-vip Pod在节点NotReady时无法调度
- 错误：`1 node(s) had untolerated taint {node.kubernetes.io/not-ready: }`
- 形成循环依赖：Calico未安装 → 节点NotReady → kube-vip无法调度

**解决方案**：
1. **调整部署顺序**：先安装Calico，再部署kube-vip
2. **添加tolerations**：kube-vip Pod配置中添加control-plane tolerations
3. **等待节点Ready**：确保节点状态为Ready后再部署应用

### 问题4：kube-vip功能未启用

**原问题**：
- kube-vip启动失败，日志显示：`level=fatal msg="no features are enabled"`
- 原因：缺少 `cp_enable` 环境变量

**解决方案**：
- 添加 `cp_enable: "true"` 环境变量启用控制平面VIP功能
- 单节点场景添加 `vip_leaderelection: "false"`

### 问题5：Harbor安装缺少TLS配置

**原问题**：
- Harbor安装失败，错误：`The "expose.tls.auto.commonName" is required!`
- 原因：Harbor需要配置TLS证书的commonName

**解决方案**：
- 添加 `expose.tls.enabled=true` 启用TLS
- 添加 `expose.tls.auto.commonName=<IP>` 指定证书的commonName

### 问题6：Harbor PVC处于Pending状态

**原问题**：
- Harbor的Redis、Database等Pod处于Pending状态
- PVC无法绑定，状态为Pending
- 错误：`No resources found` (StorageClass)

**原因**：
- Kubernetes集群没有配置StorageClass
- 没有动态存储供应器（Provisioner）
- PVC无法自动创建PV

**解决方案**：
1. **部署local-path-provisioner**：创建默认StorageClass
2. **创建存储目录**：在所有节点创建 `/opt/local-path-provisioner`
3. **验证StorageClass**：确保有默认的StorageClass
4. **重新部署Harbor**：PVC会自动创建PV并绑定

### 问题7：local-path-provisioner启动失败

**原问题**：
- local-path-provisioner Pod崩溃，状态为CrashLoopBackOff
- 错误：`helperPod.yaml is not exist in local-path-config ConfigMap`

**原因**：
- ConfigMap中缺少 `helperPod.yaml` 配置
- local-path-provisioner需要helperPod来执行存储操作

**解决方案**：
- 在ConfigMap中添加 `helperPod.yaml` 配置
- 使用 `registry.k8s.io/pause:3.9` 镜像（Kubernetes节点已有，无需下载）
- 避免使用busybox镜像（可能导致镜像拉取超时）

### 问题8：Helper Pod镜像选择错误

**原问题**：
- Helper Pod启动失败，错误：`exec: "/bin/sh": stat /bin/sh: no such file or directory`
- pause镜像不包含shell，无法执行setup/teardown脚本

**原因**：
- pause镜像只是一个占位容器，只包含pause二进制文件
- helper pod需要执行 `/bin/sh /script/setup` 和 `/bin/sh /script/teardown`
- pause镜像无法执行shell脚本

**解决方案**：
- 使用busybox镜像（包含shell）
- **必须预先在所有节点拉取busybox镜像**：`sudo crictl pull docker.io/library/busybox:1.36.1`
- 避免helper pod启动时拉取镜像超时

### 问题9：Helper Pod缺少setup和teardown脚本

**原问题**：
- Helper Pod无法启动，状态为ContainerCreating
- 错误：`configmap references non-existent config key: setup` 或 `teardown`
- PVC创建超时

**原因**：
- ConfigMap中缺少 `setup` 和 `teardown` 脚本
- local-path-provisioner需要这两个脚本来管理存储目录
  - `setup` - 创建存储目录
  - `teardown` - 删除存储目录

**解决方案**：
- 在ConfigMap中添加 `setup` 和 `teardown` 脚本
- setup脚本用于创建和设置存储目录权限
- teardown脚本用于删除存储目录

### 证书SAN配置说明

```yaml
apiServer:
  certSANs:
  - "192.168.31.100"  # VIP地址 - 允许通过VIP访问
  - "192.168.31.30"   # 本地IP - 允许通过本地IP访问
  - "cluster1-vip"    # VIP主机名
  - "cluster1-master" # Master主机名
  - "kubernetes"      # Kubernetes服务名
  - "kubernetes.default.svc"
  - "kubernetes.default.svc.cluster.local"
  - "10.96.1.1"       # Kubernetes ClusterIP
```

### kube-vip Tolerations配置说明

```yaml
tolerations:
- key: node-role.kubernetes.io/control-plane
  effect: NoSchedule
- key: node-role.kubernetes.io/master
  effect: NoSchedule
```

### kube-vip 环境变量说明

```yaml
env:
- name: vip_arp
  value: "true"              # 启用ARP广播
- name: port
  value: "6443"              # Kubernetes API Server端口
- name: vip_interface
  value: "enp2s0"            # 网络接口名称
- name: address
  value: "192.168.31.100"    # VIP地址
- name: cp_enable
  value: "true"              # 启用控制平面VIP（必须）
- name: vip_leaderelection
  value: "false"             # 单节点禁用领导者选举
```

**重要**：
- `cp_enable` 必须设置为 `true`，否则kube-vip会报错 "no features are enabled"
- 单节点场景下 `vip_leaderelection` 设置为 `false`
- 多节点高可用场景下 `vip_leaderelection` 设置为 `true`

### 优势

- ✅ 避免鸡生蛋问题
- ✅ 初始化过程更可靠
- ✅ 证书同时支持VIP和本地IP访问
- ✅ VIP故障时可以使用本地IP管理集群
- ✅ 避免Pod调度循环依赖
- ✅ 符合Kubernetes最佳实践
