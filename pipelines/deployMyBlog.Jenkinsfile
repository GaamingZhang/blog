pipeline {
  agent any

  environment {
    DEPLOY_USER = 'root'
    DEPLOY_PATH = '/var/www/vuepress-blog'
    NGINX_CONF_REMOTE = '/etc/nginx/conf.d/myBlog.conf'
    SSH_KEY_CREDENTIAL = 'TencentNodeSSHKey'
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
          export PATH="$(npm config get prefix)/bin:$PATH"
          npm install -g corepack
          corepack enable
          corepack prepare pnpm@latest --activate
          pnpm install --frozen-lockfile
          pnpm run docs:build
        '''
      }
    }

    stage('Deploy Static Site') {
      steps {
        withCredentials([string(credentialsId: 'TencentNodeIP', variable: 'DEPLOY_HOST')]) {
          sshagent([SSH_KEY_CREDENTIAL]) {
            sh '''
              set -e
              REMOTE="$DEPLOY_USER@$DEPLOY_HOST"
              ssh -o StrictHostKeyChecking=no "$REMOTE" "mkdir -p ${DEPLOY_PATH}_new"
              rsync -avz --delete src/.vuepress/dist/ "$REMOTE:${DEPLOY_PATH}_new/"
              ssh "$REMOTE" "rm -rf ${DEPLOY_PATH}_backup && mv ${DEPLOY_PATH} ${DEPLOY_PATH}_backup || true"
              ssh "$REMOTE" "mv ${DEPLOY_PATH}_new ${DEPLOY_PATH}"
            '''
          }
        }
      }
    }

    stage('Deploy Nginx Config') {
      steps {
        withCredentials([string(credentialsId: 'TencentNodeIP', variable: 'DEPLOY_HOST')]) {
          sshagent([SSH_KEY_CREDENTIAL]) {
            sh '''
              set -e
              REMOTE="$DEPLOY_USER@$DEPLOY_HOST"
              scp -o StrictHostKeyChecking=no src/pipelines/nginx/myBlog.conf "$REMOTE:$NGINX_CONF_REMOTE"
              ssh "$REMOTE" "nginx -t"
              ssh "$REMOTE" "systemctl reload nginx"
            '''
          }
        }
      }
    }
  }
}
