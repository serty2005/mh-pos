import '@quasar/extras/material-icons/material-icons.css';
import 'quasar/src/css/index.sass';

import { Quasar } from 'quasar';
import { createApp } from 'vue';

import App from './App.vue';
import { i18n } from './shared/i18n';
import './styles.css';

createApp(App)
  .use(i18n)
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
