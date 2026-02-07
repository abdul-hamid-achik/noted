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
  ],
})

export default router
