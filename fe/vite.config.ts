import { defineConfig } from "vite";

// Порт 7173 — см. README, «Доступ из локальной сети».
export default defineConfig({
  server: {
    host: true, // слушать все интерфейсы — доступ из локальной сети
    port: 7173,
    strictPort: true, // не перескакивать молча на другой (возможно, закрытый) порт
    watch: {
      // В Docker на macOS события ФС из bind-mount доходят не всегда — включаем опрос.
      usePolling: process.env.VITE_USE_POLLING === "true",
      interval: 300,
    },
    // Браузер ходит только на порт фронта; API и картинки проксируются.
    // Так коллегам в локальной сети достаточно одного открытого порта.
    proxy: {
      "/api": {
        target: process.env.API_URL ?? "http://localhost:8080",
        changeOrigin: true,
      },
      "/s3": {
        target: process.env.S3_URL ?? "http://localhost:9000",
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/s3/, ""),
      },
    },
  },
  preview: {
    host: true,
    port: 7174,
    strictPort: true,
  },
});
