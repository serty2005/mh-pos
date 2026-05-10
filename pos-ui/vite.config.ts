import { quasar, transformAssetUrls } from '@quasar/vite-plugin';
import vue from '@vitejs/plugin-vue';
import { configDefaults, defineConfig } from 'vitest/config';

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
  test: {
    exclude: [...configDefaults.exclude, 'e2e/**'],
  },
});
