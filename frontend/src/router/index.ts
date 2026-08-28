import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/components' },
    // 旧管理页地址兼容
    { path: '/manage', redirect: '/rooms' },
    { path: '/rooms', name: 'rooms', component: () => import('@/views/RoomsView.vue') },
    { path: '/cabinets', name: 'cabinets', component: () => import('@/views/CabinetsView.vue') },
    { path: '/components', name: 'components', component: () => import('@/views/ComponentsView.vue') },
  ],
})

export default router
