# Harbor 权限配置指南

## 概述

本文档详细介绍如何配置 Harbor 权限，让 Jenkins 流水线能够推送镜像到 Harbor 镜像仓库。

## 一、Harbor 用户和项目配置

### 1.1 创建 Harbor 用户

1. 登录 Harbor Web 界面（例如：https://192.168.31.30:30003）
2. 点击 **系统管理** → **用户/成员**
3. 点击 **新建用户**
4. 填写用户信息：
   - **用户名**: `jenkins`
   - **邮箱**: `jenkins@gaaming.com.cn`
   - **密码**: 设置强密码（建议使用自动生成的密码）
   - **角色**: 选择 `项目管理员` 或 `开发人员`
5. 点击 **确定** 创建用户

### 1.2 创建 Harbor 项目

1. 在 Harbor 中点击 **项目** → **新建项目**
2. 填写项目信息：
   - **项目名称**: `gaamingzhang`
   - **访问级别**: `公开` 或 `私有`
   - **项目配额**: 根据需要设置
3. 点击 **确定** 创建项目

### 1.3 配置项目成员权限

1. 进入项目 `gaamingzhang`
2. 点击 **成员** 标签
3. 点击 **添加成员**
4. 添加用户 `jenkins`
5. 设置角色为 `开发人员` 或 `项目管理员`
6. 点击 **确定**

### 1.4 配置机器人账户（推荐方式）

为了更安全，建议使用机器人账户而不是普通用户：

1. 在项目 `gaamingzhang` 中，点击 **机器人账户** 标签
2. 点击 **新建机器人账户**
3. 填写信息：
   - **名称**: `jenkins-pipeline`
   - **描述**: Jenkins CI/CD Pipeline Account
   - **过期时间**: 选择 `永不过期` 或设置过期时间
4. 点击 **确定**
5. **重要**: 保存生成的机器人账户令牌（只显示一次）

## 二、Jenkins 凭据配置

### 2.1 添加用户名密码凭据

#### 方式一：使用普通用户凭据

1. 登录 Jenkins Web 界面
2. 点击 **凭据** → **系统** → **全局凭据** → **添加凭据**
3. 选择 **Username with password**
4. 填写信息：
   - **范围**: 全局
   - **类型**: Username with password
   - **用户名**: `jenkins`（Harbor 用户名）
   - **密码**: Harbor 用户密码
   - **ID**: `harbor-cluster1-credentials`
   - **描述**: Harbor Cluster1 Credentials
5. 点击 **确定**

#### 方式二：使用机器人账户凭据（推荐）

1. 登录 Jenkins Web 界面
2. 点击 **凭据** → **系统** → **全局凭据** → **添加凭据**
3. 选择 **Username with password**
4. 填写信息：
   - **范围**: 全局
   - **类型**: Username with password
   - **用户名**: `robot$jenkins-pipeline`（机器人账户名）
   - **密码**: 机器人账户令牌
   - **ID**: `harbor-cluster1-credentials`
   - **描述**: Harbor Cluster1 Robot Account
5. 点击 **确定**

### 2.2 为第二个集群配置凭据

重复上述步骤，创建 `harbor-cluster2-credentials` 凭据：
- **ID**: `harbor-cluster2-credentials`
- **用户名**: Harbor 集群2 的用户名或机器人账户名
- **密码**: Harbor 集群2 的密码或机器人账户令牌

## 三、验证配置

### 3.1 测试 Harbor 登录

在 Jenkins 服务器上手动测试登录：

```bash
# 测试集群1登录
docker login 192.168.31.30:30003
# 输入用户名和密码

# 测试集群2登录
docker login 192.168.31.31:30003
# 输入用户名和密码
```

### 3.2 测试镜像推送

```bash
# 拉取测试镜像
docker pull nginx:alpine

# 标记镜像
docker tag nginx:alpine 192.168.31.30:30003/gaamingzhang/blog:test
docker tag nginx:alpine 192.168.31.31:30003/gaamingzhang/blog:test

# 推送镜像到集群1
docker push 192.168.31.30:30003/gaamingzhang/blog:test

# 推送镜像到集群2
docker push 192.168.31.31:30003/gaamingzhang/blog:test
```

## 四、Jenkins 流水线中使用凭据

### 4.1 在 Jenkinsfile 中使用凭据

```groovy
stage('Push to Harbor - Cluster1') {
  steps {
    script {
      withCredentials([usernamePassword(credentialsId: 'harbor-cluster1-credentials', usernameVariable: 'HARBOR_USER', passwordVariable: 'HARBOR_PASSWORD')]) {
        sh """
          echo "${HARBOR_PASSWORD}" | docker login 192.168.31.30:30003 -u "${HARBOR_USER}" --password-stdin
          docker push 192.168.31.30:30003/gaamingzhang/blog:${IMAGE_TAG}
        """
      }
    }
  }
}
```

### 4.2 安全最佳实践

1. **使用机器人账户**: 机器人账户比普通用户更安全，可以设置细粒度权限
2. **定期轮换凭据**: 定期更新 Harbor 密码和 Jenkins 凭据
3. **最小权限原则**: 只授予必要的推送权限，避免授予管理权限
4. **审计日志**: 定期检查 Harbor 访问日志，监控异常访问

## 五、常见问题排查

### 5.1 登录失败

**错误**: `Error: denied: requested access to the resource is denied`

**解决方案**:
- 检查用户名和密码是否正确
- 础认用户有项目访问权限
- 检查项目是否存在

### 5.2 推送失败

**错误**: `denied: requested access to the resource is denied`

**解决方案**:
- 确认用户有推送权限
- 检查项目配额是否足够
- 确认镜像名称和标签格式正确

### 5.3 网络问题

**错误**: `Error: Get https://192.168.31.30:30003/v2/: dial tcp 192.168.31.30:30003: connect: connection refused`

**解决方案**:
- 检查 Harbor 服务是否运行
- 确认网络连接正常
- 检查防火墙设置

## 六、Harbor 配置清单

### 6.1 集群1 (192.168.31.30:30003)

- [ ] 创建 Harbor 用户 `jenkins`
- [ ] 创建项目 `gaamingzhang`
- [ ] 配置用户项目权限
- [ ] 创建 Jenkins 凭据 `harbor-cluster1-credentials`
- [ ] 测试登录和推送

### 6.2 集群2 (192.168.31.31:30003)

- [ ] 创建 Harbor 用户 `jenkins`
- [ ] 创建项目 `gaamingzhang`
- [ ] 配置用户项目权限
- [ ] 创建 Jenkins 凭据 `harbor-cluster2-credentials`
- [ ] 测试登录和推送

## 七、高级配置（可选）

### 7.1 配置 HTTPS

如果 Harbor 使用 HTTPS：

1. 在 Jenkins 服务器上添加证书信任：
```bash
# 添加 Harbor CA 证书到 Docker 信任列表
mkdir -p /etc/docker/certs.d
echo "Harbor CA certificate" >> /etc/docker/certs.d/harbor-ca.crt
systemctl restart docker
```

2. 在 Jenkinsfile 中使用 HTTPS：
```groovy
environment {
  HARBOR_URL_CLUSTER1 = 'https://192.168.31.30:30003'
  HARBOR_URL_CLUSTER2 = 'https://192.168.31.31:30003'
}
```

### 7.2 配置镜像签名

如果需要镜像签名验证：

1. 在 Harbor 中配置签名策略
2. 在 Jenkins 中配置签名密钥
3. 在 Jenkinsfile 中添加签名步骤

## 八、总结

通过以上步骤，Jenkins 流水线将能够：
1. 使用安全的凭据访问 Harbor
2. 推送镜像到指定的 Harbor 项目
3. 支持多个 Kubernetes 集群的镜像分发

确保按照安全最佳实践配置权限，定期审计访问日志，保障系统安全。
