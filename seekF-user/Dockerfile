# 第一阶段：安装依赖，构建 Nuxt 用户前端。
FROM node:22-alpine AS builder

WORKDIR /app
COPY package.json package-lock.json ./
# 依赖安装层只依赖清单，Nuxt 项目的准备脚本等源码复制后再执行。
RUN --mount=type=cache,id=seekf-user-npm,target=/root/.npm \
    npm pkg delete scripts.postinstall && npm ci

COPY . .
RUN npm run postinstall && npm run build

# 第二阶段：只保留构建产物，使用 Node 运行。
FROM node:22-alpine

WORKDIR /app
COPY --from=builder /app/.output ./.output

ENV NODE_ENV=production \
    NITRO_HOST=0.0.0.0 \
    NITRO_PORT=3000

EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]
