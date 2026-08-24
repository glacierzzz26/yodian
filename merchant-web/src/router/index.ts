import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import type { Role } from '@/types'
import { useAuthStore } from '@/stores/auth'
import { firstPath } from './menu'

// 静态 import 映射（生产构建必需）：`import(`@/views/${path}.vue`)` 含运行时变量，
// Rolldown 无法静态收集 → 生产包缺全部页面 → 点击菜单空白（dev 因实时解析 /src 而不受影响）。
// 显式列出全部视图，Vite 才能分包 + 生成真实 chunk URL。
const VIEW_COMPONENTS = {
  Login: () => import('@/views/Login.vue'),
  Kds: () => import('@/views/Kds.vue'),
  Dashboard: () => import('@/views/pages/Dashboard.vue'),
  TableMap: () => import('@/views/pages/TableMap.vue'),
  OrderList: () => import('@/views/pages/OrderList.vue'),
  Pos: () => import('@/views/pages/Pos.vue'),
  Offline: () => import('@/views/pages/Offline.vue'),
  Manual: () => import('@/views/pages/Manual.vue'),
  PrintTasks: () => import('@/views/pages/PrintTasks.vue'),
  Menu: () => import('@/views/pages/Menu.vue'),
  Perhead: () => import('@/views/pages/Perhead.vue'),
  Soldout: () => import('@/views/pages/Soldout.vue'),
  Import: () => import('@/views/pages/Import.vue'),
  Tbmgr: () => import('@/views/pages/Tbmgr.vue'),
  Qrcode: () => import('@/views/pages/Qrcode.vue'),
  Report: () => import('@/views/pages/Report.vue'),
  Recon: () => import('@/views/pages/Recon.vue'),
  Refund: () => import('@/views/pages/Refund.vue'),
  Staff: () => import('@/views/pages/Staff.vue'),
  Log: () => import('@/views/pages/Log.vue'),
  Setting: () => import('@/views/pages/Setting.vue'),
  Backup: () => import('@/views/pages/Backup.vue'),
} as const

type ViewKey = keyof typeof VIEW_COMPONENTS
const lazy = (key: ViewKey) => VIEW_COMPONENTS[key]

// 路由 key → 视图组件 key（显式映射，页面命名可自由描述，不必等于路由 key）
const VIEW_MAP: Record<string, ViewKey> = {
  dashboard: 'Dashboard',
  tables: 'TableMap',
  orders: 'OrderList',
  pos: 'Pos',
  offline: 'Offline',
  manual: 'Manual',
  print: 'PrintTasks',
  menu: 'Menu',
  perhead: 'Perhead',
  soldout: 'Soldout',
  import: 'Import',
  tbmgr: 'Tbmgr',
  qrcode: 'Qrcode',
  report: 'Report',
  recon: 'Recon',
  refund: 'Refund',
  staff: 'Staff',
  log: 'Log',
  setting: 'Setting',
  backup: 'Backup',
}

// 页面名 → 允许角色（16.2 权限矩阵）
const ROUTE_ROLES: Record<string, Role[]> = {
  dashboard: ['cashier', 'owner'],
  tables: ['cashier', 'owner'],
  orders: ['cashier', 'owner'],
  pos: ['cashier', 'owner'],
  offline: ['cashier', 'owner'],
  manual: ['cashier', 'owner'],
  print: ['kitchen', 'owner', 'cashier'],
  menu: ['owner'],
  perhead: ['owner'],
  soldout: ['owner'],
  import: ['owner'],
  tbmgr: ['owner'],
  qrcode: ['owner'],
  report: ['cashier', 'owner'],
  recon: ['owner'],
  refund: ['owner'],
  staff: ['owner'],
  log: ['owner'],
  setting: ['owner'],
  backup: ['owner'],
}

const pageKeys = Object.keys(ROUTE_ROLES)

const routes: RouteRecordRaw[] = [
  { path: '/login', name: 'login', component: lazy('Login'), meta: { public: true } },
  // KDS 后厨大屏：独立全屏暗色视图，不带商家端外壳
  { path: '/kds', name: 'kds', component: lazy('Kds'), meta: { roles: ['kitchen', 'owner'] } },
  {
    path: '/',
    component: () => import('@/layouts/MainLayout.vue'),
    children: [
      // 根路径直达营业概览：否则 / 只匹配父路由无子路由，内容区空白
      { path: '', redirect: '/dashboard' },
      ...pageKeys
        .filter((key) => key !== 'kds')
        .map((key) => ({
          path: key,
          name: key,
          component: lazy(VIEW_MAP[key]),
          meta: { roles: ROUTE_ROLES[key], title: key },
        })),
    ],
  },
  { path: '/:pathMatch(.*)*', redirect: '/dashboard' },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.meta.public) {
    return auth.isLoggedIn ? { path: firstPath(auth.role as Role) } : true
  }
  if (!auth.isLoggedIn) return { path: '/login', query: { redirect: to.fullPath } }
  const roles = (to.meta.roles ?? []) as Role[]
  if (roles.length && !roles.includes(auth.role as Role)) {
    return { path: firstPath(auth.role as Role) }
  }
  return true
})

export default router
