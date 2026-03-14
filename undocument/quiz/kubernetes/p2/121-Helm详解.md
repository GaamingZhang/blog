---
date: 2026-03-13
author: Gaaming Zhang
isOriginal: false
article: true
category:
  - Kubernetes
tag:
  - Kubernetes
  - Helm
  - 包管理
  - Chart
---

# Kubernetes Helm 详解

## 引言

Helm 是 Kubernetes 的包管理工具，被称为"Kubernetes 的 apt/yum"。它将 Kubernetes 资源打包成 Chart，实现应用的标准化部署和管理。通过 Helm，可以简化复杂应用的部署流程，实现配置的复用和版本管理。

## Helm 概述

### Helm 核心概念

```
┌─────────────────────────────────────────────────────────────┐
│                  Helm 核心概念                               │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Chart：                                                     │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  • Helm 的打包格式，类似于 deb/rpm 包               │   │
│  │  • 包含 Kubernetes 资源模板                         │   │
│  │  • 可配置参数（values.yaml）                        │   │
│  │  • 可版本化、可共享                                 │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  Release：                                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  • Chart 的一次安装实例                             │   │
│  │  • 每次安装产生一个 Release                         │   │
│  │  • 同一 Chart 可安装多次                            │   │
│  │  • Release 名称唯一                                 │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  Repository：                                                │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  • Chart 的存储仓库                                 │   │
│  │  • 可以是本地或远程                                 │   │
│  │  • 支持公共和私有仓库                               │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Helm 架构

```
┌─────────────────────────────────────────────────────────────┐
│                  Helm 架构                                   │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                    Helm CLI                           │  │
│  │  • 本地 Chart 开发                                   │  │
│  │  • 仓库管理                                          │  │
│  │  • Release 管理                                      │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  Chart Repository                     │  │
│  │  • 存储 Chart 包                                     │  │
│  │  • 提供 index.yaml                                   │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  Kubernetes Cluster                   │  │
│  │  • 部署 Release                                      │  │
│  │  • 存储状态（ConfigMap/Secret）                      │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Helm 安装与配置

### 安装 Helm

```bash
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

helm version
```

### 添加仓库

```bash
helm repo add bitnami https://charts.bitnami.com/bitnami

helm repo add stable https://charts.helm.sh/stable

helm repo update

helm repo list
```

### 搜索 Chart

```bash
helm search repo nginx

helm search hub wordpress

helm show chart bitnami/nginx

helm show readme bitnami/nginx

helm show values bitnami/nginx
```

## Chart 结构

### 标准 Chart 目录结构

```
mychart/
├── Chart.yaml          # Chart 元数据
├── values.yaml         # 默认配置值
├── charts/             # 依赖的 Chart
├── templates/          # 模板文件
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── configmap.yaml
│   ├── _helpers.tpl    # 模板助手
│   └── NOTES.txt       # 安装说明
└── .helmignore         # 打包时忽略的文件
```

### Chart.yaml 示例

```yaml
apiVersion: v2
name: mychart
description: A Helm chart for Kubernetes
type: application
version: 1.0.0
appVersion: "1.0.0"
maintainers:
  - name: Gaaming Zhang
    email: example@example.com
dependencies:
  - name: redis
    version: "16.x.x"
    repository: "https://charts.bitnami.com/bitnami"
```

### values.yaml 示例

```yaml
replicaCount: 3

image:
  repository: nginx
  tag: "1.21"
  pullPolicy: IfNotPresent

service:
  type: ClusterIP
  port: 80

resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 512Mi

ingress:
  enabled: false
  className: ""
  hosts:
    - host: chart-example.local
      paths:
        - path: /
          pathType: ImplementationSpecific
```

## 模板语法

### 基本模板

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "mychart.fullname" . }}
  labels:
    {{- include "mychart.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      {{- include "mychart.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "mychart.selectorLabels" . | nindent 8 }}
    spec:
      containers:
        - name: {{ .Chart.Name }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          ports:
            - containerPort: {{ .Values.service.port }}
```

### 控制结构

```yaml
{{- if .Values.ingress.enabled -}}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ include "mychart.fullname" . }}
  annotations:
    {{- toYaml .Values.ingress.annotations | nindent 4 }}
spec:
  {{- if .Values.ingress.className }}
  ingressClassName: {{ .Values.ingress.className }}
  {{- end }}
  rules:
    {{- range .Values.ingress.hosts }}
    - host: {{ .host | quote }}
      http:
        paths:
          {{- range .paths }}
          - path: {{ .path }}
            pathType: {{ .pathType }}
            backend:
              service:
                name: {{ include "mychart.fullname" $ }}
                port:
                  number: {{ $.Values.service.port }}
          {{- end }}
    {{- end }}
{{- end }}
```

### 助手模板

```yaml
{{- define "mychart.labels" -}}
helm.sh/chart: {{ include "mychart.chart" . }}
{{ include "mychart.selectorLabels" . }}
{{- if .Chart.AppVersion -}}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end -}}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "mychart.selectorLabels" -}}
app.kubernetes.io/name: {{ include "mychart.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "mychart.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}
```

## Helm 常用命令

### 安装与升级

```bash
helm install my-release bitnami/nginx

helm install my-release bitnami/nginx -f values.yaml

helm install my-release bitnami/nginx --set replicaCount=3

helm upgrade my-release bitnami/nginx

helm upgrade my-release bitnami/nginx --set service.type=LoadBalancer

helm upgrade --install my-release bitnami/nginx
```

### 查看与管理

```bash
helm list

helm list -n <namespace>

helm status my-release

helm get all my-release

helm get values my-release

helm get manifest my-release

helm history my-release
```

### 卸载与回滚

```bash
helm uninstall my-release

helm uninstall my-release --keep-history

helm rollback my-release 1

helm rollback my-release
```

## 自定义 Chart 开发

### 创建 Chart

```bash
helm create mychart

helm lint mychart

helm template mychart

helm package mychart
```

### 调试 Chart

```bash
helm template mychart -f values.yaml

helm template mychart --debug

helm install my-release mychart --dry-run --debug
```

## Helm 最佳实践

### 1. 版本控制

```yaml
apiVersion: v2
name: mychart
version: 1.0.0
appVersion: "1.0.0"
```

### 2. 使用 values.yaml

```yaml
helm install my-release mychart -f custom-values.yaml
```

### 3. 命名规范

```yaml
{{- define "mychart.fullname" -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
```

### 4. 文档说明

```markdown
{{ template "chart.header" . }}
{{ template "chart.versionBadge" . }}
{{ template "chart.typeBadge" . }}
{{ template "chart.appVersionBadge" . }}

{{ template "chart.description" . }}

{{ template "chart.homepageLine" . }}
```

## 面试回答

**问题**: 什么是 Helm？

**回答**: Helm 是 Kubernetes 的包管理工具，被称为"Kubernetes 的 apt/yum"。

**核心概念**：**Chart** 是 Helm 的打包格式，包含 Kubernetes 资源模板和可配置参数，类似于 deb/rpm 包；**Release** 是 Chart 的一次安装实例，每次安装产生一个 Release，同一 Chart 可安装多次；**Repository** 是 Chart 的存储仓库，支持公共和私有仓库。

**主要功能**：**应用打包**，将 Kubernetes 资源打包成 Chart，实现标准化部署；**版本管理**，支持应用版本控制和回滚；**配置管理**，通过 values.yaml 实现配置复用和定制；**依赖管理**，支持 Chart 依赖关系管理。

**常用命令**：`helm install` 安装 Chart；`helm upgrade` 升级 Release；`helm rollback` 回滚 Release；`helm uninstall` 卸载 Release；`helm list` 查看所有 Release；`helm repo add` 添加仓库；`helm search` 搜索 Chart。

**Chart 结构**：Chart.yaml（元数据）、values.yaml（默认配置）、templates/（模板文件）、charts/（依赖 Chart）。

**最佳实践**：使用版本控制管理 Chart；通过 values.yaml 定制配置；遵循命名规范；编写完善的文档说明；使用 helm lint 检查 Chart；使用 --dry-run 调试模板。
