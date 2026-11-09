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
  },
  preview: {
    host: true,
    port: 7174,
    strictPort: true,
  },
});
