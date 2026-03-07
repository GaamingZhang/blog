pipeline {
  agent any

  environment {
    VERSION_FILE = 'k8s/jenkins/version'
    GIT_REMOTE = 'git@192.168.31.50:gaamingzhang/blog.git'
    GIT_BRANCH = 'main'
  }

  stages {
    stage('Checkout') {
      steps {
        script {
          // 检查是否为git仓库
          def isGitRepo = sh(script: 'git rev-parse --is-inside-work-tree 2>/dev/null || echo "false"', returnStdout: true).trim()
          
          withCredentials([sshUserPrivateKey(credentialsId: 'Jenkins_Pipeline_Agent_SSH_Key', keyFileVariable: 'SSH_KEY')]) {
            if (isGitRepo == "false") {
              // 如果不是git仓库，克隆仓库
              sh '''
                GIT_SSH_COMMAND="ssh -i ${SSH_KEY}" git clone ${GIT_REMOTE} .
              '''
              echo "Cloned repository from ${GIT_REMOTE}"
            } else {
              // 如果是git仓库，更新远程地址并拉取代码
              sh '''
                git remote set-url origin ${GIT_REMOTE}
                git pull origin ${GIT_BRANCH}
              '''
              echo "Pull from origin ${GIT_BRANCH} branch"
            }
          }
        }
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
          sh '''
            git config user.name "Jenkins CI"
            git config user.email "jenkins@gaaming.com.cn"
            git add ${VERSION_FILE}
            git commit -m "Update version to ${env.NEW_VERSION}"
          '''
        }
      }
    }

    stage('Push Changes') {
      steps {
        script {
          // 推送提交
          withCredentials([sshUserPrivateKey(credentialsId: 'Jenkins_Pipeline_Agent_SSH_Key', keyFileVariable: 'SSH_KEY')]) {
            sh '''
              GIT_SSH_COMMAND="ssh -i ${SSH_KEY}" git push ${GIT_REMOTE} ${env.BRANCH_NAME}
            '''
          }
          echo "Pushed changes to origin"
        }
      }
    }

    stage('Create Git Tag and Branch') {
      steps {
        script {
          // 创建git tag和official分支
          def officialBranch = "official.${env.NEW_VERSION}"
          withCredentials([sshUserPrivateKey(credentialsId: 'Jenkins_Pipeline_Agent_SSH_Key', keyFileVariable: 'SSH_KEY')]) {
            sh '''
              GIT_SSH_COMMAND="ssh -i ${SSH_KEY}" git tag -a v${env.NEW_VERSION} -m "Version ${env.NEW_VERSION}"
              GIT_SSH_COMMAND="ssh -i ${SSH_KEY}" git push ${GIT_REMOTE} v${env.NEW_VERSION}
              
              # 创建并推送official分支
              GIT_SSH_COMMAND="ssh -i ${SSH_KEY}" git checkout -b ${officialBranch}
              GIT_SSH_COMMAND="ssh -i ${SSH_KEY}" git push ${GIT_REMOTE} ${officialBranch}
              
              # 切回原分支
              GIT_SSH_COMMAND="ssh -i ${SSH_KEY}" git checkout ${env.BRANCH_NAME}
            '''
          }
          echo "Created and pushed tag v${env.NEW_VERSION}"
          echo "Created and pushed branch ${officialBranch}"
        }
      }
    }
  }

  post {
    success {
      echo "Version update completed successfully!"
      echo "New version: ${env.NEW_VERSION}"
    }
    failure {
      echo "Version update failed!"
    }
  }
}
