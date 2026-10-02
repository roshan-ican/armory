import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import { resolve } from "node:path";

export default defineConfig({
  plugins: [svelte()],
  build: {
    outDir: "../backend/web/static/svelte",
    emptyOutDir: true,
    cssCodeSplit: false,
    lib: { entry: resolve(import.meta.dirname, "src/main.js"), formats: ["es"], fileName: () => "kiosk.js" },
    rollupOptions: { output: { assetFileNames: "kiosk.css" } }
  }
});
