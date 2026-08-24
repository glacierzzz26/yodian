-- 0002_seed down：清空业务数据（保留表结构）
TRUNCATE employees, service_calls, print_tasks, refunds, payments, order_items,
  orders, dishes, categories, per_head_charges, bills, table_sessions, customers,
  tables, table_areas, shops RESTART IDENTITY CASCADE;
