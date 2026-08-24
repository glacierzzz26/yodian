-- 0002_seed: 悦点 基础数据，与 merchant-web/src/api/mock.ts 对齐（阶段 4 前端切换可对照）
-- 菜品 id 显式 1-18（与前端 Dish.id 一致）；金额单位「元」
-- 映射决策：前端 mock 的「下架 status:'off'」与「沽清 soldOut」在后端 7.1 只有 is_sold_out 一个字段，
--   MVP 统一映射 is_sold_out=true（不可售、顾客端不可见），两种 tag 的区分在阶段 4 联调时收口。

INSERT INTO shops (id, name, address) VALUES (1, '悦点', '示例路 88 号');

INSERT INTO categories (id, shop_id, name, sort) VALUES
  (1, 1, '热菜', 1),
  (2, 1, '凉菜', 2),
  (3, 1, '汤羹', 3),
  (4, 1, '主食', 4),
  (5, 1, '酒水', 5),
  (6, 1, '自助', 6);

INSERT INTO dishes (id, shop_id, category_id, name, price, specs, is_sold_out, sort) VALUES
  (1,  1, 1, '宫保鸡丁',   46.00,  NULL, false, 1),
  (2,  1, 1, '水煮牛肉',   68.00,  NULL, false, 2),
  (3,  1, 1, '剁椒鱼头',   98.00,  NULL, false, 3),
  (4,  1, 1, '红烧肉',     78.00,  NULL, false, 4),
  (5,  1, 1, '清蒸鲈鱼',   128.00, NULL, false, 5),
  (6,  1, 1, '佛跳墙',     298.00, NULL, true,  6),
  (7,  1, 2, '口水鸡',     38.00,  NULL, false, 7),
  (8,  1, 2, '酸辣土豆丝', 18.00,  NULL, false, 8),
  (9,  1, 2, '夫妻肺片',   42.00,  NULL, true,  9),   -- mock status:'off'（下架）
  (10, 1, 1, '烤羊排',     128.00, NULL, false, 10),
  (11, 1, 3, '青菜豆腐汤', 22.00,  NULL, false, 11),
  (12, 1, 3, '西红柿蛋汤', 16.00,  NULL, false, 12),
  (13, 1, 4, '米饭',       3.00,   NULL, false, 13),
  (14, 1, 4, '手工水饺',   22.00,  NULL, false, 14),
  (15, 1, 5, '青岛啤酒',   10.00,  NULL, false, 15),
  (16, 1, 5, '自制酸梅汤', 12.00,  NULL, false, 16),
  (17, 1, 6, '自助火锅',   58.00,  NULL, false, 17),  -- per_head（按人）
  (18, 1, 6, '自助烧烤',   68.00,  NULL, false, 18);  -- per_head（按人）

INSERT INTO table_areas (id, shop_id, name, sort_order, default_pay_mode) VALUES
  (1, 1, '一楼大厅', 1, 'prepay'),
  (2, 1, '包间 · 二楼', 2, 'postpay'),
  (3, 1, '露台', 3, 'prepay');

INSERT INTO tables (shop_id, area_id, table_no, seats, status, pay_mode, sort_order) VALUES
  (1, 1, 'A01', 4, 'empty', 'prepay',  1), (1, 1, 'A02', 4, 'empty', 'prepay',  2),
  (1, 1, 'A03', 4, 'empty', 'prepay',  3), (1, 1, 'A04', 4, 'empty', 'prepay',  4),
  (1, 1, 'A05', 4, 'empty', 'prepay',  5), (1, 1, 'A06', 4, 'empty', 'prepay',  6),
  (1, 1, 'A07', 4, 'empty', 'prepay',  7), (1, 1, 'A08', 4, 'empty', 'prepay',  8),
  (1, 1, 'A09', 4, 'empty', 'prepay',  9), (1, 1, 'A10', 4, 'empty', 'prepay', 10),
  (1, 2, 'B01', 10, 'empty', 'postpay', 1), (1, 2, 'B02', 10, 'empty', 'postpay', 2),
  (1, 2, 'B03', 10, 'empty', 'postpay', 3), (1, 2, 'B04', 10, 'empty', 'postpay', 4),
  (1, 2, 'B05', 10, 'empty', 'postpay', 5), (1, 2, 'B06', 10, 'empty', 'postpay', 6),
  (1, 2, 'B07', 10, 'empty', 'postpay', 7), (1, 2, 'B08', 10, 'empty', 'postpay', 8),
  (1, 3, 'C01', 4, 'empty', 'prepay', 1), (1, 3, 'C02', 4, 'empty', 'prepay', 2),
  (1, 3, 'C03', 4, 'empty', 'prepay', 3), (1, 3, 'C04', 4, 'empty', 'prepay', 4),
  (1, 3, 'C05', 4, 'empty', 'prepay', 5), (1, 3, 'C06', 4, 'empty', 'prepay', 6);

INSERT INTO per_head_charges (shop_id, name, price, is_default, is_required, sort) VALUES
  (1, '餐位费', 5.00, true, true,  1),
  (1, '湿巾',   1.00, true, false, 2);

-- 员工密码统一 123456（bcrypt cost=10），上线后首登强制修改（阶段 2.4 登录时校验）
INSERT INTO employees (shop_id, name, employee_no, password_hash, role, phone, status) VALUES
  (1, '张店长', '1001', '$2a$10$vCi3uoTRRLQ.Dn98a4f91ObC6u.IpUiOugpot/inSmlfkB4HTQ8ZO', 'owner',   '13800000001', 'active'),
  (1, '李收银', '1002', '$2a$10$vCi3uoTRRLQ.Dn98a4f91ObC6u.IpUiOugpot/inSmlfkB4HTQ8ZO', 'cashier', '13800000002', 'active'),
  (1, '王厨师', '1003', '$2a$10$vCi3uoTRRLQ.Dn98a4f91ObC6u.IpUiOugpot/inSmlfkB4HTQ8ZO', 'kitchen', '13800000003', 'active');
