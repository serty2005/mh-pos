import { createRouter, createWebHistory } from 'vue-router';

import LockPage from './pages/LockPage.vue';
import LoginPage from './pages/LoginPage.vue';
import PairPage from './pages/PairPage.vue';
import PosPage from './pages/PosPage.vue';
import RootRedirect from './pages/RootRedirect.vue';

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: RootRedirect },
    { path: '/pair', component: PairPage },
    { path: '/login', component: LoginPage },
    { path: '/lock', component: LockPage },
    { path: '/pos', component: PosPage },
  ],
});
