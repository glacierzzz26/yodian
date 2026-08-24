# 阶段 2 · Go 后端工程契约（定稿）

> 状态：定稿待执行
> 业务契约：`docs/扫码点餐系统设计方案.md` v1.8（7.1 表结构 / 7.2 状态机 / 7.4 接口 / 8 配置 / 16 鉴权 为唯一权威）
> 本文件只做「工程落地决策 + 子阶段拆分」，不重复抄业务契约，冲突时以设计方案 v1.8 为准。

## 1. 范围与边界

**做**：Go 后端服务，覆盖设计文档第 3/5/7/8/16 章在服务端的全部内容——订单状态机、支付网关抽象层（mock 优先）、桌台会话与三态结账、退款分流、六口径对账、员工鉴权与权限矩阵、操作日志、限流熔断、WS 推送、打印任务、备份接口。

**不做**（归属后续阶段）：前端从 mock 切真实 API（阶段 4 联调）、云部署 compose/systemd 生产化（阶段 4）、备份任务实际运行与恢复演练（阶段 4）、二维码签名与桌贴 URL 的 H5 中间页（阶段 3）。

## 2. 技术选型

| 组件 | 选择 | 理由 |
|---|---|---|
| HTTP | Gin | 设计 16.2 明确用 gin；中间件生态齐全 |
| ORM | GORM + `datatypes.JSON`（JSONB） | 设计 7.1 给出两种选项，GORM 开发效率高 |
| 迁移 | golang-migrate（SQL 文件，up/down 成对） | 设计 15.5 要求「DB 迁移先备份 + 可逆脚本」 |
| DB | PostgreSQL（开发用 docker compose 本地实例；生产用云 PG，阶段 4） | 设计 15.8 条件① |
| 缓存/锁/限流 | Redis（go-redis/v9） | 幂等锁、会话锁、限流窗口、沽清缓存 |
| JWT | golang-jwt/jwt/v5 | 顾客/员工各一枚签名密钥 |
| 密码 | bcrypt（cost=12） | 设计 16.4 要求 cost ≥ 10 |
| WS | gorilla/websocket | `/ws/kitchen` 握手 `?token=`，心跳 30s |
| 配置 | godotenv + os.Getenv；`.env` + `.env.example` | 设计 8 全部外部依赖可配置 |
| 日志 | log/slog | 标准库够用 |
| 金额 | **int64 分**，JSON 传输整数；入库 `NUMERIC(10,2)` 时 ÷100 | 设计 3.2 硬约束，禁止 float |

## 3. 目录结构

```
server/
├── cmd/server/main.go          # 入口：配置 → 连接 → 迁移 → 路由 → 启动
├── internal/
│   ├── config/config.go        # 加载 .env（8 章全部变量，含 MOCK_* 开关）
│   ├── model/                  # GORM models（对应 7.1 全部表）
│   ├── store/                  # repository 层（订单/会话/账单/菜品/支付…）
│   ├── service/                # 业务层：状态机 / 结账 / 对账 / 退款
│   ├── pay/                    # PaymentGateway 接口 + wechat/alipay/mock 适配器
│   ├── handler/                # HTTP handlers（按 7.4 接口表）
│   ├── middleware/             # JWTAuth / RequireRole / 限流 / 包络
│   └── pkg/respond/            # 统一包络 {code,msg,data} + 错误码表
├── migrations/                 # golang-migrate：0001_init.up/down.sql + 0002_seed
├── deploy/docker-compose.dev.yml   # 本地 PG + Redis
├── .env.example
├── Makefile                    # run / migrate-up / down / test / compose-up
└── go.mod
```

## 4. 工程约束（后端执行必须遵守）

1. **响应信封**：`{"code":0,"msg":"ok","data":...}`；`code=0` 成功，非 0 业务错误；HTTP 200 含业务错误，400 参数错 / 401 未鉴权 / 403 无权限 / 429 限流 / 5xx 系统（16.1）。所有 handler 统一经 respond 包络，禁止裸写 JSON。
2. **错误码表**：1xxxx 顾客端/参数、2xxxx 订单/桌台/会话、3xxxx 支付、4xxxx 鉴权/权限、5xxxx 系统/第三方（16.1）。落地 `pkg/respond/errors.go` 常量。
3. **金额 int64 分**：JSON 出入一律整数；GORM 模型存 `NUMERIC` 字段为 `int64`（存分），或模型 `int64` 分 + 手写转换。**统一：模型字段 `MoneyAmount int64`（分）**，序列化直接输出分。
4. **枚举 ASCII 小写**：全库枚举入库值与前端 mock 完全一致（`pending/paid/…/unsettled`、`wechat/alipay/cash/pos`、`prepay/postpay/frontend`、`dish/per_head`）。禁止中文入库（7.1 枚举表唯一权威）。
5. **路由前缀 `/api`**：业务路径以 7.4 接口表为准（`/admin/orders/offline`、`/admin/sessions/:sid/close`…）。**16.2 示例中的 `/api/admin` + `/admin/...` 双 admin 前缀为文档笔误，实际按 7.4 单层前缀**。
6. **mock 开关正交**（8 章）：`ENABLE_*` 与 `MOCK_*` 独立。Mock 网关的 `handlePaidSuccess` 内部处理函数与真实渠道**完全相同**，只替换「验签/预下单」适配段。
7. **幂等三道防线**（3.3/6.10）：Redis 锁（`pay:lock:<out_trade_no>`）+ `orders.out_trade_no` 唯一 + `payments` 部分唯一索引；唯一键冲突一律按「重复支付」处理（告警 + 标记需人工 + 自动退款流程，16.4）。
8. **种子数据与前端 mock 一致**：seed 建「悦点」1 店、分类/18 道菜/桌台区域与 `merchant-web/src/api/mock.ts` 同名同价同状态，保证阶段 4 前端切换后数据可对照。
9. **凭据**（12.6）：密钥/证书/私钥只进 `.env` 不进仓库；`.env.example` 全占位符；支付证书目录权限 600；`gitignore` 排除 `.env` 与 `certs/`。

## 5. 子阶段拆分与验收

| 子阶段 | 交付物 | 独立验收 |
|---|---|---|
| **2.1 工程地基** | 工程骨架 + 配置 + PG/Redis 连接 + 7.1 全表迁移 + seed + 统一包络/错误码 + `/api/health` | `docker compose up` 后迁移成功、全表存在、`/api/health` 200、`/api/nonexist` 返回统一信封 |
| **2.2 核心交易闭环** | 顾客登录（mock code）、`GET /tables/:tid/menu`（签名+沽清）、`POST /orders`（服务端算金额）、`POST /pay/prepay` + Mock 网关 + 回调幂等、prepay 状态机 | curl 走通 下单→预下单→mock 回调→订单已支付；重复回调不重复入账 |
| **2.3 会话与三态结账** | table_sessions 开/清台、bills 结账单（服务端聚合）、后付/前台结状态机、防并发唯一索引 + 锁单、清台强制校验 + 店长核销、按人项会话级一次性计入、退款分流 | 后付下单即出单→结账生成唯一 bill→支付批量置位→清台；并发结账仅一张 bill；未结账清台被拒 |
| **2.4 商家/管理 API + 运维** | 员工 JWT + 权限矩阵、订单状态/补录/沽清/退款/核销、六口径对账、操作日志、备份接口、WS `/ws/kitchen`、打印任务 mock、限流熔断 | curl 冒烟全部 admin 端点；越权返回 40003；WS 收到新单推送 |

每子阶段结束：写 `docs/phase2/stages/<n>.md` 归档 + 更新 `进度总表.md` + 冻结该子阶段。

## 6. 执行状态

- [x] **2.1 工程地基**（`stages/2-1.md`）：骨架/迁移/seed/health ✅
- [x] **2.2 核心交易闭环**（`stages/2-2.md`）：登录/菜单签名沽清/下单服务端算金额/支付 mock/回调幂等三道防线/状态机 ✅
- [x] **2.3 会话与三态结账**（`stages/2-3.md`）：bills 服务端聚合/防并发双防线/清台强制校验/店长核销/按人项会话级一次性/退款分流/员工登录 ✅
- [x] **2.4 商家/管理 API + 运维**（`stages/2-4.md`）：单 `/api/admin` 前缀 + 权限矩阵三组 / 补录/改状态/前台收银入账/沽清/核销/退款 / 六口径对账（diff=0）/ 操作日志 audit_logs / WS `/ws/kitchen` / 限流 ✅

设计缺口记录（对照现状发现，已定稿）：
- `employees` 表：7.1 缺失，按 16.2 员工 JWT + 前端 Staff 契约补齐。
- 下架 vs 沽清：后端仅有 `dishes.is_sold_out`，前端两种 tag 统一映射（阶段 4 收口）。
- 路由前缀：以 7.4 接口表为准（16.2 示例双 admin 前缀为笔误）。
- `orders.status` 列宽：7.2 枚举含 `partially_refunded`（18 字符），`VARCHAR(16)` 溢出 → `0003` 扩为 `VARCHAR(32)`。
- 会话状态三态：账单支付成功 → 会话置 `settled`（3.7 权威），2.4 修正 2.3 的「仅解锁」实现；`CloseSession` 接受 `active|settled`。
- 六口径对账口径：per-head 折入账单不折入订单 → 应收 = 订单直付（bill_id NULL）+ 已支付账单（含 closed）；`diff = 应收合计 − 支付实收`，0 为平（`stages/2-4.md`）。

## 7. 阶段 2.1 任务分解（已完成，留档）

1. `go mod init` + 拉依赖（gin/gorm/pgx/golang-migrate/jwt/bcrypt/go-redis/godotenv/gorilla-websocket/datatypes）
2. `deploy/docker-compose.dev.yml`：PG16 + Redis7，端口 5432/6379，持久卷
3. `.env.example`（8 章变量全量）+ `config.go` 解析
4. `migrations/0001_init.up/down.sql`：7.1 全表（含 service_calls）逐字落地
5. `migrations/0002_seed`：悦点 1 店 + 分类 + 18 菜 + 桌台区域 + 3 员工（owner/cashier/kitchen）+ 二维码签名密钥测试值
6. `pkg/respond`：信封 + 错误码常量
7. `model/` 全表 GORM models
8. `cmd/server/main.go`：连接 → 迁移（启动时自动 up）→ 路由（health/notfound 走信封）→ 启动
9. 验收：compose up → 迁移 → 表齐全 → `/api/health` 200
