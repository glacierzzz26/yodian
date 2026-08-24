-- 设计缺口修正：7.2 枚举含 partially_refunded（18 字符），orders.status VARCHAR(16) 存不下。
-- 扩大列宽，保证状态机全枚举可入库。
ALTER TABLE orders ALTER COLUMN status TYPE VARCHAR(32);
