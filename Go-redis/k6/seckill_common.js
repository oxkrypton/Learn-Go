import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';

export const successOrders = new Counter('successful_orders');
export const failedOrders = new Counter('failed_orders');
export const failStockNotEnough = new Counter('fail_stock_not_enough');
export const failDuplicateRequest = new Counter('fail_duplicate_request');
export const failNotLogin = new Counter('fail_not_login');
export const failServerError = new Counter('fail_server_error');
export const failUnknown = new Counter('fail_unknown');
export const successLatency = new Trend('success_latency', true);

export function getBaseConfig(defaultVoucherId) {
  return {
    baseUrl: __ENV.BASE_URL || 'http://localhost:8080',
    voucherId: 13,
    sleepMs: Number(__ENV.SLEEP_MS || '0'),
  };
}

export function buildSeckillRequest(config) {
  const userId = (__ITER % 10000) + 1;
  return {
    url: `${config.baseUrl}/voucher/seckill/${config.voucherId}`,
    params: {
      headers: {
        'Content-Type': 'application/json',
        'X-User-Id': `${userId}`,
      },
    },
  };
}

function classifyFailure(message, statusCode) {
  const msg = (message || '').toLowerCase();
  if (msg.includes('stock is not enough')) {
    failStockNotEnough.add(1);
    return;
  }
  if (msg.includes('duplicate request') || msg.includes('only order once')) {
    failDuplicateRequest.add(1);
    return;
  }
  if (msg.includes('not login') || statusCode === 401) {
    failNotLogin.add(1);
    return;
  }
  if (statusCode >= 500) {
    failServerError.add(1);
    return;
  }
  failUnknown.add(1);
}

export function runSeckill(config) {
  const req = buildSeckillRequest(config);
  const res = http.post(req.url, null, req.params);

  check(res, {
    'status is 200': (r) => r.status === 200,
  });

  let isBusinessSuccess = false;
  let failureMessage = '';

  try {
    const body = JSON.parse(res.body);
    isBusinessSuccess = res.status === 200 && body.success === true && !!body.data;
    failureMessage = body && body.errMsg ? body.errMsg : '';
  } catch (e) {
    failureMessage = 'invalid response body';
  }

  if (isBusinessSuccess) {
    successOrders.add(1);
    successLatency.add(res.timings.duration);
  } else {
    failedOrders.add(1);
    classifyFailure(failureMessage, res.status);
  }

  if (config.sleepMs > 0) {
    sleep(config.sleepMs / 1000);
  }
}

function getCount(data, metricName) {
  if (!data.metrics[metricName] || !data.metrics[metricName].values) {
    return 0;
  }
  return data.metrics[metricName].values.count || 0;
}

export function buildSummary(name) {
  return function handleSummary(data) {
    const success = getCount(data, 'successful_orders');
    const failed = getCount(data, 'failed_orders');

    console.log(`\n===== ${name} =====`);
    console.log(`成功下单: ${success}`);
    console.log(`下单失败: ${failed}`);
    console.log(`库存不足: ${getCount(data, 'fail_stock_not_enough')}`);
    console.log(`重复下单: ${getCount(data, 'fail_duplicate_request')}`);
    console.log(`未登录: ${getCount(data, 'fail_not_login')}`);
    console.log(`服务异常: ${getCount(data, 'fail_server_error')}`);
    console.log(`未知失败: ${getCount(data, 'fail_unknown')}`);
    console.log('========================\n');
    return {};
  };
}
