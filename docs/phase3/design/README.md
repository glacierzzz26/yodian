# 阶段 3 · 顾客端 uni-app 双小程序 + 中间页 H5 工程契约（定稿）

> 状态：定稿待执行
> 业务契约：`docs/扫码点餐系统设计方案.md` v1.8（第 4 章一码双扫 / 第 9 章顾客端设计 / 3.7 结账三态 / 3.8 按人项 / 3.9 就餐方式 / 7.4 接口表 / 8 配置 / 16 鉴权 为唯一权威）
> 本文件只做「工程落地决策 + 子阶段拆分 + 契约缺口清单」，不重复抄业务契约，冲突时以设计方案 v1.8 为准。
> 阶段 2 续接依据：`phase2/stages/2-4.md`（后端已就绪面 + 留白清单）。

## 1. 范围与边界

**做**：

- `miniprogram/` 顾客端 uni-app：**一套代码双端编译**（微信/支付宝），覆盖第 9 章堂食主流程——静默登录、扫码进台（tid+sig）、选就餐人数（pax）、菜单（分类/沽清/规格）、购物车、下单（服务端算价）、prepay+requestPayment、订单轮询、拼桌会话互见、后付统一结账、换渠道重付、沽清同步、呼叫服务员、手机号授权（可选）。
- `landing/` 中间页 H5：**单文件 HTML 兜底页**（4.3），UA 分流 + 引导 + 二维码无效屏，由 Go 后端 `go:embed` 挂载 `/t`（根路径）。
- 后端顾客端**契约缺口补齐**（7.4 中顾客侧未实现端点，见 §5 缺口清单）。

**不做**（归属后续阶段，现在只留口子）：

- **真实支付资质接入**（微信 code2session、真实统一下单/回调验签）→ 阶段 4 联调（依赖客户资质办理，9.1）。本期顾客端走 mock（`code=uid`、`MOCK_*` 网关）。
- **出餐通知**（订阅消息/模板消息）→ 9.3 决策：MVP 可降级，不做通知功能，订单页轮询即可。
- **自提/外带**（3.9 明确 Phase 2）→ `orders.order_type='dine_in'` 已在代码固定；`pickup_no`/`order_type` 预留字段不动。
- **管理员处理呼叫闭环**（`/admin/service-calls/:id/resolve` + 商家端呼叫 UI）→ 阶段 4；本期只做顾客端发起（`/sessions/:sid/call-waiter`）。
- **菜品图片上传/菜品/分类 CRUD**、员工 CRUD、部分退款、对账按营业日归属 → 阶段 4。

## 2. 技术选型

| 组件 | 选择 | 理由 |
|---|---|---|
| 顾客端框架 | uni-app（Vue3 + Vite + TypeScript），官方 CLI 模板 `uni-preset-vue#vite-ts` | 9.2 定稿：一套代码编译双小程序；支付宝端兼容成熟度优于 Taro |
| UI 库 | **Wot Design Uni**（easycom 自动引入） | 9.2 定稿：原生面向 uni-app、双端编译完整；点餐缺失组件（规格弹窗、人数选择、桌台胶囊）自写 |
| 视觉基调 | 暖色系（橙红主色，餐饮惯例，9.2）；组件库默认变量覆盖主色 | 与商家端 AntV 蓝显专业区分 |
| 状态 | Pinia（购物车草稿、会话/桌台上下文、登录态） | 9.2；提交价以服务端为准，本地仅预览 |
| 网络层 | 封装 `uni.request`：信封 `{code,msg,data}` 拦截、401 静默重登、loading | 16.1 信封唯一权威 |
| 平台差异收敛 | 差异全部收进 `utils/auth.js`（登录取码）/`utils/pay.js`（requestPayment）/`utils/notify.js`（MVP 占位）/`utils/entry.js`（扫码入参解析） | 9.2/9.3 硬约束：业务页面零 `#ifdef` |
| 实时性 | **订单状态轮询 3–5s** 替代 WS（`utils/poller.js`） | 9.3 决策：规避双端 WS 差异，双端一致 |
| 中间页 H5 | 单文件 `index.html`，无框架、内联样式、`go:embed` 进 server 挂 `/t` | 4.3：必须极轻（预算 ≤400ms），随后端一个二进制发布，`QR_BASE_URL` 即根路径 |
| 金额 | **int64 分**，JSON 整数；前端只展示，永远以服务端 `total_amount` 为准 | 3.2 硬约束 |
| 构建脚本 | `npm run dev:mp-weixin` / `dev:mp-alipay` → `dist/dev/mp-*`，导入对应开发者工具 | 微信/支付宝开发者工具在 Windows 宿主（WSL 下走目录映射） |

## 3. 目录结构

```
yodian/
├── docs/phase3/
│   ├── design/README.md      # 本文件
│   └── stages/                # 3.1–3.5 归档
├── landing/
│   └── index.html             # 中间页 H5 单文件（go:embed 进 server）
├── miniprogram/
│   ├── src/
│   │   ├── pages/
│   │   │   ├── entry/         # 扫码入口页（解析 tid/sig → 校验 → 跳菜单）
│   │   │   ├── menu/          # 菜单 + 购物车
│   │   │   ├── order/         # 下单确认（人数/按人项/合计）
│   │   │   ├── pay/           # 支付中转（prepay → requestPayment → 轮询）
│   │   │   ├── orders/        # 本桌会话订单（拼桌互见）
│   │   │   ├── checkout/      # 后付统一结账
│   │   │   └── profile/       # 手机号授权（可选）
│   │   ├── components/        # 规格弹窗 / 人数选择 / 桌台胶囊（自写）
│   │   ├── stores/            # Pinia：cart / session / auth
│   │   ├── utils/             # auth.js pay.js notify.js entry.js poller.js request.js
│   │   ├── types/             # 后端信封/DTO 对齐类型
│   │   ├── styles/            # 主题变量（暖橙）
│   │   ├── App.vue / main.ts / pages.json / manifest.json / uni.scss
│   ├── package.json           # dev:mp-weixin / dev:mp-alipay 脚本
│   └── tsconfig.json
└── server/
    └── internal/…             # 阶段 3 补缺口（§5 清单）+ static/ 嵌 landing/index.html
```

`manifest.json` 双端 appid 为占位（资质办理后填真实值，9.1）；条件编译保证**任一端可单独发版**。

## 4. 工程约束（前端执行必须遵守）

1. **条件编译收敛**：平台差异只在 `utils/auth.js` / `pay.js` / `notify.js` / `entry.js` 四处出现，业务页面零平台判断（9.3 对照表逐项过）。
2. **金额 int64 分**：前端不参与任何金额计算；下单/结账合计一律取服务端响应。
3. **响应信封**：`{code,msg,data}`；`code≠0` 统一提示；HTTP 401 触发静默重登（重新取码换 token，不打断当前流程）。
4. **静默登录**：`uni.login`/`my.getAuthCode` 取码 → `/auth/login` → token；**不索取昵称头像**（9.4）。手机号授权独立于登录，拒绝不影响点餐。
5. **轮询**：订单状态 3–5s；支付后必须轮询定终态，不做本地乐观更新（9.8）。
6. **沽清**：菜单灰显（`is_sold_out`）+ 下单二次校验（服务端拒绝为最终防线，9.6）。
7. **Pax**：1–20 步进选择，服务端兜底（默认 2）；**改人数唯一入口 = `PATCH /sessions/:sid/pax`**（16.1），下单 `pax` 仅用于开台首设。
8. **弱网**：菜单可缓存浏览，但**下单入口禁用**（9.8，禁离线下单）；提交幂等（同一 `out_trade_no`）。
9. **环境**：微信开发者工具 + 支付宝开发者工具（Windows 宿主）；WSL 下 `dist/dev/mp-weixin|alipay` 目录映射到宿主导入。
10. **顾客端不直接打开小程序**：H5 兜底页只做 UA 分流 + 引导 + 无效屏；真正进小程序靠平台「扫普通链接二维码跳小程序」规则（4.2），规则配置与域名校验是上线硬前置（13 章检查单）。

## 5. 契约缺口清单（对照 7.4 逐项核对后端现状，2026-08-25）

### 5.1 已就绪（复用，不改）

| 端点 | 现状 | 备注 |
|---|---|---|
| `POST /auth/login` | ✅ | mock：`code=uid`；响应 `{token, customer_id, channel}`；渠道启用校验 |
| `GET /tables/:tid/menu` | ✅ | 校验 `sig`；返回 `table{id,table_no,seats,pay_mode}` + `categories[{dishes[{id,name,price,is_sold_out,image_url,specs}]}]` + `per_head[{id,name,price,is_required}]` |
| `POST /orders` | ✅ | req `{tid,pax,items[{dish_id,qty,specs?,remark?}]}`；resp `{order_id,out_trade_no,total_amount,status,pay_mode,session_id}`；prepay 首单自动含 per_head（去重） |
| `POST /sessions/:sid/bill` | ✅ | 生成结账单（加锁+唯一索引）；resp `{bill_id,session_id,out_trade_no,total_amount,status}` |
| `POST /pay/notify/mock` | ✅ | dev 挂载，mock 支付成功回调 |

### 5.2 缺口（→ **3.1 后端契约补齐**）

| 端点 | 7.4 契约 | 现状 | 实现方案 |
|---|---|---|---|
| `GET /sessions/:sid/orders` | 取本桌会话全部订单（拼桌共享） | ❌ 无 | service 聚合会话内全部订单 + order_items（含跨渠道），按 created_at 排序，返回 `{session_id, pax, pay_mode, orders[]}` |
| `PATCH /sessions/:sid/pax` | 设置就餐人数 | ❌ 无 | 校验 1–20 → 更新 `table_sessions.pax`；**联动刷新 per_head**（3.8：prepay 首单未付 / 未结账单的 per_head 金额按新 pax 重算） |
| `GET /sessions/:sid/bill/preview` | 结账预览：聚合未结账合计 | ❌ 无 | **只读聚合、不加锁**（与 `POST /bill` 区分——后者锁会话防并发）；返回未结账订单合计 + per_head 计 + 总额 |
| `POST /pay/prepay` | `{order_id 或 bill_id, channel}` | ⚠️ 仅 order_id | handler 接受 `bill_id`；`pay.PrepayReq` 已支持 BillID，网关/mock 已通；回调幂等已覆盖 bill（2.3/2.4） |
| `POST /pay/switch` | 换渠道重付（关原单+新单号） | ❌ 无 | 按时序图 4：CloseOrder 原单 → 生成新 out_trade_no → 新渠道预下单；唯一键冲突按重复支付处理 |
| `GET /orders/:id/status` | 主动查单（弱网补单） | ❌ 无 | 返回订单 `{status, total_amount, pay_mode}` + 支付终态（payments/refunds 摘要） |
| `POST /auth/phone-code` | 发验证码（Redis TTL 5min） | ❌ 无 | `SMS_PROVIDER=mock` 固定码；需顾客 JWT，限流 16.3 |
| `POST /auth/phone-bind` | 校验码并绑定 phone | ❌ 无 | 校验 → 写 `customers.phone` + `phone_verified_at` |
| `POST /sessions/:sid/call-waiter` | 呼叫服务员 | ❌ 无 | 写 `service_calls`（pending）+ WS 推商家端 |
| `GET /per-head-charges` | 取按人项配置 | 菜单接口已带 | **决策：取消独立端点，复用 `menu.per_head`**；3.1 在菜单 DTO 补 `is_default_checked`（数据源 `per_head_charges.is_default_checked`） |

### 5.3 阶段 4（本期不做，留口子）

| 端点/项 | 归属 |
|---|---|
| `POST /admin/service-calls/:id/resolve` + 商家端呼叫处理 UI | 4 |
| 微信 code2session / 支付宝 code 换 user_id（真实登录） | 4（资质到位） |
| 真实渠道统一下单/回调验签 | 4 |
| 备份/降级开关/菜品图片上传/菜品分类 CRUD/员工 CRUD/部分退款/对账按日 | 4 |

### 5.4 设计差异记录（对照现状，已定稿）

- **桌贴签名参数**：设计 4.2 写 `?tid=&ts=&sign=`；**实现为 `?tid=&sig=`**，`sig = HMAC-SHA256(QR_SIGN_SECRET, "s{shop_id}.t{table_id}.v{qr_version}")`，无 `ts`（`qr_version` 承担吊销，4.4 防伪造语义不变）。**顾客端入口页与 H5 一律按 `?tid=&sig=` 解析**（微信 `scene` / 支付宝 `query` 均如此映射）。
- **签名校验位置**：`secret` 不下发，H5 无法本地校验 → **签名校验由顾客端入口页调 `GET /tables/:tid/menu` 完成**，失败显示「二维码无效，请联系服务员」屏（4.3 语义在 mini-program 内承接）；H5 仅 UA 分流 + 引导 + 通用提示。
- **per_head 勾选切换**：3.8 提「默认勾选/可切换」；现后端**全量计入**（prepay 首单 / bill 均自动加全部活动项 × pax），无承载勾选的接口。**MVP 顾客端只展示服务端计量的 per_head 合计，不做切换**（记录为取舍，二期随优惠体系扩展）。
- **就餐方式**：3.9 自提/外带 Phase 2；本期顾客端只做堂食，`order_type='dine_in'` 已在代码固定，就餐方式选择器不做（避免假 UI）。
- **顾客端真实登录**：本期 `code=uid` 直通；顾客端收敛层 `auth.js` 照常取平台 code 上传，后端 mock 语义一致，真实换码留阶段 4。

## 6. 子阶段拆分与验收

| 子阶段 | 交付物 | 独立验收 |
|---|---|---|
| **3.1 后端契约补齐** | §5.2 全部缺口端点 + prepay bill_id + menu per_head 补 `is_default_checked` | curl 冒烟：拼桌订单/改人数/结账预览/prepay(bill_id)→mock 回调→bill paid+session settled/换渠道不重复扣款/查单/手机号绑定/呼叫；越权与参数校验正确 |
| **3.2 中间页 H5 一码双扫** | `landing/index.html` 单文件 + server `go:embed` 挂 `/t`（根路径，非 /api） | curl `GET /t` 返回 HTML；UA 三分支（微信/支付宝→引导、其他→提示）；无 `sig` 兜底文案；`QR_BASE_URL` 命中 |
| **3.3 顾客端骨架 + 登录** | `miniprogram/` 脚手架 + Wot Design Uni + 收敛层 + 静默登录 + 入口页（tid/sig→校验→菜单） | 双端编译通过；微信/支付宝模拟器静默登录拿 `customer_id`；非法 sig 显示无效屏 |
| **3.4 点餐主链路** | 人数选择 → 菜单（分类/沽清/规格）→ 购物车 → 下单 → prepay → requestPayment → mock 回调 | 双端全链路下单+支付成功；prepay 台加菜 per_head 不重复计；下单后入口页到菜单不回头 |
| **3.5 会话/查单/结账/边界** | 拼桌会话互见、结账预览+统一结账、订单轮询补单、换渠道重付、沽清同步、呼叫服务员、手机号授权（可选） | 跨渠道同桌订单互见；统一结账 bill 支付成功 + session settled；弱网模拟轮询补单；换渠道不重复扣款 |

每子阶段结束：写 `docs/phase3/stages/<n>.md` 归档 + 更新 `进度总表.md` + 冻结该子阶段。

## 7. 执行状态

- [x] **3.1 后端契约补齐**（前置，解锁 3.3–3.5）· 归档 `stages/3-1.md`（2026-08-25）
- [x] **3.2 中间页 H5 一码双扫**（可与 3.1 并行）· 归档 `stages/3-2.md`（2026-08-25）
- [x] **3.3 顾客端骨架 + 登录**（依赖 3.1）· 归档 `stages/3-3.md`（2026-08-25）
- [x] **3.4 点餐主链路**（依赖 3.3）· 归档 `stages/3-4.md`（2026-08-25）
- [x] **3.5 会话/查单/结账/边界**（依赖 3.3）· 归档 `stages/3-5.md`（2026-08-25）
