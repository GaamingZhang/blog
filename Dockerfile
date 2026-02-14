# 基础镜像
FROM node:20-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制项目文件
COPY . .

# 安装pnpm
RUN npm install pnpm

# 设置CI环境变量，避免pnpm在非交互式环境中的提示
ENV CI=true

# 安装依赖
RUN pnpm install

# 构建生产版本
RUN pnpm run docs:build

# 第二阶段：使用Nginx作为生产服务器
FROM nginx:alpine

# 复制构建产物到Nginx目录
COPY --from=builder /app/src/.vuepress/dist /usr/share/nginx/html

# 复制Nginx配置文件
COPY pipelines/nginx/myBlog.local.conf /etc/nginx/conf.d/default.conf

# 暴露80端口
EXPOSE 80

# 启动Nginx
CMD ["nginx", "-g", "daemon off;"]
