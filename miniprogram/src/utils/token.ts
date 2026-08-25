// token 存取（独立模块：request/auth 均引用，避免循环依赖在模块求值期互相访问绑定）
const KEY = 'auth_token'

export function getToken(): string {
  return uni.getStorageSync(KEY) || ''
}

export function setToken(t: string) {
  uni.setStorageSync(KEY, t)
}

export function clearToken() {
  uni.removeStorageSync(KEY)
}
