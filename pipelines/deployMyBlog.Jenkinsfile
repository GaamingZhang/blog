pipeline {
  agent any

  tools {
    nodejs 'NodeJS'
  }

  environment {
    DEPLOY_USER = 'ubuntu'
    DEPLOY_PATH = '/var/www/vuepress-blog'
    NGINX_CONF_REMOTE = '/etc/nginx/conf.d/myBlog.conf'
    SSH_KEY_CREDENTIAL = 'TencentNodeSSHKey'
    WXPUSH_APPID = credentials('wxpush_appID')
    WXPUSH_SECRET = credentials('wxpush_secret')
    WXPUSH_USERID = credentials('wxpush_userID')
    WXPUSH_TEMPLATEID = credentials('wxpush_templateID')
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

    stage('Deploy Static Site') {
      steps {
        withCredentials([
          string(credentialsId: 'TencentNodeIP', variable: 'DEPLOY_HOST'),
          sshUserPrivateKey(credentialsId: SSH_KEY_CREDENTIAL, keyFileVariable: 'SSH_KEY')
        ]) {
          sh '''
            set -e
            REMOTE="$DEPLOY_USER@$DEPLOY_HOST"
            ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "$REMOTE" "sudo mkdir -p ${DEPLOY_PATH}_new"
            rsync -avz --delete --rsync-path="sudo rsync" -e "ssh -i $SSH_KEY -o StrictHostKeyChecking=no" src/.vuepress/dist/ "$REMOTE:${DEPLOY_PATH}_new/"
            ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "$REMOTE" "sudo rm -rf ${DEPLOY_PATH}_backup && sudo mv ${DEPLOY_PATH} ${DEPLOY_PATH}_backup || true"
            ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "$REMOTE" "sudo mv ${DEPLOY_PATH}_new ${DEPLOY_PATH}"
          '''
        }
      }
    }

    stage('Deploy Nginx Config') {
      steps {
        withCredentials([
          string(credentialsId: 'TencentNodeIP', variable: 'DEPLOY_HOST'),
          sshUserPrivateKey(credentialsId: SSH_KEY_CREDENTIAL, keyFileVariable: 'SSH_KEY')
        ]) {
          sh '''
            set -e
            REMOTE="$DEPLOY_USER@$DEPLOY_HOST"
            scp -i "$SSH_KEY" -o StrictHostKeyChecking=no pipelines/nginx/myBlog.conf "$REMOTE:/tmp/myBlog.conf"
            ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "$REMOTE" "sudo mv /tmp/myBlog.conf $NGINX_CONF_REMOTE"
            ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "$REMOTE" "sudo nginx -t"
            ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "$REMOTE" "sudo systemctl reload nginx"
          '''
        }
      }
    }
  }

  post {
    success {
      sh """
        /var/wxpush/wxpush -appID ${WXPUSH_APPID} -secret ${WXPUSH_SECRET} -userID ${WXPUSH_USERID} -templateID ${WXPUSH_TEMPLATEID} -title "博客部署成功" -content "博客项目 gaamingzhangblog v.${BUILD_NUMBER} 已成功部署到生产环境"
      """
    }
    failure {
      sh """
        /var/wxpush/wxpush -appID ${WXPUSH_APPID} -secret ${WXPUSH_SECRET} -userID ${WXPUSH_USERID} -templateID ${WXPUSH_TEMPLATEID} -title "博客部署失败" -content "博客项目 gaamingzhangblog v.${BUILD_NUMBER} 部署失败"
      """
    }
  }
}
