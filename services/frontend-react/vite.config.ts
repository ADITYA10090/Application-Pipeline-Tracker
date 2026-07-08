import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// The API base is read at runtime from window config / env; in dev we proxy
// /api and /health to the Node gateway so the app is same-origin.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/api": { target: "http://localhost:8080", changeOrigin: true },
      "/health": { target: "http://localhost:8080", changeOrigin: true },
    },
  },
});
