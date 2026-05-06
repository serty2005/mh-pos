import '@quasar/extras/material-icons/material-icons.css';
import 'quasar/src/css/index.sass';

import { VueQueryPlugin } from '@tanstack/vue-query';
import { Quasar } from 'quasar';
import { createPinia } from 'pinia';
import { createApp } from 'vue';

import App from './App.vue';
import { i18n } from './shared/i18n';
import { queryClient } from './shared/query';
import { router } from './router';
import './styles.css';

createApp(App)
  .use(createPinia())
  .use(router)
  .use(i18n)
  .use(VueQueryPlugin, { queryClient })
  .use(Quasar, {
    config: {
      brand: {
        primary: '#1f6f5b',
        secondary: '#344054',
        accent: '#b8563f',
        dark: '#202124',
      },
    },
  })
  .mount('#app');
