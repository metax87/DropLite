import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// 当前保持精简的 Vite 配置，后续可随模块增加再调整别名及插件。
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    open: true,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true
      }
    }
  }
});
