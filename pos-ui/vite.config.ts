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
    allowedHosts: ['host.docker.internal'],
    port: 5173,
  },
});
