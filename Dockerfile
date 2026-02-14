FROM node:20-alpine AS builder

WORKDIR /app

COPY package.json pnpm-lock.yaml ./

RUN corepack enable && corepack prepare pnpm@latest --activate

RUN pnpm install --frozen-lockfile

COPY . .

ENV CI=true

RUN pnpm run docs:build

FROM nginx:alpine

COPY --from=builder /app/src/.vuepress/dist /usr/share/nginx/html

COPY pipelines/nginx/myBlog.local.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
