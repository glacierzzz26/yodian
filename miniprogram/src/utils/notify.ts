// 平台差异收敛点 #3：出餐/订单通知（订阅消息/模板消息）。9.3 决策：MVP 不做通知，订单页轮询即可。
// 本模块保留骨架，阶段 4 若启用订阅消息再落地；业务暂不调用。
// #ifdef MP-WEIXIN
export function requestNotifyPermission(): Promise<boolean> {
  // TODO(4): uni.requestSubscribeMessage（微信订阅消息）
  return Promise.resolve(false)
}
// #endif
// #ifdef MP-ALIPAY
export function requestNotifyPermission(): Promise<boolean> {
  // TODO(4): 支付宝模板消息
  return Promise.resolve(false)
}
// #endif
