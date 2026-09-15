// 网络层封装（16.1 信封唯一权威）：code≠0 → 统一 toast + reject(BizError)；HTTP 401 → 静默重登不打断当前流程。
// 约定：业务页面不得直接 uni.request；本模块是唯一出网口（收敛点 #0）。
import { BASE_URL } from '@/config'
import { getToken } from './token'
import type { ApiResp } from '@/types/api'

export interface BizError {
  __biz: true
  code: number
  msg: string
}

// @dcloudio/types 的 RequestOptions.method 未含 PATCH，但 uni.request 运行时支持（16.1 的
// PATCH /sessions/:sid/pax 需要）。此处放宽 method 联合，其余字段沿用官方类型。
export interface RequestOptions extends Omit<UniApp.RequestOptions, 'method'> {
  auth?: boolean // 默认 true：自动带 Authorization
  method?: UniApp.RequestOptions['method'] | 'PATCH'
}

// 401 静默重登处理器由 utils/auth.ts 注册（setReloginHandler），避免 request↔auth 循环依赖。
// 返回值不消费（silentLogin 返回 LoginResp），故用 Promise<unknown> 放宽类型。
let reloginHandler: (() => Promise<unknown>) | null = null
export function setReloginHandler(fn: () => Promise<unknown>) {
  reloginHandler = fn
}

let loggingIn = false
const waiters: Array<() => void> = []

function requestRaw<T>(opts: RequestOptions): Promise<T> {
  return new Promise((resolve, reject) => {
    const doRequest = () => {
      uni.request({
        ...opts,
        // PATCH 不在官方 RequestOptions.method 联合中（运行时支持），此处收敛转型
        method: opts.method as UniApp.RequestOptions['method'],
        url: /^https?:\/\//.test(opts.url) ? opts.url : BASE_URL + opts.url,
        header: {
          'Content-Type': 'application/json',
          ...(opts.auth === false || !getToken() ? {} : { Authorization: 'Bearer ' + getToken() }),
          ...(opts.header || {}),
        },
        success: (res) => {
          // HTTP 401 → 静默重登（token 过期/被清），登成后重放原请求
          if (res.statusCode === 401) {
            if (loggingIn) {
              waiters.push(() => doRequest())
              return
            }
            loggingIn = true
            if (!reloginHandler) {
              loggingIn = false
              reject(res)
              return
            }
            reloginHandler()
              .then(() => {
                loggingIn = false
                waiters.splice(0).forEach((fn) => fn())
                doRequest()
              })
              .catch(() => {
                loggingIn = false
                waiters.splice(0).forEach((fn) => fn())
                reject(res)
              })
            return
          }
          const body = res.data as ApiResp<T> | undefined
          if (body && body.code !== 0) {
            uni.showToast({ title: body.msg || '请求失败', icon: 'none' })
            const err: BizError = { __biz: true, code: body.code, msg: body.msg }
            reject(err)
            return
          }
          if (body) {
            resolve(body.data)
          } else {
            reject(res)
          }
        },
        fail: (err) => reject(err),
      })
    }
    doRequest()
  })
}

export function request<T = unknown>(opts: RequestOptions): Promise<T> {
  return requestRaw<T>(opts)
}
