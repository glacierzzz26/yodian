/// <reference types="vite/client" />

declare module '*.vue' {
  import { DefineComponent } from 'vue'
  // eslint-disable-next-line @typescript-eslint/no-explicit-any, @typescript-eslint/ban-types
  const component: DefineComponent<{}, {}, any>
  export default component
}

// 支付宝小程序全局对象（仅在 MP-ALIPAY #ifdef 分支内引用；声明避免 vue-tsc 报 undefined）
declare const my: any
