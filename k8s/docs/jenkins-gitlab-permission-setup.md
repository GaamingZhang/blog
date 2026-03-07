# Jenkins 与 GitLab 权限配置指南

## 概述

本文档详细介绍如何配置 Jenkins 流水线以获得推送代码到 GitLab 远程分支的权限。

## 一、GitLab SSH 密钥配置

### 1.1 生成 SSH 密钥对

在 Jenkins 服务器上生成 SSH 密钥对：

```bash
# 切换到 Jenkins 用户
sudo su - jenkins

# 生成 SSH 密钥对（使用空密码）
ssh-keygen -t rsa -b 4096 -C "jenkins@gaaming.com.cn" -f ~/.ssh/id_rsa_jenkins

# 查看公钥
cat ~/.ssh/id_rsa_jenkins.pub
```

### 1.2 在 GitLab 中添加 SSH 密钥

1. 登录 GitLab 项目
2. 点击用户头像 → **Settings** → **SSH Keys**
3. 将上一步生成的公钥内容粘贴到 **Key** 字段
4. 添加一个描述，例如 "Jenkins CI Server"
5. 点击 **Add Key**

### 1.3 测试 SSH 连接

在 Jenkins 服务器上测试连接：

```bash
# 切换到 Jenkins 用户
sudo su - jenkins

# 添加 GitLab 到 known_hosts
ssh-keyscan -H gitlab.com >> ~/.ssh/known_hosts

# 测试连接
ssh -i ~/.ssh/id_rsa_jenkins git@gitlab.com
```

## 二、Jenkins 凭据配置

### 2.1 添加 SSH 凭据

1. 登录 Jenkins 管理界面
2. 点击 **凭据** → **系统** → **全局凭据** → **添加凭据**
3. 选择 **SSH Username with private key**
4. 填写以下信息：
   - **Username**: `git`
   - **Private Key**: 选择 "Enter directly"，粘贴 `~/.ssh/id_rsa_jenkins` 的内容
   - **ID**: `gitlab-ssh-key`（用于在流水线中引用）
   - **Description**: GitLab SSH Key for CI/CD
5. 点击 **确定**

### 2.2 配置 Git 插件

确保 Jenkins 安装了以下插件：
- Git Plugin
- Git Credentials Plugin

## 三、Jenkins 流水线配置

### 3.1 更新 updateVersion.Jenkinsfile

确保流水线使用正确的凭据：

```groovy
stage('Checkout') {
  steps {
    checkout([
      $class: 'GitSCM',
      branches: [[name: env.BRANCH_NAME]],
      userRemoteConfigs: [[
        url: 'git@gitlab.com:gaamingzhang/gaamingzhang-blog.git',
        credentialsId: 'gitlab-ssh-key'
      ]]
    ])
  }
}
```

### 3.2 配置 Git 推送权限

在流水线的 Git 操作步骤中，确保使用正确的 SSH 密钥：

```groovy
stage('Push Changes') {
  steps {
    script {
      // 推送提交
      sh """
        GIT_SSH_COMMAND='ssh -i ~/.ssh/id_rsa_jenkins' git push origin ${env.BRANCH_NAME}
      """
      echo "Pushed changes to origin"
    }
  }
}
```

## 四、GitLab 项目权限设置

### 4.1 确保 Jenkins 用户有推送权限

1. 在 GitLab 项目中，点击 **Settings** → **Members**
2. 添加 Jenkins 使用的 GitLab 用户
3. 设置角色为 **Developer** 或更高（需要推送权限）
4. 点击 **Add to project**

### 4.2 分支保护设置

如果目标分支有保护设置，需要：

1. 点击 **Settings** → **Repository** → **Protected branches**
2. 找到目标分支（例如 `main`）
3. 确保 Jenkins 用户或其所在组有 **Allowed to push** 权限
4. 如果需要，可以添加 Jenkins 用户到允许推送的用户列表

## 五、测试验证

### 5.1 运行测试流水线

1. 在 Jenkins 中创建一个测试任务，使用 `updateVersion.Jenkinsfile`
2. 运行流水线，观察是否成功：
   - 版本号是否更新
   - Git 提交是否成功
   - 推送是否成功
   - 标签是否创建

### 5.2 验证 GitLab 状态

1. 登录 GitLab 项目
2. 检查 **Commits** 页面，确认新的提交
3. 检查 **Tags** 页面，确认新的标签
4. 检查版本文件内容是否更新

## 六、常见问题排查

### 6.1 SSH 连接失败

**症状**：`Permission denied (publickey)`

**解决方法**：
- 检查 SSH 密钥是否正确添加到 GitLab
- 确保 Jenkins 运行用户拥有 SSH 密钥文件的正确权限
- 测试 SSH 连接命令

### 6.2 推送被拒绝

**症状**：`remote: You are not allowed to push code to this branch.`

**解决方法**：
- 检查 GitLab 项目权限设置
- 确保 Jenkins 用户有推送权限
- 检查分支保护规则

### 6.3 凭据未找到

**症状**：`CredentialsId "gitlab-ssh-key" not found`

**解决方法**：
- 检查 Jenkins 凭据配置
- 确保凭据 ID 与流水线中使用的一致
- 验证凭据权限

## 七、最佳实践

1. **使用专用 SSH 密钥**：为 Jenkins 创建专用的 SSH 密钥，避免使用个人密钥
2. **定期轮换密钥**：定期更新 SSH 密钥以提高安全性
3. **使用凭据绑定**：在流水线中使用 `withCredentials` 绑定 SSH 密钥
4. **日志记录**：在流水线中添加详细的日志，便于排查问题
5. **错误处理**：添加适当的错误处理和重试机制

## 八、完整的流水线示例

以下是完整的 `updateVersion.Jenkinsfile` 示例，包含权限配置：

```groovy
pipeline {
  agent any

  environment {
    VERSION_FILE = 'k8s/jenkins/version'
    GITLAB_REPO = 'git@gitlab.com:gaamingzhang/gaamingzhang-blog.git'
  }

  stages {
    stage('Checkout') {
      steps {
        checkout([
          $class: 'GitSCM',
          branches: [[name: env.BRANCH_NAME]],
          userRemoteConfigs: [[
            url: env.GITLAB_REPO,
            credentialsId: 'gitlab-ssh-key'
          ]]
        ])
      }
    }

    stage('Read Current Version') {
      steps {
        script {
          // 读取当前版本号
          if (fileExists(VERSION_FILE)) {
            def currentVersion = readFile(VERSION_FILE).trim()
            echo "Current version: ${currentVersion}"
            env.CURRENT_VERSION = currentVersion
          } else {
            // 如果文件不存在，使用默认版本
            echo "Version file not found, using default version 1.0.0"
            env.CURRENT_VERSION = '1.0.0'
          }
        }
      }
    }

    stage('Increment Version') {
      steps {
        script {
          // 解析版本号并递增
          def versionParts = env.CURRENT_VERSION.split('\\.')
          def major = versionParts[0].toInteger()
          def minor = versionParts[1].toInteger()
          def patch = versionParts[2].toInteger()

          // 递增补丁号
          patch++
          
          // 处理进位
          if (patch > 99) {
            patch = 0
            minor++
            if (minor > 99) {
              minor = 0
              major++
              if (major > 99) {
                error "Version number exceeds maximum limit (99.99.99)"
              }
            }
          }

          // 生成新版本号
          def newVersion = "${major}.${minor}.${patch}"
          echo "New version: ${newVersion}"
          env.NEW_VERSION = newVersion
        }
      }
    }

    stage('Update Version File') {
      steps {
        script {
          // 更新版本文件
          writeFile file: VERSION_FILE, text: env.NEW_VERSION
          echo "Updated version file to ${env.NEW_VERSION}"
        }
      }
    }

    stage('Git Commit') {
      steps {
        script {
          // 提交更改到git
          sh """
            git config user.name "Jenkins CI"
            git config user.email "jenkins@gaaming.com.cn"
            git add ${VERSION_FILE}
            git commit -m "Update version to ${env.NEW_VERSION}"
          """
        }
      }
    }

    stage('Push Changes') {
      steps {
        script {
          // 推送提交
          withCredentials([sshUserPrivateKey(credentialsId: 'gitlab-ssh-key', keyFileVariable: 'SSH_KEY')]) {
            sh """
              GIT_SSH_COMMAND='ssh -i ${SSH_KEY}' git push origin ${env.BRANCH_NAME}
            """
          }
          echo "Pushed changes to origin"
        }
      }
    }

    stage('Create Git Tag') {
      steps {
        script {
          // 创建git tag
          withCredentials([sshUserPrivateKey(credentialsId: 'gitlab-ssh-key', keyFileVariable: 'SSH_KEY')]) {
            sh """
              GIT_SSH_COMMAND='ssh -i ${SSH_KEY}' git tag -a v${env.NEW_VERSION} -m 'Version ${env.NEW_VERSION}'
              GIT_SSH_COMMAND='ssh -i ${SSH_KEY}' git push origin v${env.NEW_VERSION}
            """
          }
          echo "Created and pushed tag v${env.NEW_VERSION}"
        }
      }
    }

    stage('Get Remote Commit ID') {
      steps {
        script {
          // 从远程获取最新的commit id
          withCredentials([sshUserPrivateKey(credentialsId: 'gitlab-ssh-key', keyFileVariable: 'SSH_KEY')]) {
            sh """
              GIT_SSH_COMMAND='ssh -i ${SSH_KEY}' git fetch origin
            """
          }
          def commitId = sh(script: "git rev-parse origin/${env.BRANCH_NAME}", returnStdout: true).trim()
          echo "Remote Commit ID: ${commitId}"
          env.COMMIT_ID = commitId
        }
      }
    }
  }

  post {
    success {
      echo "Version update completed successfully!"
      echo "New version: ${env.NEW_VERSION}"
      echo "Commit ID: ${env.COMMIT_ID}"
      
      // 返回commit id
      writeFile file: 'commit_id.txt', text: env.COMMIT_ID
    }
    failure {
      echo "Version update failed!"
    }
  }
}
```

## 九、总结

通过以上配置，Jenkins 流水线将能够：

1. 使用 SSH 密钥认证连接到 GitLab
2. 推送代码变更到远程分支
3. 创建和推送 Git 标签
4. 从远程获取准确的 commit id

这些配置确保了 CI/CD 流水线的顺利运行，实现了版本号的自动管理和代码的自动部署。