import { defineConfig } from 'vite';
import path from 'path';

export default defineConfig({
  resolve: {
    alias: {
      // Use the local repo runtime sources instead of the published package
      '@gailsio/runtime': path.resolve(__dirname, '../../../internal/runtime/desktop/@gailsio/runtime/src/index.ts'),
    },
  },
  server: {
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
});
