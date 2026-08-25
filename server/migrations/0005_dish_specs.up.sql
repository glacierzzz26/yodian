-- 阶段 3.4：为种子菜品补充「规格」数据（specs JSONB），供顾客端规格弹窗演示。
-- 结构：[{"name":规格组, "options":[{"name":选项, "price_delta":分}]}]
-- 后端不计规格加价（specs 随小票展示，设计 810），price_delta 为保留字段。
UPDATE dishes SET specs = '[{"name":"辣度","options":[{"name":"微辣","price_delta":0},{"name":"中辣","price_delta":0},{"name":"特辣","price_delta":0}]}]'::jsonb WHERE id = 8 AND specs IS NULL;
UPDATE dishes SET specs = '[{"name":"份量","options":[{"name":"小份","price_delta":0},{"name":"大份","price_delta":0}]}]'::jsonb WHERE id = 13 AND specs IS NULL;
