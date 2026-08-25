// 平台差异收敛点 #4：扫码入参解析。业务页面零 #ifdef。
// 微信：小程序码/普通链接扫码进小程序 → onLoad(options.scene) 为 URL 编码的查询串 'tid=8&sig=xxx'；
//       开发工具「编译模式」自定义参数直接给 options.tid / options.sig。
// 支付宝：onLoad(options.query) 为解析后的对象 {tid, sig}。
// 两平台最终归一为 { tid, sig }（契约 #5：桌贴 URL 形如 /t?tid=&sig=，无 ts）。
export interface QRParams {
  tid: string
  sig: string
}

function parseQuery(str: string): Record<string, string> {
  const out: Record<string, string> = {}
  if (!str) return out
  str.split('&').forEach((seg) => {
    const idx = seg.indexOf('=')
    if (idx <= 0) return
    const k = seg.slice(0, idx)
    const v = decodeURIComponent(seg.slice(idx + 1).replace(/\+/g, ' '))
    if (k) out[k] = v
  })
  return out
}

// #ifdef MP-WEIXIN
export function parseEntry(options: Record<string, any>): QRParams {
  const fromScene = parseQuery(decodeURIComponent(options.scene || ''))
  return {
    tid: fromScene.tid || options.tid || '',
    sig: fromScene.sig || options.sig || '',
  }
}
// #endif

// #ifdef MP-ALIPAY
export function parseEntry(options: Record<string, any>): QRParams {
  const q = options.query && typeof options.query === 'object' ? options.query : options
  return {
    tid: q.tid || options.tid || '',
    sig: q.sig || options.sig || '',
  }
}
// #endif
