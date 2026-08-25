// 环境配置：开发期直连本地 mock 后端；上线替换为 https 域名
// （13 章域名校验 + 4.2 跳转规则硬前置；manifest urlCheck=false 仅限开发工具）。
export const BASE_URL = 'http://localhost:8080/api'
