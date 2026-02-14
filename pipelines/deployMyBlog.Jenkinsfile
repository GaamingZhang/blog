pipeline {
  agent any

  tools {
    nodejs 'NodeJS'
  }

  parameters {
    booleanParam(name: 'SKIP_ACCESS_LOG_PROCESSOR', defaultValue: false, description: '是否跳过博客访问日志处理服务的部署')
    booleanParam(name: 'DEPLOY_TO_ALI_BEIJING_NODE', defaultValue: false, description: '是否部署到Ali BeiJing Node')
    booleanParam(name: 'DEPLOY_TO_TENCENT_GUANGZHOU_NODE', defaultValue: false, description: '是否部署到Tencent Guangzhou Node')
    booleanParam(name: 'DEPLOY_TO_KUBERNETES', defaultValue: false, description: '是否部署到Kubernetes集群')
    booleanParam(name: 'SYNC_SSL_CERTS', defaultValue: false, description: '是否同步SSL证书文件')
    choice(name: 'SSL_CERTS_SOURCE_SERVER', choices: ['AliBeijing', 'TencentGuangzhou'], description: 'SSL证书源服务器')
    choice(name: 'SSL_CERTS_TARGET_SERVER', choices: ['TencentGuangzhou', 'AliBeijing'], description: 'SSL证书目标服务器')
  }

  environment {
    BLOG_DEPLOY_PATH = '/var/www/vuepress-blog'
    NGINX_CONF_REMOTE = '/etc/nginx/conf.d/myBlog.conf'
    ALI_BEIJING_NODE_IP = 'AliBeijingNodeIP'
    ALI_BEIJING_NODE_DEPLOY_USER = 'AliBeijingNodeDeployUser'
    ALI_BEIJING_NODE_SSH_KEY_CREDENTIAL = 'AliBeijingNodeSSH'
    TENCENT_GUANGZHOU_NODE_IP = 'TencentGuangzhouNodeIP'
    TENCENT_GUANGZHOU_NODE_DEPLOY_USER = 'TencentGuangzhouNodeDeployUser'
    TENCENT_GUANGZHOU_NODE_SSH_KEY_CREDENTIAL = 'TencentGuangzhouNodeSSH'
    VERSION = "${BUILD_NUMBER}"
    MAX_BACKUPS = 10
    LOG_PROCESS_SCRIPTS = "${BLOG_DEPLOY_PATH}/scripts"
    SSL_CERTS_SOURCE_DIR = '/etc/letsencrypt/live/gaaming.com.cn'
    SSL_CERTS_TARGET_DIR = '/etc/letsencrypt/live/gaaming.com.cn'
    SSL_DHPARAM_SOURCE = '/etc/letsencrypt/ssl-dhparams.pem'
    SSL_DHPARAM_TARGET = '/etc/letsencrypt/ssl-dhparams.pem'
    SSL_OPTIONS_SOURCE = '/etc/letsencrypt/options-ssl-nginx.conf'
    SSL_OPTIONS_TARGET = '/etc/letsencrypt/options-ssl-nginx.conf'
    REGISTRY_NODE_PORT = '30500'
    K8S_NODE_IP = '192.168.31.40'
    IMAGE_NAME = 'gaamingzhang-blog'
    K8S_NAMESPACE = 'default'
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

    stage('Deploy Static Site') {
      steps {
        script {
          def deployJobs = [:]
          
          // 只有当DEPLOY_TO_ALI_BEIJING_NODE为true时才添加Ali BeiJing Node的部署任务
          if (params.DEPLOY_TO_ALI_BEIJING_NODE) {
            deployJobs["Deploy to Ali BeiJing Node"] = {
              withCredentials([
                string(credentialsId: ALI_BEIJING_NODE_IP, variable: 'DEPLOY_HOST'),
                string(credentialsId: ALI_BEIJING_NODE_DEPLOY_USER, variable: 'DEPLOY_USER'),
                sshUserPrivateKey(credentialsId: ALI_BEIJING_NODE_SSH_KEY_CREDENTIAL, keyFileVariable: 'SSH_KEY')
              ]) {
                deployToRemote()
              }
            }
          }
          
          // 只有当DEPLOY_TO_TENCENT_GUANGZHOU_NODE为true时才添加Tencent Guangzhou Node的部署任务
          if (params.DEPLOY_TO_TENCENT_GUANGZHOU_NODE) {
            deployJobs["Deploy to Tencent Guangzhou Node"] = {
              withCredentials([
                string(credentialsId: TENCENT_GUANGZHOU_NODE_IP, variable: 'DEPLOY_HOST'),
                string(credentialsId: TENCENT_GUANGZHOU_NODE_DEPLOY_USER, variable: 'DEPLOY_USER'),
                sshUserPrivateKey(credentialsId: TENCENT_GUANGZHOU_NODE_SSH_KEY_CREDENTIAL, keyFileVariable: 'SSH_KEY')
              ]) {
                deployToRemote()
              }
            }
          }
          
          // 执行部署任务
          if (!deployJobs.isEmpty()) {
            parallel(deployJobs)
          } else {
            echo "没有启用任何部署任务"
          }
        }
      }
    }

    stage('Process Blog Access Log') {
      when {
        expression { params.SKIP_ACCESS_LOG_PROCESSOR != true }
      }
      steps {
        script {
          def processJobs = [:]
          
          // 只有当DEPLOY_TO_ALI_BEIJING_NODE为true时才添加Ali BeiJing Node的处理任务
          if (params.DEPLOY_TO_ALI_BEIJING_NODE) {
            processJobs["Process Blog Access Log on Ali BeiJing Node"] = {
              withCredentials([
                string(credentialsId: ALI_BEIJING_NODE_IP, variable: 'DEPLOY_HOST'),
                string(credentialsId: ALI_BEIJING_NODE_DEPLOY_USER, variable: 'DEPLOY_USER'),
                sshUserPrivateKey(credentialsId: ALI_BEIJING_NODE_SSH_KEY_CREDENTIAL, keyFileVariable: 'SSH_KEY')
              ]) {
                processBlogAccessLog()
              }
            }
          }
          
          // 只有当DEPLOY_TO_TENCENT_GUANGZHOU_NODE为true时才添加Tencent Guangzhou Node的处理任务
          if (params.DEPLOY_TO_TENCENT_GUANGZHOU_NODE) {
            processJobs["Process Blog Access Log on Tencent Guangzhou Node"] = {
              withCredentials([
                string(credentialsId: TENCENT_GUANGZHOU_NODE_IP, variable: 'DEPLOY_HOST'),
                string(credentialsId: TENCENT_GUANGZHOU_NODE_DEPLOY_USER, variable: 'DEPLOY_USER'),
                sshUserPrivateKey(credentialsId: TENCENT_GUANGZHOU_NODE_SSH_KEY_CREDENTIAL, keyFileVariable: 'SSH_KEY')
              ]) {
                processBlogAccessLog()
              }
            }
          }
          
          // 执行处理任务
          if (!processJobs.isEmpty()) {
            parallel(processJobs)
          } else {
            echo "没有启用任何博客访问日志处理任务"
          }
        }
      }
    }

    stage('Sync SSL Certs') {
      when {
        expression { params.SYNC_SSL_CERTS == true }
      }
      steps {
        script {
          def sourceServer = params.SSL_CERTS_SOURCE_SERVER
          def targetServer = params.SSL_CERTS_TARGET_SERVER
          
          if (sourceServer == targetServer) {
            error "源服务器和目标服务器不能相同"
          }
          
          def sourceIpCredential = sourceServer == 'AliBeijing' ? ALI_BEIJING_NODE_IP : TENCENT_GUANGZHOU_NODE_IP
          def sourceUserCredential = sourceServer == 'AliBeijing' ? ALI_BEIJING_NODE_DEPLOY_USER : TENCENT_GUANGZHOU_NODE_DEPLOY_USER
          def sourceSshCredential = sourceServer == 'AliBeijing' ? ALI_BEIJING_NODE_SSH_KEY_CREDENTIAL : TENCENT_GUANGZHOU_NODE_SSH_KEY_CREDENTIAL
          
          def targetIpCredential = targetServer == 'AliBeijing' ? ALI_BEIJING_NODE_IP : TENCENT_GUANGZHOU_NODE_IP
          def targetUserCredential = targetServer == 'AliBeijing' ? ALI_BEIJING_NODE_DEPLOY_USER : TENCENT_GUANGZHOU_NODE_DEPLOY_USER
          def targetSshCredential = targetServer == 'AliBeijing' ? ALI_BEIJING_NODE_SSH_KEY_CREDENTIAL : TENCENT_GUANGZHOU_NODE_SSH_KEY_CREDENTIAL
          
          withCredentials([
            string(credentialsId: sourceIpCredential, variable: 'SOURCE_HOST'),
            string(credentialsId: sourceUserCredential, variable: 'SOURCE_USER'),
            sshUserPrivateKey(credentialsId: sourceSshCredential, keyFileVariable: 'SOURCE_SSH_KEY'),
            string(credentialsId: targetIpCredential, variable: 'TARGET_HOST'),
            string(credentialsId: targetUserCredential, variable: 'TARGET_USER'),
            sshUserPrivateKey(credentialsId: targetSshCredential, keyFileVariable: 'TARGET_SSH_KEY')
          ]) {
            syncSSLCerts()
          }
        }
      }
    }

    // TODO: 创建新的流水线部署 Nginx
    // TODO: 增加备份旧版本的 stage
    // TODO: 增加回滚 stage
    stage('Deploy Nginx Config') {
      steps {
        script {
          def nginxJobs = [:]
          
          // 只有当DEPLOY_TO_ALI_BEIJING_NODE为true时才添加Ali BeiJing Node的Nginx配置部署任务
          if (params.DEPLOY_TO_ALI_BEIJING_NODE) {
            nginxJobs["Deploy Nginx Config to Ali BeiJing Node"] = {
              withCredentials([
                string(credentialsId: ALI_BEIJING_NODE_IP, variable: 'DEPLOY_HOST'),
                string(credentialsId: ALI_BEIJING_NODE_DEPLOY_USER, variable: 'DEPLOY_USER'),
                sshUserPrivateKey(credentialsId: ALI_BEIJING_NODE_SSH_KEY_CREDENTIAL, keyFileVariable: 'SSH_KEY')
              ]) {
                deployNginxConfig()
              }
            }
          }
          
          // 只有当DEPLOY_TO_TENCENT_GUANGZHOU_NODE为true时才添加Tencent Guangzhou Node的Nginx配置部署任务
          if (params.DEPLOY_TO_TENCENT_GUANGZHOU_NODE) {
            nginxJobs["Deploy Nginx Config to Tencent Guangzhou Node"] = {
              withCredentials([
                string(credentialsId: TENCENT_GUANGZHOU_NODE_IP, variable: 'DEPLOY_HOST'),
                string(credentialsId: TENCENT_GUANGZHOU_NODE_DEPLOY_USER, variable: 'DEPLOY_USER'),
                sshUserPrivateKey(credentialsId: TENCENT_GUANGZHOU_NODE_SSH_KEY_CREDENTIAL, keyFileVariable: 'SSH_KEY')
              ]) {
                deployNginxConfig()
              }
            }
          }
          
          // 执行Nginx配置部署任务
          if (!nginxJobs.isEmpty()) {
            parallel(nginxJobs)
          } else {
            echo "没有启用任何Nginx配置部署任务"
          }
        }
      }
    }

    stage('Deploy to Kubernetes') {
      when {
        expression { params.DEPLOY_TO_KUBERNETES == true }
      }
      steps {
        script {
          echo "部署到 Kubernetes 集群..."
          
          withCredentials([kubeconfigFile(credentialsId: 'kubernetes-kubeconfig', variable: 'KUBECONFIG')]) {
            sh '''
              set -e
              
              REGISTRY_URL="${K8S_NODE_IP}:${REGISTRY_NODE_PORT}"
              
              echo "更新 deployment 镜像版本..."
              sed -i.bak "s|image: .*gaamingzhang-blog:.*|image: ${REGISTRY_URL}/${IMAGE_NAME}:${VERSION}|g" k8s/deployment.yaml
              
              echo "应用 Kubernetes 配置..."
              kubectl apply -f k8s/deployment.yaml
              kubectl apply -f k8s/service.yaml
              kubectl apply -f k8s/ingress.yaml
              
              echo "等待部署完成..."
              kubectl rollout status deployment/gaamingzhang-blog -n ${K8S_NAMESPACE} --timeout=300s
              
              echo "验证部署状态..."
              kubectl get pods -n ${K8S_NAMESPACE} -l app=gaamingzhang-blog
              kubectl get services -n ${K8S_NAMESPACE}
              kubectl get ingress -n ${K8S_NAMESPACE}
            '''
          }
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
          /var/wxpush/wxpush -appID ${WXPUSH_APPID} -secret ${WXPUSH_SECRET} -userID ${WXPUSH_USERID} -templateID ${WXPUSH_TEMPLATEID} -title "博客部署成功" -content "gaamingzhangblog v.'${BUILD_NUMBER}' 部署成功"
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
          /var/wxpush/wxpush -appID ${WXPUSH_APPID} -secret ${WXPUSH_SECRET} -userID ${WXPUSH_USERID} -templateID ${WXPUSH_TEMPLATEID} -title "博客部署失败" -content "gaamingzhangblog v.'${BUILD_NUMBER}' 部署失败"
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

def syncSSLCerts() {
    sh """
        set -e
        SOURCE_REMOTE="\${SOURCE_USER}@\${SOURCE_HOST}"
        TARGET_REMOTE="\${TARGET_USER}@\${TARGET_HOST}"
        
        echo "同步SSL证书从 \$SOURCE_REMOTE 到 \$TARGET_REMOTE"
        
        # 在目标服务器创建目录
        echo "在目标服务器创建目录..."
        ssh -i "\${TARGET_SSH_KEY}" -o StrictHostKeyChecking=no "\$TARGET_REMOTE" "sudo mkdir -p ${SSL_CERTS_TARGET_DIR}"
        
        # 直接从源服务器复制文件到目标服务器（通过SSH管道，不经过Jenkins节点）
        echo "同步 fullchain.pem..."
        ssh -i "\${SOURCE_SSH_KEY}" -o StrictHostKeyChecking=no "\$SOURCE_REMOTE" "sudo cat ${SSL_CERTS_SOURCE_DIR}/fullchain.pem" | ssh -i "\${TARGET_SSH_KEY}" -o StrictHostKeyChecking=no "\$TARGET_REMOTE" "sudo tee ${SSL_CERTS_TARGET_DIR}/fullchain.pem > /dev/null"
        
        echo "同步 privkey.pem..."
        ssh -i "\${SOURCE_SSH_KEY}" -o StrictHostKeyChecking=no "\$SOURCE_REMOTE" "sudo cat ${SSL_CERTS_SOURCE_DIR}/privkey.pem" | ssh -i "\${TARGET_SSH_KEY}" -o StrictHostKeyChecking=no "\$TARGET_REMOTE" "sudo tee ${SSL_CERTS_TARGET_DIR}/privkey.pem > /dev/null"
        
        echo "同步 options-ssl-nginx.conf..."
        ssh -i "\${SOURCE_SSH_KEY}" -o StrictHostKeyChecking=no "\$SOURCE_REMOTE" "sudo cat ${SSL_OPTIONS_SOURCE}" | ssh -i "\${TARGET_SSH_KEY}" -o StrictHostKeyChecking=no "\$TARGET_REMOTE" "sudo tee ${SSL_OPTIONS_TARGET} > /dev/null"
        
        echo "同步 ssl-dhparams.pem..."
        ssh -i "\${SOURCE_SSH_KEY}" -o StrictHostKeyChecking=no "\$SOURCE_REMOTE" "sudo cat ${SSL_DHPARAM_SOURCE}" | ssh -i "\${TARGET_SSH_KEY}" -o StrictHostKeyChecking=no "\$TARGET_REMOTE" "sudo tee ${SSL_DHPARAM_TARGET} > /dev/null"
        
        echo "SSL证书同步完成"
    """
}

def processBlogAccessLog() {
    sh """
        set -e
        REMOTE="\${DEPLOY_USER}@\${DEPLOY_HOST}"
        echo "处理博客访问日志服务: \$REMOTE"
        
        # 设置脚本权限
        ssh -i "\${SSH_KEY}" -o StrictHostKeyChecking=no "\$REMOTE" "sudo chmod +x $LOG_PROCESS_SCRIPTS/process_blog_access.sh"
        
        # 复制服务文件
        ssh -i "\${SSH_KEY}" -o StrictHostKeyChecking=no "\$REMOTE" "sudo cp $LOG_PROCESS_SCRIPTS/process_blog_access.service /etc/systemd/system/"
        ssh -i "\${SSH_KEY}" -o StrictHostKeyChecking=no "\$REMOTE" "sudo cp $LOG_PROCESS_SCRIPTS/process_blog_access.timer /etc/systemd/system/"
        
        # 重新加载systemd配置
        ssh -i "\${SSH_KEY}" -o StrictHostKeyChecking=no "\$REMOTE" "sudo systemctl daemon-reload"
        
        # 禁用并停止定时器
        ssh -i "\${SSH_KEY}" -o StrictHostKeyChecking=no "\$REMOTE" "sudo systemctl disable process_blog_access.timer"
        ssh -i "\${SSH_KEY}" -o StrictHostKeyChecking=no "\$REMOTE" "sudo systemctl stop process_blog_access.timer"
        
        echo "博客访问日志服务处理完成"
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