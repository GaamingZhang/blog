pipeline {
  agent any

  tools {
    nodejs 'NodeJS'
  }

  parameters {
    booleanParam(name: 'SKIP_ACCESS_LOG_PROCESSOR', defaultValue: false, description: '是否跳过博客访问日志处理服务的部署')
  }

  environment {
    BLOG_DEPLOY_PATH = '/var/www/vuepress-blog'
    NGINX_CONF_REMOTE = '/etc/nginx/conf.d/myBlog.conf'
    TENCENT_NODE_IP = 'TencentNodeIP'
    TENCENT_NODE_DEPLOY_USER = 'TencentNodeDeployUser'
    TENCENT_NODE_SSH_KEY_CREDENTIAL = 'TencentNodeSSHKey'
    TENCENT_GUANGZHOU_NODE_IP = 'TencentGuangzhouNodeIP'
    TENCENT_GUANGZHOU_NODE_DEPLOY_USER = 'TencentGuangzhouNodeDeployUser'
    TENCENT_GUANGZHOU_NODE_SSH_KEY_CREDENTIAL = 'TencentGuangzhouNodeSSH'
    VERSION = "${BUILD_NUMBER}"
    MAX_BACKUPS = 10
    LOG_PROCESS_SCRIPTS = "${BLOG_DEPLOY_PATH}/scripts"
  }

  // TODO: 部署前生成 official_blog_<buildNumber> 分支
  // TODO： 根据 指定分支 拉取代码
  // TODO: 增加 stage 回滚到上一个版本
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
        script {
          parallel(
            "Deploy to Tencent Node": {
            withCredentials([
              string(credentialsId: TENCENT_NODE_IP, variable: 'DEPLOY_HOST'),
              string(credentialsId: TENCENT_NODE_DEPLOY_USER, variable: 'DEPLOY_USER'),
              sshUserPrivateKey(credentialsId: TENCENT_NODE_SSH_KEY_CREDENTIAL, keyFileVariable: 'SSH_KEY')
            ]) {
              deployToRemote()
            }
          },
          "Deploy to Tencent Guangzhou Node": {
            withCredentials([
              string(credentialsId: TENCENT_GUANGZHOU_NODE_IP, variable: 'DEPLOY_HOST'),
              string(credentialsId: TENCENT_GUANGZHOU_NODE_DEPLOY_USER, variable: 'DEPLOY_USER'),
              sshUserPrivateKey(credentialsId: TENCENT_GUANGZHOU_NODE_SSH_KEY_CREDENTIAL, keyFileVariable: 'SSH_KEY')
            ]) {
              deployToRemote()
            }
          }
          )
        }
      }
    }

    stage('Process Blog Access Log') {
      when {
        expression { params.SKIP_ACCESS_LOG_PROCESSOR != true }
      }
      steps {
        withCredentials([
          string(credentialsId: TENCENT_NODE_IP, variable: 'DEPLOY_HOST'),
          sshUserPrivateKey(credentialsId: TENCENT_NODE_SSH_KEY_CREDENTIAL, keyFileVariable: 'SSH_KEY')
        ]) {
          sh '''
            set -e
            REMOTE="$TENCENT_NODE_DEPLOY_USER@$DEPLOY_HOST"
            ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "$REMOTE" "sudo chmod +x $LOG_PROCESS_SCRIPTS/process_blog_access.sh"
            ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "$REMOTE" "sudo cp $LOG_PROCESS_SCRIPTS/process_blog_access.service /etc/systemd/system/"
            ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "$REMOTE" "sudo cp $LOG_PROCESS_SCRIPTS/process_blog_access.timer /etc/systemd/system/"
            ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "$REMOTE" "sudo systemctl daemon-reload"
            ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "$REMOTE" "sudo systemctl disable process_blog_access.timer"
            ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "$REMOTE" "sudo systemctl stop process_blog_access.timer"
          '''
        }
      }
    }

    // TODO: 创建新的流水线部署 Nginx
    // TODO: 增加备份旧版本的 stage
    // TODO: 增加回滚 stage
    stage('Deploy Nginx Config') {
      steps {
        script {
          parallel(
            "Deploy Nginx Config to Tencent Node": {
            withCredentials([
              string(credentialsId: TENCENT_NODE_IP, variable: 'DEPLOY_HOST'),
              string(credentialsId: TENCENT_NODE_DEPLOY_USER, variable: 'DEPLOY_USER'),
              sshUserPrivateKey(credentialsId: TENCENT_NODE_SSH_KEY_CREDENTIAL, keyFileVariable: 'SSH_KEY')
            ]) {
              deployNginxConfig()
            }
          },
          "Deploy Nginx Config to Tencent Guangzhou Node": {
            withCredentials([
              string(credentialsId: TENCENT_GUANGZHOU_NODE_IP, variable: 'DEPLOY_HOST'),
              string(credentialsId: TENCENT_GUANGZHOU_NODE_DEPLOY_USER, variable: 'DEPLOY_USER'),
              sshUserPrivateKey(credentialsId: TENCENT_GUANGZHOU_NODE_SSH_KEY_CREDENTIAL, keyFileVariable: 'SSH_KEY')
            ]) {
              deployNginxConfig()
            }
          }
          )
        }
      }
    }
  }

  post {
    success {
      withCredentials([
        string(credentialsId: 'wxpush_appID', variable: 'WXPUSH_APPID'),
        string(credentialsId: 'wxpush_secret', variable: 'WXPUSH_SECRET'),
        string(credentialsId: 'wxpush_userID', variable: 'WXPUSH_USERID'),
        string(credentialsId: 'wxpush_templateID', variable: 'WXPUSH_TEMPLATEID')
      ]) {
        sh '''
          /var/wxpush/wxpush -appID ${WXPUSH_APPID} -secret ${WXPUSH_SECRET} -userID ${WXPUSH_USERID} -templateID ${WXPUSH_TEMPLATEID} -title "博客部署成功" -content "博客项目 gaamingzhangblog v.'${BUILD_NUMBER}' 已成功部署到生产环境"
        '''
      }
    }
    failure {
      withCredentials([
        string(credentialsId: 'wxpush_appID', variable: 'WXPUSH_APPID'),
        string(credentialsId: 'wxpush_secret', variable: 'WXPUSH_SECRET'),
        string(credentialsId: 'wxpush_userID', variable: 'WXPUSH_USERID'),
        string(credentialsId: 'wxpush_templateID', variable: 'WXPUSH_TEMPLATEID')
      ]) {
        sh '''
          /var/wxpush/wxpush -appID ${WXPUSH_APPID} -secret ${WXPUSH_SECRET} -userID ${WXPUSH_USERID} -templateID ${WXPUSH_TEMPLATEID} -title "博客部署失败" -content "博客项目 gaamingzhangblog v.'${BUILD_NUMBER}' 部署失败"
        '''
      }
    }
  }
}

def deployToRemote() {
    sh """
        set -e
        REMOTE="\${DEPLOY_USER}@\${DEPLOY_HOST}"
        echo "连接远程服务器: \$REMOTE"
        echo "部署版本: ${VERSION}"
        
        # 在远程服务器删除旧目录并创建新目录
        ssh -i "\${SSH_KEY}" -o StrictHostKeyChecking=no "\$REMOTE" "sudo rm -rf ${BLOG_DEPLOY_PATH}_new && sudo mkdir -p ${BLOG_DEPLOY_PATH}_new"
        
        # 同步二进制文件到远程服务器
        echo "同步二进制文件..."
        rsync -avz --delete --rsync-path="sudo rsync" -e "ssh -i \${SSH_KEY} -o StrictHostKeyChecking=no" src/.vuepress/dist/ "\$REMOTE:${BLOG_DEPLOY_PATH}_new/"
        
        # 在远程服务器执行部署脚本
        ssh -i "\${SSH_KEY}" -o StrictHostKeyChecking=no "\$REMOTE" bash <<'ENDSSH'
${deploy()}
ENDSSH
    """
}

def deployNginxConfig() {
    sh """
        set -e
        REMOTE="\${DEPLOY_USER}@\${DEPLOY_HOST}"
        echo "部署Nginx配置到: \$REMOTE"
        
        # 上传Nginx配置文件
        scp -i "\${SSH_KEY}" -o StrictHostKeyChecking=no pipelines/nginx/myBlog.conf "\$REMOTE:/tmp/myBlog.conf"
        
        # 替换Nginx配置文件
        ssh -i "\${SSH_KEY}" -o StrictHostKeyChecking=no "\$REMOTE" "sudo mv /tmp/myBlog.conf $NGINX_CONF_REMOTE"
        
        # 测试Nginx配置
        ssh -i "\${SSH_KEY}" -o StrictHostKeyChecking=no "\$REMOTE" "sudo nginx -t"
        
        # 重新加载Nginx
        ssh -i "\${SSH_KEY}" -o StrictHostKeyChecking=no "\$REMOTE" "sudo systemctl reload nginx"
        
        echo "Nginx配置已部署并重新加载"
    """
}

def deploy() {
    """
        # 定义备份目录名称（带版本号后缀）
        BACKUP_DIR="${BLOG_DEPLOY_PATH}_backup_v${VERSION}"
        
        # 备份旧版本
        ${backupOldVersion()}
        
        # 使用 rsync 同步新版本到部署路径（自动处理新增、修改和删除的文件）
        sudo rsync -avz --delete ${BLOG_DEPLOY_PATH}_new/ ${BLOG_DEPLOY_PATH}/
        
        # 删除临时目录
        sudo rm -rf ${BLOG_DEPLOY_PATH}_new
        
        # 清理旧备份
        ${cleanupOldBackups()}
        
        # 验证部署
        echo ""
        echo "已部署文件："
        ls -lh ${BLOG_DEPLOY_PATH}/
    """
}

def backupOldVersion() {
    def backupScript = """
        if [ -d '${BLOG_DEPLOY_PATH}' ]; then
            sudo cp -r ${BLOG_DEPLOY_PATH} '${BLOG_DEPLOY_PATH}_backup_v${VERSION}'
            echo '已备份旧版本'
        else
            echo '首次部署，无需备份'
        fi
    """
    
    return """
        echo "备份旧版本到 ${BLOG_DEPLOY_PATH}_backup_v${VERSION}..."
        ${backupScript}
    """
}

def cleanupOldBackups() {
    def cleanupScript = """
        BACKUP_COUNT=\$(ls -d ${BLOG_DEPLOY_PATH}_backup_v* 2>/dev/null | wc -l | tr -d ' ')
        echo '当前备份数量: '\$BACKUP_COUNT
        
        if [ "\${BACKUP_COUNT:-0}" -gt ${MAX_BACKUPS} ]; then
            DELETE_COUNT=\$((BACKUP_COUNT - MAX_BACKUPS))
            echo '需要删除 '\$DELETE_COUNT' 个旧备份'
            for dir in \$(ls -dt ${BLOG_DEPLOY_PATH}_backup_v* | tail -n \$DELETE_COUNT); do
                echo "删除旧备份: \$dir"
                sudo rm -rf "\$dir" && echo "已删除: \$dir" || echo "删除失败: \$dir"
            done
        else
            echo '备份数量在限制范围内，无需清理'
        fi
        
        echo ''
        echo '当前备份列表：'
        ls -lhdt ${BLOG_DEPLOY_PATH}_backup_v* 2>/dev/null || echo '暂无备份'
    """
    
    return """
        echo "清理旧备份文件..."
        ${cleanupScript}
    """
}