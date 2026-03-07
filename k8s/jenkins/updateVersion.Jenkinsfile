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
          def gitRemote = env.GIT_REMOTE
          
          withCredentials([sshUserPrivateKey(credentialsId: 'Jenkins_Pipeline_Agent_SSH_Key', keyFileVariable: 'SSH_KEY')]) {
            sh 'GIT_SSH_COMMAND="ssh -i $SSH_KEY" git clone $GIT_REMOTE .'
            echo "Cloned repository from ${gitRemote}"
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
          def versionFile = env.VERSION_FILE
          def newVersion = env.NEW_VERSION
          sh "git config user.name 'Jenkins CI'"
          sh "git config user.email 'jenkins@gaaming.com.cn'"
          sh "git add ${versionFile}"
          sh "git commit -m 'Update version to ${newVersion}'"
        }
      }
    }

    stage('Push Changes') {
      steps {
        script {
          // 推送提交
          def gitRemote = env.GIT_REMOTE
          def branchName = env.BRANCH_NAME
          withCredentials([sshUserPrivateKey(credentialsId: 'Jenkins_Pipeline_Agent_SSH_Key', keyFileVariable: 'SSH_KEY')]) {
            sh 'GIT_SSH_COMMAND="ssh -i $SSH_KEY" git push $GIT_REMOTE $BRANCH_NAME'
          }
          echo "Pushed changes to origin"
        }
      }
    }

    stage('Create Git Tag and Branch') {
      steps {
        script {
          // 创建git tag和official分支
          env.OFFICIAL_BRANCH = "official.${env.NEW_VERSION}"
          def newVersion = env.NEW_VERSION
          def officialBranch = env.OFFICIAL_BRANCH
          def gitRemote = env.GIT_REMOTE
          def branchName = env.BRANCH_NAME
          
          // 创建tag（本地操作，不需要SSH）
          sh "git tag -a v${newVersion} -m 'Version ${newVersion}'"
          
          // 创建分支（本地操作，不需要SSH）
          sh "git checkout -b ${officialBranch}"
          
          withCredentials([sshUserPrivateKey(credentialsId: 'Jenkins_Pipeline_Agent_SSH_Key', keyFileVariable: 'SSH_KEY')]) {
            // 推送tag
            sh 'GIT_SSH_COMMAND="ssh -i $SSH_KEY" git push $GIT_REMOTE v$NEW_VERSION'
            // 推送分支
            sh 'GIT_SSH_COMMAND="ssh -i $SSH_KEY" git push $GIT_REMOTE $OFFICIAL_BRANCH'
          }
          
          // 切回原分支（本地操作，不需要SSH）
          sh "git checkout ${branchName}"
          
          echo "Created and pushed tag v${newVersion}"
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
