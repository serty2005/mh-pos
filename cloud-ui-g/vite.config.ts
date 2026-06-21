import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

const env = globalThis as typeof globalThis & {
  process?: {
    env?: Record<string, string | undefined>;
    platform?: string;
  };
};
const disableHmr = env.process?.env?.DISABLE_HMR === 'true';
const usePollingWatch = env.process?.platform === 'linux' && env.process?.env?.VITE_USE_NATIVE_WATCH !== 'true';

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    host: '0.0.0.0',
    port: 5174,
    allowedHosts: ['host.docker.internal'],
    hmr: !disableHmr,
    // На Linux polling не расходует inotify watchers и предотвращает ENOSPC при обычном npm run dev.
    watch: disableHmr ? null : usePollingWatch ? { usePolling: true, interval: 1000 } : {},
  },
});
