pipeline {
  agent any

  tools {
    nodejs 'NodeJS'
  }

  parameters {
    booleanParam(name: 'DEPLOY_TO_KUBERNETES', defaultValue: false, description: '是否部署到Kubernetes集群')
  }

  environment {
    VERSION = "${BUILD_NUMBER}"
    REGISTRY_NODE_PORT = '30500'
    K8S_NODE_IP = '192.168.31.40'
    IMAGE_NAME = 'gaamingzhang-blog'
    K8S_NAMESPACE = 'default'
  }

  stages {
    stage('Checkout') {
      steps {
        checkout scm
      }
    }
    
    stage('Install & Build') {
      steps {
        sh '''
          set -e
          corepack enable
          corepack prepare pnpm@latest --activate
          pnpm install --frozen-lockfile
          pnpm run docs:build
        '''
      }
    }

    stage('Build & Push Docker Image') {
      when {
        expression { params.DEPLOY_TO_KUBERNETES == true }
      }
      steps {
        script {
          withCredentials([
            usernamePassword(credentialsId: 'registry-credentials', usernameVariable: 'REGISTRY_USER', passwordVariable: 'REGISTRY_PASSWORD')
          ]) {
            sh '''
              set -e
              
              REGISTRY_URL="${K8S_NODE_IP}:${REGISTRY_NODE_PORT}"
              
              echo "构建 Docker 镜像..."
              docker build -t ${REGISTRY_URL}/${IMAGE_NAME}:${VERSION} -t ${REGISTRY_URL}/${IMAGE_NAME}:latest .
              
              echo "登录 Registry (${REGISTRY_URL})..."
              echo "${REGISTRY_PASSWORD}" | docker login ${REGISTRY_URL} -u "${REGISTRY_USER}" --password-stdin
              
              echo "推送镜像到 Registry..."
              docker push ${REGISTRY_URL}/${IMAGE_NAME}:${VERSION}
              docker push ${REGISTRY_URL}/${IMAGE_NAME}:latest
              
              echo "镜像推送完成: ${REGISTRY_URL}/${IMAGE_NAME}:${VERSION}"
            '''
          }
        }
      }
    }
  }
}