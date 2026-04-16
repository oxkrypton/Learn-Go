import { buildSummary, getBaseConfig, runSeckill } from './seckill_common.js';

const config = getBaseConfig(13);

export const options = {
  scenarios: {
    stress_limit: {
      executor: 'shared-iterations',
      vus: Number(__ENV.VUS || '1500'),
      iterations: Number(__ENV.ITERATIONS || '5000'),
      maxDuration: __ENV.MAX_DURATION || '120s',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<1200'],
    http_req_failed: ['rate<0.995'],
    success_latency: ['p(95)<1200'],
  },
};

export default function () {
  runSeckill(config);
}

//k6 run --summary-export=reports/baseline.json k6/stress_limit.js
