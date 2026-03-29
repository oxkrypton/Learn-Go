import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter } from 'k6/metrics';

// ========== 配置 ==========
const BASE_URL = 'http://localhost:8080';
const VOUCHER_ID = __ENV.VOUCHER_ID || '1';  // 通过 -e VOUCHER_ID=xxx 传入

// 自定义指标
const successOrders = new Counter('successful_orders');
const failedOrders  = new Counter('failed_orders');

// ========== 并发配置 ==========
export const options = {
  scenarios: {
    spike: {
      executor: 'shared-iterations',
      vus: 1000,
      iterations: 1000,
      maxDuration: '30s',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500'],   // 95% 请求耗时 < 500ms
    http_req_failed:   ['rate<0.8'],    // 允许大部分请求因库存不足而"失败"
  },
};

// ========== 测试函数 ==========
export default function () {
  // 每个 VU 使用自己的 ID 作为 userId，模拟不同用户
  const userId = __VU;

  const url = `${BASE_URL}/voucher/seckill/${VOUCHER_ID}`;
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-User-Id': `${userId}`,  // 配合方案 A 的 Header
    },
  };

  const res = http.post(url, null, params);

  // 检查响应
  const isSuccess = check(res, {
    'status is 200': (r) => r.status === 200,
  });

  // 解析 body 判断是否真正下单成功
  try {
    const body = JSON.parse(res.body);
    if (res.status === 200 && body.data) {
      successOrders.add(1);
    } else {
      failedOrders.add(1);
    }
  } catch (e) {
    failedOrders.add(1);
  }

  sleep(0.1);  // 微小间隔，避免完全同步
}

// ========== 结果汇总 ==========
export function handleSummary(data) {
  const success = data.metrics.successful_orders
    ? data.metrics.successful_orders.values.count
    : 0;
  const failed  = data.metrics.failed_orders
    ? data.metrics.failed_orders.values.count
    : 0;

  console.log('\n===== 秒杀压测结果 =====');
  console.log(`成功下单: ${success}`);
  console.log(`下单失败: ${failed}`);
  console.log(`总请求数: ${success + failed}`);
  console.log('========================\n');

  return {};
}
