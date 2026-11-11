# 安装依赖并构建管理前端，供本地容器预览使用。
FROM node:22-alpine

WORKDIR /app
COPY . .
RUN npm ci
RUN npm run build

# Vite preview 使用现有代理规则，不需要 Nginx。
EXPOSE 3002
CMD ["npm", "run", "preview", "--", "--host", "0.0.0.0", "--strictPort"]
