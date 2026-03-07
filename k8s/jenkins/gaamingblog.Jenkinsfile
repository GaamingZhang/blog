pipeline {
  agent any

  environment {
    VERSION_FILE = 'k8s/jenkins/version'
    GIT_REMOTE = 'git@192.168.31.50:gaamingzhang/blog.git'
    WORKDIR = 'workspace'
  }

  stages {
    stage('Trigger updateVersion') {
      steps {
        script {
          // 触发updateVersion.Jenkinsfile的构建并等待完成
          build job: 'updateVersion', wait: true
          echo "updateVersion build completed"
        }
      }
    }

    stage('Checkout Official Branch') {
      steps {
        script {
          def workdir = env.WORKDIR
          
          // 删除workdir目录及其内容
          sh "rm -rf ${workdir}"
          
          // 创建workdir目录
          sh "mkdir -p ${workdir}"
          
          // 切换到workdir目录
          dir(workdir) {
            withCredentials([sshUserPrivateKey(credentialsId: 'Jenkins_Pipeline_Agent_SSH_Key', keyFileVariable: 'SSH_KEY')]) {
              // 克隆仓库
              sh 'GIT_SSH_COMMAND="ssh -i $SSH_KEY" git clone $GIT_REMOTE .'
              
              // 读取版本号
              def version = readFile(VERSION_FILE).trim()
              echo "Current version: ${version}"
              
              // 切换到official分支
              def officialBranch = "official.${version}"
              sh "git checkout ${officialBranch}"
              echo "Switched to branch: ${officialBranch}"
            }
          }
        }
      }
    }
  }

  post {
    success {
      echo "Pipeline completed successfully!"
    }
    failure {
      echo "Pipeline failed!"
    }
  }
}
