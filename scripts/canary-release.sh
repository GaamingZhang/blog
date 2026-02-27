#!/bin/bash

# 金丝雀发布脚本
# 用法: ./canary-release.sh <weight>

WEIGHT=${1:-5}
STABLE_WEIGHT=$((100 - WEIGHT))

echo "设置金丝雀流量权重为 ${WEIGHT}%"

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
      weight: ${STABLE_WEIGHT}
    - destination:
        host: gaamingzhang-blog-service
        subset: canary
      weight: ${WEIGHT}
EOF

echo "流量分配已更新:"
echo "- 稳定版本: ${STABLE_WEIGHT}%"
echo "- 金丝雀版本: ${WEIGHT}%"
