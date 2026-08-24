-- 0001_init: 7.1 表结构逐字落地（设计方案 v1.8）
-- 金额统一 NUMERIC(10,2) 存「元」，Go 业务层用 int64 分（3.2）
-- 枚举一律 ASCII 小写（7.1 枚举表唯一权威）
-- 补充表：employees —— 设计文档 7.1 缺失、按 16.2 员工 JWT 与前端 Staff 契约补齐

CREATE TABLE shops (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(128) NOT NULL,
  address VARCHAR(256),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE table_areas (
  id BIGSERIAL PRIMARY KEY,
  shop_id BIGINT NOT NULL REFERENCES shops(id),
  name VARCHAR(32) NOT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  default_pay_mode VARCHAR(16),
  is_active BOOLEAN NOT NULL DEFAULT true,
  UNIQUE(shop_id, name)
);

CREATE TABLE tables (
  id BIGSERIAL PRIMARY KEY,
  shop_id BIGINT NOT NULL REFERENCES shops(id),
  area_id BIGINT REFERENCES table_areas(id),
  table_no VARCHAR(32) NOT NULL,
  seats INT NOT NULL DEFAULT 2,
  status VARCHAR(16) NOT NULL DEFAULT 'empty',
  pay_mode VARCHAR(16) NOT NULL DEFAULT 'prepay',
  sort_order INT NOT NULL DEFAULT 0,
  remark VARCHAR(128),
  qr_version INT NOT NULL DEFAULT 1,
  qrcode_url VARCHAR(512),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(shop_id, table_no)
);
COMMENT ON COLUMN tables.pay_mode IS 'prepay 先付后吃 / postpay 后付统一结账 / frontend 仅点单前台结';
COMMENT ON COLUMN tables.area_id IS '区域/位置，桌台图分组、报表按区域统计、小票打印标注';

CREATE TABLE customers (
  id BIGSERIAL PRIMARY KEY,
  channel VARCHAR(16) NOT NULL,
  channel_uid VARCHAR(128) NOT NULL,
  nickname VARCHAR(64),
  phone VARCHAR(20),
  phone_verified_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(channel, channel_uid)
);
COMMENT ON COLUMN customers.channel IS 'wechat / alipay';
COMMENT ON COLUMN customers.channel_uid IS '微信 openid 或支付宝 user_id';

CREATE TABLE table_sessions (
  id BIGSERIAL PRIMARY KEY,
  shop_id BIGINT NOT NULL REFERENCES shops(id),
  table_id BIGINT NOT NULL REFERENCES tables(id),
  pay_mode VARCHAR(16) NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'active',
  locked_for_bill BOOLEAN NOT NULL DEFAULT false,
  pax INT NOT NULL DEFAULT 2,
  opened_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  closed_at TIMESTAMPTZ
);
COMMENT ON COLUMN table_sessions.status IS 'active 进行中 / settled 已结账 / closed 已清台';
CREATE UNIQUE INDEX uniq_active_session_per_table
  ON table_sessions(table_id) WHERE status = 'active';

CREATE TABLE bills (
  id BIGSERIAL PRIMARY KEY,
  shop_id BIGINT NOT NULL REFERENCES shops(id),
  session_id BIGINT NOT NULL REFERENCES table_sessions(id),
  total_amount NUMERIC(10,2) NOT NULL,
  discount_amount NUMERIC(10,2) NOT NULL DEFAULT 0,
  payable_amount NUMERIC(10,2) NOT NULL,
  status VARCHAR(16) NOT NULL,
  pay_channel VARCHAR(16),
  out_trade_no VARCHAR(64) UNIQUE,
  operator_id BIGINT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  paid_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX uniq_session_active_bill
  ON bills(session_id) WHERE status IN ('pending','paid');

CREATE TABLE per_head_charges (
  id BIGSERIAL PRIMARY KEY,
  shop_id BIGINT NOT NULL REFERENCES shops(id),
  name VARCHAR(64) NOT NULL,
  price NUMERIC(10,2) NOT NULL,
  is_default BOOLEAN NOT NULL DEFAULT true,
  is_required BOOLEAN NOT NULL DEFAULT false,
  is_active BOOLEAN NOT NULL DEFAULT true,
  sort INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
COMMENT ON COLUMN per_head_charges.price IS '单价（元/位），金额 = 单价 × 就餐人数';

CREATE TABLE categories (
  id BIGSERIAL PRIMARY KEY,
  shop_id BIGINT NOT NULL REFERENCES shops(id),
  name VARCHAR(64) NOT NULL,
  sort INT NOT NULL DEFAULT 0
);

CREATE TABLE dishes (
  id BIGSERIAL PRIMARY KEY,
  shop_id BIGINT NOT NULL REFERENCES shops(id),
  category_id BIGINT NOT NULL REFERENCES categories(id),
  name VARCHAR(128) NOT NULL,
  price NUMERIC(10,2) NOT NULL,
  specs JSONB,
  is_sold_out BOOLEAN NOT NULL DEFAULT false,
  image_url VARCHAR(512),
  sort INT NOT NULL DEFAULT 0
);

CREATE TABLE orders (
  id BIGSERIAL PRIMARY KEY,
  shop_id BIGINT NOT NULL REFERENCES shops(id),
  table_id BIGINT REFERENCES tables(id),
  session_id BIGINT REFERENCES table_sessions(id),
  customer_id BIGINT REFERENCES customers(id),
  bill_id BIGINT REFERENCES bills(id),
  pay_mode VARCHAR(16) NOT NULL DEFAULT 'prepay',
  order_type VARCHAR(16) NOT NULL DEFAULT 'dine_in',
  pickup_no VARCHAR(16),
  source VARCHAR(16) NOT NULL DEFAULT 'online',
  status VARCHAR(16) NOT NULL,
  pay_channel VARCHAR(16),
  operator_id BIGINT,
  total_amount NUMERIC(10,2) NOT NULL,
  discount_amount NUMERIC(10,2) NOT NULL DEFAULT 0,
  paid_amount NUMERIC(10,2),
  member_id BIGINT,
  out_trade_no VARCHAR(64) UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  paid_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ
);
COMMENT ON COLUMN orders.pay_channel IS 'wechat / alipay / cash / pos（二期扩 balance 储值）';
COMMENT ON COLUMN orders.source IS 'online 扫码单 / offline 人工补录单（二期扩 staff 服务员代点）';
COMMENT ON COLUMN orders.status IS '枚举见 7.1 枚举表：prepay 主干 pending/paid/preparing/served/done/…；postpay|frontend 主干 unsettled/paid/…';
CREATE INDEX idx_orders_session ON orders(session_id);
CREATE INDEX idx_orders_status_created ON orders(status, created_at);
CREATE INDEX idx_orders_unsettled ON orders(session_id) WHERE status = 'unsettled';

CREATE TABLE order_items (
  id BIGSERIAL PRIMARY KEY,
  order_id BIGINT NOT NULL REFERENCES orders(id),
  dish_id BIGINT REFERENCES dishes(id),
  item_type VARCHAR(16) NOT NULL DEFAULT 'dish',
  name VARCHAR(128) NOT NULL,
  price NUMERIC(10,2) NOT NULL,
  qty INT NOT NULL DEFAULT 1,
  specs JSONB,
  remark VARCHAR(128),
  is_served BOOLEAN NOT NULL DEFAULT false,
  is_refunded BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE payments (
  id BIGSERIAL PRIMARY KEY,
  order_id BIGINT REFERENCES orders(id),
  bill_id BIGINT REFERENCES bills(id),
  out_trade_no VARCHAR(64) NOT NULL UNIQUE,
  channel VARCHAR(16) NOT NULL,
  channel_trade_no VARCHAR(64),
  amount NUMERIC(10,2) NOT NULL,
  status VARCHAR(16) NOT NULL,
  callback_raw JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT chk_payment_subject CHECK (
    (order_id IS NOT NULL AND bill_id IS NULL) OR
    (order_id IS NULL AND bill_id IS NOT NULL)
  )
);
COMMENT ON COLUMN payments.channel IS 'wechat / alipay';
COMMENT ON COLUMN payments.channel_trade_no IS '微信 transaction_id / 支付宝 trade_no';
COMMENT ON COLUMN payments.callback_raw IS '回调原文存档，对账与纠纷举证唯一凭据';

CREATE UNIQUE INDEX uniq_order_success_payment
  ON payments(order_id) WHERE status = 'success' AND order_id IS NOT NULL;
CREATE UNIQUE INDEX uniq_bill_success_payment
  ON payments(bill_id) WHERE status = 'success' AND bill_id IS NOT NULL;

CREATE TABLE refunds (
  id BIGSERIAL PRIMARY KEY,
  order_id BIGINT REFERENCES orders(id),
  bill_id BIGINT REFERENCES bills(id),
  payment_id BIGINT REFERENCES payments(id),
  out_refund_no VARCHAR(64) NOT NULL UNIQUE,
  channel VARCHAR(16) NOT NULL,
  amount NUMERIC(10,2) NOT NULL,
  status VARCHAR(16) NOT NULL,
  reason VARCHAR(128),
  operator_id BIGINT,
  callback_raw JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT chk_refund_subject CHECK (
    (order_id IS NOT NULL AND bill_id IS NULL) OR
    (order_id IS NULL AND bill_id IS NOT NULL)
  )
);

CREATE TABLE print_tasks (
  id BIGSERIAL PRIMARY KEY,
  order_id BIGINT REFERENCES orders(id),
  bill_id BIGINT REFERENCES bills(id),
  printer_sn VARCHAR(64) NOT NULL,
  station VARCHAR(32),
  status VARCHAR(16) NOT NULL DEFAULT 'pending',
  retry_count INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT chk_print_subject CHECK (
    (order_id IS NOT NULL AND bill_id IS NULL) OR
    (order_id IS NULL AND bill_id IS NOT NULL)
  )
);

CREATE TABLE service_calls (
  id BIGSERIAL PRIMARY KEY,
  shop_id BIGINT NOT NULL REFERENCES shops(id),
  session_id BIGINT NOT NULL REFERENCES table_sessions(id),
  table_id BIGINT NOT NULL REFERENCES tables(id),
  customer_id BIGINT REFERENCES customers(id),
  reason VARCHAR(128),
  status VARCHAR(16) NOT NULL DEFAULT 'pending',
  operator_id BIGINT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  resolved_at TIMESTAMPTZ
);
CREATE INDEX idx_service_calls_pending ON service_calls(status, created_at);

-- 补充表：员工（设计 7.1 缺失，按 16.2 员工 JWT + 前端 Staff 契约补齐）
CREATE TABLE employees (
  id BIGSERIAL PRIMARY KEY,
  shop_id BIGINT NOT NULL REFERENCES shops(id),
  name VARCHAR(32) NOT NULL,
  employee_no VARCHAR(32) NOT NULL,
  password_hash VARCHAR(128) NOT NULL,
  role VARCHAR(16) NOT NULL,               -- owner / cashier / kitchen
  phone VARCHAR(20),
  status VARCHAR(16) NOT NULL DEFAULT 'active', -- active / disabled
  last_login_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(shop_id, employee_no)
);
