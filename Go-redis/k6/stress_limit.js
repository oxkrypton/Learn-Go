import { buildSummary, getBaseConfig, runSeckill } from './seckill_common.js';

const config = getBaseConfig(13);

export const options = {
  scenarios: {
    stress_limit: {
      executor: 'constant-arrival-rate',
      rate: Number(__ENV.RATE || '3000'),
      timeUnit: '1s',
      duration: __ENV.DURATION || '60s',
      preAllocatedVUs: Number(__ENV.PRE_ALLOCATED_VUS || '500'),
      maxVUs: Number(__ENV.MAX_VUS || '3000'),
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<1200'],
    http_req_failed: ['rate<0.1'],
    success_latency: ['p(95)<1200'],
  },
};

export default function () {
  runSeckill(config);
}

// k6 run --summary-export=reports/stress.json k6/stress_limit.js
