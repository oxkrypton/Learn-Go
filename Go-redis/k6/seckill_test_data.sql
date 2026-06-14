-- Active: 1761388216724@@127.0.0.1@3306
-- =========================
-- 1. 测试前查询
-- =========================

SELECT * FROM tb_voucher WHERE id = 13;
SELECT * FROM tb_seckill_voucher WHERE voucher_id = 13;
SELECT * FROM tb_voucher_order WHERE voucher_id = 13 ;

-- =========================
-- 3. 如果没有测试数据就创建
-- =========================

INSERT INTO tb_voucher
(id, shop_id, title, sub_title, rules, pay_value, actual_value, type, status, create_time, update_time)
SELECT
13, 1, 'k6秒杀测试券', '仅供压测使用', '测试专用', 100, 1000, 1, 1, NOW(), NOW()
FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM tb_voucher WHERE id = 13);

INSERT INTO tb_seckill_voucher
(voucher_id, stock, create_time, begin_time, end_time, update_time)
SELECT
13, 5000, NOW(), DATE_SUB(NOW(), INTERVAL 1 HOUR), DATE_ADD(NOW(), INTERVAL 7 DAY), NOW()
FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM tb_seckill_voucher WHERE voucher_id = 13);

-- =========================
-- 4. 重置本次压测数据
--    删除历史订单，并恢复秒杀库存
-- =========================

DELETE FROM tb_voucher_order
WHERE voucher_id = 13 AND user_id BETWEEN 1 AND 10000;

UPDATE tb_seckill_voucher
SET stock = 5000,
    begin_time = DATE_SUB(NOW(), INTERVAL 1 HOUR),
    end_time = DATE_ADD(NOW(), INTERVAL 7 DAY),
    update_time = NOW()
WHERE voucher_id = 13;

-- 再查一次，确认已准备好
SELECT * FROM tb_voucher WHERE id = 13;
SELECT * FROM tb_seckill_voucher WHERE voucher_id = 13;
SELECT COUNT(*) AS order_count
FROM tb_voucher_order
WHERE voucher_id = 13 AND user_id BETWEEN 1 AND 10000;


-- 测试完校验

SELECT * FROM tb_voucher WHERE id = 13;
SELECT * FROM tb_seckill_voucher WHERE voucher_id = 13;
SELECT COUNT(*) AS order_count, COUNT(DISTINCT user_id) AS user_count
FROM tb_voucher_order
WHERE voucher_id = 13 AND user_id BETWEEN 1 AND 10000;

SELECT * FROM tb_voucher_order
WHERE voucher_id = 13 AND user_id BETWEEN 1 AND 10000
ORDER BY user_id, id;
