import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { resolveDevServerProxy } from "./src/bootstrap/devServerProxyConfig";

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: resolveDevServerProxy()
  },
  build: {
    outDir: "../static/dist",
    emptyOutDir: true
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: "./src/testing/setup.ts"
  }
});
