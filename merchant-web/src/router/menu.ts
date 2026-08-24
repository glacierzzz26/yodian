import type { Role } from '@/types'

export interface MenuItem {
  key: string
  icon: string
  label: string
  path: string
  roles: Role[] // 空数组 = 所有角色可见
}

export interface MenuGroup {
  title: string
  items: MenuItem[]
}

// 侧边栏分组（与 商家端原型.html 导航一致，按 16.2 权限矩阵标注角色）
export const MENU_GROUPS: MenuGroup[] = [
  {
    title: '收银工作台',
    items: [
      { key: 'dashboard', icon: '◱', label: '营业概览', path: '/dashboard', roles: ['cashier', 'owner'] },
      { key: 'tables', icon: '▦', label: '桌台图', path: '/tables', roles: ['cashier', 'owner'] },
      { key: 'orders', icon: '☰', label: '订单管理', path: '/orders', roles: ['cashier', 'owner'] },
      { key: 'pos', icon: '✎', label: '代客点单', path: '/pos', roles: ['cashier', 'owner'] },
      { key: 'offline', icon: '⎘', label: '补录人工单', path: '/offline', roles: ['cashier', 'owner'] },
      { key: 'manual', icon: '⚠', label: '需人工处理', path: '/manual', roles: ['cashier', 'owner'] },
    ],
  },
  {
    title: '后厨',
    items: [
      { key: 'kds', icon: '▤', label: 'KDS 大屏', path: '/kds', roles: ['kitchen', 'owner'] },
      { key: 'print', icon: '⎙', label: '打印任务', path: '/print', roles: ['kitchen', 'owner', 'cashier'] },
    ],
  },
  {
    title: '菜单与桌台',
    items: [
      { key: 'menu', icon: '❏', label: '分类与菜品', path: '/menu', roles: ['owner'] },
      { key: 'perhead', icon: '☶', label: '按人收费项', path: '/perhead', roles: ['owner'] },
      { key: 'soldout', icon: '⊘', label: '沽清管理', path: '/soldout', roles: ['owner'] },
      { key: 'import', icon: '⇪', label: '批量导入', path: '/import', roles: ['owner'] },
      { key: 'tbmgr', icon: '▩', label: '桌台管理', path: '/tbmgr', roles: ['owner'] },
      { key: 'qrcode', icon: '▣', label: '二维码管理', path: '/qrcode', roles: ['owner'] },
    ],
  },
  {
    title: '报表与对账',
    items: [
      { key: 'report', icon: '▥', label: '营业报表', path: '/report', roles: ['cashier', 'owner'] },
      { key: 'recon', icon: '⇋', label: '多来源对账', path: '/recon', roles: ['owner'] },
      { key: 'refund', icon: '↩', label: '退款与核销', path: '/refund', roles: ['owner'] },
    ],
  },
  {
    title: '系统',
    items: [
      { key: 'staff', icon: '☺', label: '员工与权限', path: '/staff', roles: ['owner'] },
      { key: 'log', icon: '◷', label: '操作日志', path: '/log', roles: ['owner'] },
      { key: 'setting', icon: '⚙', label: '系统设置', path: '/setting', roles: ['owner'] },
      { key: 'backup', icon: '⛁', label: '备份与恢复', path: '/backup', roles: ['owner'] },
    ],
  },
]

export function visibleGroups(role: Role): MenuGroup[] {
  return MENU_GROUPS
    .map((g) => ({ ...g, items: g.items.filter((it) => it.roles.length === 0 || it.roles.includes(role)) }))
    .filter((g) => g.items.length > 0)
}

export function firstPath(role: Role): string {
  for (const g of MENU_GROUPS) {
    for (const it of g.items) {
      if (it.roles.includes(role)) return it.path
    }
  }
  return '/dashboard'
}
