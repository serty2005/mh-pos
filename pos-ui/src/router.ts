import { createRouter, createWebHistory } from 'vue-router';

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: () => import('./pages/RootRedirect.vue') },
    { path: '/pair', component: () => import('./pages/PairPage.vue') },
    { path: '/login', component: () => import('./pages/LoginPage.vue') },
    { path: '/lock', component: () => import('./pages/LockPage.vue') },
    { path: '/pos', component: () => import('./pages/PosPage.vue') },
    { path: '/pos/cashier', component: () => import('./pages/PosPage.vue') },
    {
      path: '/pos/waiter',
      component: () => import('./pages/WorkspaceShellPage.vue'),
      meta: { titleKey: 'pos.waiterMobile', bodyKey: 'pos.interfaceNotAvailableBody' },
    },
    {
      path: '/pos/kitchen',
      component: () => import('./pages/WorkspaceShellPage.vue'),
      meta: { titleKey: 'pos.kitchenDisplay', bodyKey: 'pos.interfaceNotAvailableBody' },
    },
    {
      path: '/pos/manager',
      component: () => import('./pages/WorkspaceShellPage.vue'),
      meta: { titleKey: 'pos.managerWorkspace', bodyKey: 'pos.interfaceNotAvailableBody' },
    },
  ],
});
