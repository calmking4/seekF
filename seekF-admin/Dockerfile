# 安装依赖并构建管理前端，供本地容器预览使用。
FROM node:22-alpine

WORKDIR /app
COPY . .
RUN npm ci
RUN npm run build

# 容器内通过 Vite preview 提供页面，外部 Nginx 统一转发入口。
EXPOSE 3002
CMD ["npm", "run", "preview", "--", "--host", "0.0.0.0", "--strictPort"]
