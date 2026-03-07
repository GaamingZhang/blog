pipeline {
  agent any

  tools {
    nodejs 'NodeJS'
  }

  environment {
    VERSION_FILE = 'k8s/jenkins/version'
    GIT_REMOTE = 'git@192.168.31.50:gaamingzhang/blog.git'
    WORKDIR = 'kubernetes_deploy_workspace'
    UPDATE_VERSION_JOB = 'GaamingBlogUpdateVersion'
    HARBOR_URL_CLUSTER1 = '192.168.31.30:30002'
    HARBOR_URL_CLUSTER2 = '192.168.31.31:30002'
    IMAGE_NAME = 'gaaming/blog'
    ARGOCD_GIT_REMOTE = 'git@192.168.31.50:gaamingzhang/gaamingblogkubernetesargocd.git'
    ARGOCD_WORKDIR = 'argocd_workspace'
  }

  stages {
    stage('Trigger updateVersion') {
      steps {
        script {
          // 触发updateVersion.Jenkinsfile的构建并等待完成
          build job: env.UPDATE_VERSION_JOB, wait: true
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
              env.IMAGE_TAG = version
              
              // 切换到official分支
              def officialBranch = "official.${version}"
              sh "git checkout ${officialBranch}"
              echo "Switched to branch: ${officialBranch}"
            }
          }
        }
      }
    }

    stage('Build Image') {
      steps {
        script {
          // 切换到workdir目录
          dir(env.WORKDIR) {
            def imageTag = env.IMAGE_TAG
            
            // 构建镜像
            sh "docker build -t ${env.IMAGE_NAME}:${imageTag} -t ${env.IMAGE_NAME}:latest ."
            echo "Built image: ${env.IMAGE_NAME}:${imageTag}"
          }
        }
      }
    }

    stage('Push to Harbor - Cluster1') {
      steps {
        script {
          dir(env.WORKDIR) {
            withCredentials([
              string(credentialsId: 'Harbor_Robot_Account_Name_Cluster1', variable: 'HARBOR_USER'),
              string(credentialsId: 'Harbor_Robot_Account_Token_Cluster1', variable: 'HARBOR_PASSWORD')
            ]) {
              withEnv([
                "IMAGE_TAG=${env.IMAGE_TAG}",
                "HARBOR_URL=${env.HARBOR_URL_CLUSTER1}",
                "IMAGE_NAME=${env.IMAGE_NAME}"
              ]) {
                sh '''
                  docker tag ${IMAGE_NAME}:${IMAGE_TAG} ${HARBOR_URL}/${IMAGE_NAME}:${IMAGE_TAG}
                  docker tag ${IMAGE_NAME}:${IMAGE_TAG} ${HARBOR_URL}/${IMAGE_NAME}:latest
                  echo "${HARBOR_PASSWORD}" | docker login ${HARBOR_URL} -u "${HARBOR_USER}" --password-stdin
                  docker push ${HARBOR_URL}/${IMAGE_NAME}:${IMAGE_TAG}
                  docker push ${HARBOR_URL}/${IMAGE_NAME}:latest
                '''
              }
            }
            echo "Pushed image to Harbor Cluster1: ${env.HARBOR_URL_CLUSTER1}/${env.IMAGE_NAME}:${env.IMAGE_TAG}"
          }
        }
      }
    }

    stage('Push to Harbor - Cluster2') {
      steps {
        script {
          dir(env.WORKDIR) {
            withCredentials([
              string(credentialsId: 'Harbor_Robot_Account_Name_Cluster2', variable: 'HARBOR_USER'),
              string(credentialsId: 'Harbor_Robot_Account_Token_Cluster2', variable: 'HARBOR_PASSWORD')
            ]) {
              withEnv([
                "IMAGE_TAG=${env.IMAGE_TAG}",
                "HARBOR_URL=${env.HARBOR_URL_CLUSTER2}",
                "IMAGE_NAME=${env.IMAGE_NAME}"
              ]) {
                sh '''
                  docker tag ${IMAGE_NAME}:${IMAGE_TAG} ${HARBOR_URL}/${IMAGE_NAME}:${IMAGE_TAG}
                  docker tag ${IMAGE_NAME}:${IMAGE_TAG} ${HARBOR_URL}/${IMAGE_NAME}:latest
                  echo "${HARBOR_PASSWORD}" | docker login ${HARBOR_URL} -u "${HARBOR_USER}" --password-stdin
                  docker push ${HARBOR_URL}/${IMAGE_NAME}:${IMAGE_TAG}
                  docker push ${HARBOR_URL}/${IMAGE_NAME}:latest
                '''
              }
            }
            echo "Pushed image to Harbor Cluster2: ${env.HARBOR_URL_CLUSTER2}/${env.IMAGE_NAME}:${env.IMAGE_TAG}"
          }
        }
      }
    }

    stage('Render and Push to ArgoCD') {
      steps {
        script {
          def argocdWorkdir = env.ARGOCD_WORKDIR
          def imageTag = env.IMAGE_TAG

          sh "rm -rf ${argocdWorkdir}"
          sh "mkdir -p ${argocdWorkdir}"

          dir(argocdWorkdir) {
            withCredentials([sshUserPrivateKey(credentialsId: 'Jenkins_Pipeline_Agent_SSH_Key', keyFileVariable: 'SSH_KEY')]) {
              sh 'GIT_SSH_COMMAND="ssh -i $SSH_KEY" git clone $ARGOCD_GIT_REMOTE .'
            }

            sh "cp -r ${WORKDIR}/k8s/kustomize ."

            dir('kustomize') {
              withEnv([
                "IMAGE_TAG=${imageTag}",
                "HARBOR_URL_CLUSTER1=${env.HARBOR_URL_CLUSTER1}",
                "HARBOR_URL_CLUSTER2=${env.HARBOR_URL_CLUSTER2}"
              ]) {
                sh '''
                  kustomize edit set image gaaming/blog=${HARBOR_URL_CLUSTER1}/gaaming/blog:${IMAGE_TAG} --apps/blog/overlays/cluster1
                  kustomize edit set image gaaming/blog=${HARBOR_URL_CLUSTER2}/gaaming/blog:${IMAGE_TAG} --apps/blog/overlays/cluster2
                '''
              }
            }

            sh '''
              rm -rf apps/blog
              kustomize build kustomize/blog/overlays/cluster1 -o apps/blog/cluster1/
              kustomize build kustomize/blog/overlays/cluster2 -o apps/blog/cluster2/
            '''

            sh "rm -rf kustomize argocd"

            withCredentials([sshUserPrivateKey(credentialsId: 'Jenkins_Pipeline_Agent_SSH_Key', keyFileVariable: 'SSH_KEY')]) {
              sh '''
                git config user.name "Jenkins"
                git config user.email "jenkins@gaamingzhang.com"
                git add apps/blog
                git commit -m "feat: 更新 blog 镜像版本到 ${IMAGE_TAG}"
                GIT_SSH_COMMAND="ssh -i $SSH_KEY" git push origin main
              '''
            }
          }

          echo "Rendered and pushed Kubernetes manifests to ArgoCD repository"
        }
      }
    }
  }

  post {
    success {
      echo "Pipeline completed successfully!"
      echo "Image version: ${env.IMAGE_TAG}"
    }
    failure {
      echo "Pipeline failed!"
    }
  }
}
