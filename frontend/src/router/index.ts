import { createRouter, createWebHistory } from 'vue-router'
import MainLayout from '../layout/MainLayout.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: MainLayout,
      children: [
        {
          path: '',
          name: 'Dashboard',
          component: () => import('../views/Dashboard.vue')
        },
        {
          path: 'accounts',
          name: 'Accounts',
          component: () => import('../views/AccountPool.vue')
        },
        {
          path: 'apikeys',
          name: 'APIKeys',
          component: () => import('../views/APIKeys.vue')
        }
      ]
    }
  ]
})

export default router
