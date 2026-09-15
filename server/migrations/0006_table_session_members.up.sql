-- 0006: 会话参与人表（P0-1 会话归属校验 + P1-3 拼桌互见）
--
-- 背景：顾客端点 /sessions/:sid/* 原先只按 shop_id+id 查会话，不校验归属 → 任意顾客 JWT
-- 可枚举 sid 读他人整桌订单/账单、改人数、抢结账锁、冒名呼叫（IDOR）。顾客 JWT 的 SID 是
-- shop_id（16.2）而非会话 id，故「消费 SID」不能修复，正确修法是会话归属断言。
--
-- 归属 = 「该顾客是会话成员」。成员来源两条：扫码 join（POST /tables/:tid/join）、下单。
-- 参与表同时是 P1-3「拼桌互见」的前提：第二位顾客扫码即得成员身份，无须先下单即可看本桌单。
CREATE TABLE table_session_members (
  id BIGSERIAL PRIMARY KEY,
  shop_id BIGINT NOT NULL REFERENCES shops(id),
  session_id BIGINT NOT NULL REFERENCES table_sessions(id),
  customer_id BIGINT NOT NULL REFERENCES customers(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(session_id, customer_id)
);
CREATE INDEX idx_tsm_customer ON table_session_members(shop_id, customer_id);

-- 回填：已下过单的顾客即视为该会话成员（保证存量会话的合法顾客不被新校验误拒）
INSERT INTO table_session_members(shop_id, session_id, customer_id)
  SELECT DISTINCT o.shop_id, o.session_id, o.customer_id
  FROM orders o
  WHERE o.customer_id IS NOT NULL AND o.session_id IS NOT NULL;
