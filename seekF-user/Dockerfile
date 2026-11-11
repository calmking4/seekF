# 第一阶段：安装依赖，构建 Nuxt 用户前端。
FROM node:22-alpine AS builder

WORKDIR /app
COPY . .
RUN npm ci
RUN npm run build

# 第二阶段：只保留构建产物，使用 Node 运行。
FROM node:22-alpine

WORKDIR /app
COPY --from=builder /app/.output ./.output

ENV NODE_ENV=production \
    NITRO_HOST=0.0.0.0 \
    NITRO_PORT=3000

EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]
