// 平台差异收敛点 #1：静默登录。仅本文件允许出现登录相关 #ifdef；业务页面零平台判断（9.3 硬约束）。
// 微信 uni.login / 支付宝 my.getAuthCode 取 code → POST /auth/login（mock：code 即 channel_uid，
// 真实 code2session 在阶段 4 资质到位后由后端承接，前端不变）。不索取昵称头像（9.4）。
import { request, setReloginHandler } from './request'
import { setToken } from './token'
import type { LoginResp } from '@/types/api'

export type Channel = 'wechat' | 'alipay'

export function platform(): Channel {
  // #ifdef MP-WEIXIN
  return 'wechat'
  // #endif
  // #ifdef MP-ALIPAY
  return 'alipay'
  // #endif
  // #ifndef MP-WEIXIN || MP-ALIPAY
  return 'wechat' // H5/兜底（仅编译期兜底，不参与业务）
  // #endif
}

// #ifdef MP-WEIXIN
function getLoginCode(): Promise<string> {
  return new Promise((resolve, reject) => {
    uni.login({
      provider: 'weixin',
      success: (r) => resolve(r.code || ''),
      fail: (e) => reject(e),
    })
  })
}
// #endif

// #ifdef MP-ALIPAY
function getLoginCode(): Promise<string> {
  return new Promise((resolve, reject) => {
    my.getAuthCode({
      scopes: 'auth_base',
      success: (r: { authCode?: string }) => resolve(r.authCode || ''),
      fail: (e) => reject(e),
    })
  })
}
// #endif

export async function silentLogin(): Promise<LoginResp> {
  const code = await getLoginCode()
  return request<LoginResp>({
    url: '/auth/login',
    method: 'POST',
    data: { channel: platform(), code },
    auth: false,
  })
}

// relogin：401 静默重登（request.ts 的 401 分支调用）；登成即写 token。
export async function relogin(): Promise<LoginResp> {
  const resp = await silentLogin()
  setToken(resp.token)
  return resp
}

// 注册给 request.ts：避免 auth↔request 循环依赖（模块加载期无互相访问）。
setReloginHandler(relogin)
