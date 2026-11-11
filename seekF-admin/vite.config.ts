import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "VITE_");
  return {
    // 管理页面通过 Nginx 的 /admin/ 入口访问，构建资源使用相同前缀。
    base: "/admin/",
    plugins: [react(), tailwindcss()],
    server: {
      host: "0.0.0.0",
      port: 3002,
      strictPort: true,
      proxy: {
        "/api": {
          target: env.VITE_BACKEND_TARGET || "http://localhost:8080",
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/api/, ""),
        },
      },
    },
  };
});
