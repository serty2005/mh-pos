import { quasar, transformAssetUrls } from '@quasar/vite-plugin';
import vue from '@vitejs/plugin-vue';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [
    vue({
      template: { transformAssetUrls },
    }),
    quasar({
      sassVariables: undefined,
    }),
  ],
  server: {
    allowedHosts: ['host.docker.internal', 'localhost', '127.0.0.1'],
    port: 5174,
  },
});
