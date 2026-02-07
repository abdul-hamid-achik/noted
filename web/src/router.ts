import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'editor',
      component: () => import('./components/AppLayout.vue'),
    },
    {
      path: '/dashboard',
      name: 'dashboard',
      component: () => import('./components/Dashboard.vue'),
    },
    {
      path: '/settings',
      name: 'settings',
      component: () => import('./components/SettingsPage.vue'),
    },
  ],
})

export default router
