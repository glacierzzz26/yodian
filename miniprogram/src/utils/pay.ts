// 平台差异收敛点 #2：小程序支付（requestPayment）。阶段 3.4 接入 pay 页。
// 微信：wx.requestPayment（paySign 等参数由后端 prepay 下发的 params 透传）；
// 支付宝：my.tradePay（后端 prepay 下发 tradeNO）。
// mock 后端当前只返回占位 params；真实统一下单在阶段 4 联调。
export interface PaymentParams {
  // 微信
  timeStamp?: string
  nonceStr?: string
  package?: string
  signType?: string
  paySign?: string
  // 支付宝
  tradeNO?: string
}

// #ifdef MP-WEIXIN
export function requestPayment(params: PaymentParams): Promise<void> {
  return new Promise((resolve, reject) => {
    uni.requestPayment({
      provider: 'wxpay',
      timeStamp: params.timeStamp || '',
      nonceStr: params.nonceStr || '',
      package: params.package || '',
      signType: params.signType || 'RSA',
      paySign: params.paySign || '',
      success: () => resolve(),
      fail: (e) => reject(e),
    })
  })
}
// #endif

// #ifdef MP-ALIPAY
export function requestPayment(params: PaymentParams): Promise<void> {
  return new Promise((resolve, reject) => {
    my.tradePay({
      tradeNO: params.tradeNO || '',
      success: () => resolve(),
      fail: (e) => reject(e),
    })
  })
}
// #endif
