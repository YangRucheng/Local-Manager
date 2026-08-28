import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'ledger', component: () => import('@/views/LedgerView.vue') },
    { path: '/manage', name: 'manage', component: () => import('@/views/ManageView.vue') },
  ],
})

export default router
