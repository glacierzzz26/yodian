-- 操作日志（设计 16.2：敏感操作记操作日志，追责与交接班 10.1）。
-- 由管理端 handler / 中间件写入：退款、核销、沽清、补录、改状态、收银入账等。
CREATE TABLE audit_logs (
  id BIGSERIAL PRIMARY KEY,
  shop_id BIGINT NOT NULL REFERENCES shops(id),
  operator_id BIGINT,
  role VARCHAR(16),
  action VARCHAR(64) NOT NULL,
  target_type VARCHAR(32),
  target_id BIGINT,
  detail JSONB,
  ip VARCHAR(64),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_logs_shop_created ON audit_logs(shop_id, created_at DESC);
CREATE INDEX idx_audit_logs_operator ON audit_logs(operator_id);
