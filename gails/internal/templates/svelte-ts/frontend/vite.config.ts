import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import gails from "@gailsio/runtime/plugins/vite";

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  plugins: [svelte(), gails("./bindings")],
});
