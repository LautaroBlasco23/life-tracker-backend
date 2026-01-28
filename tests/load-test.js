import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Counter } from 'k6/metrics';

const errorRate = new Rate('errors');
const successfulFlows = new Counter('successful_flows');

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
  stages: [
    { duration: '30s', target: 10 },
    { duration: '1m', target: 30 },
    { duration: '2m', target: 50 },
    { duration: '1m', target: 20 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<800', 'p(99)<1500'],
    http_req_failed: ['rate<0.15'],
    errors: ['rate<0.2'],
    successful_flows: ['count>50'],
  },
};

function generateEmail() {
  return `loadtest_${Date.now()}_${Math.floor(Math.random() * 100000)}@test.com`;
}

function registerAndLogin() {
  const email = generateEmail();
  const password = 'Test123456!';

  const registerPayload = {
    email,
    password,
    firstName: 'Load',
    lastName: 'Test',
  };

  const registerRes = http.post(
    `${BASE_URL}/api/auth/register`,
    JSON.stringify(registerPayload),
    {
      headers: {
        'Content-Type': 'application/json',
        'Origin': 'http://localhost:3000',
      }
    }
  );

  const registerOk = check(registerRes, {
    'registration successful': (r) => r.status === 201,
    'has tokens in response': (r) => {
      try {
        const json = r.json();
        return json && json.data && json.data.tokens && json.data.tokens.accessToken;
      } catch (e) {
        return false;
      }
    },
  });

  if (!registerOk) {
    errorRate.add(1);
    if (__ENV.DEBUG) {
      console.error(`Registration failed: ${registerRes.status} - ${registerRes.body}`);
    }
    return null;
  }

  const tokens = registerRes.json('data.tokens');
  return {
    accessToken: tokens.accessToken,
    refreshToken: tokens.refreshToken,
    email,
    password,
  };
}

function getAuthHeaders(token) {
  return {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
    'Origin': 'http://localhost:3000',
  };
}

function testActivityFlow(token) {
  const activityPayload = {
    title: `Exercise ${Date.now()}`,
    description: 'Load test activity',
    frequency: 'daily',
    dayTime: 'morning',
    completionAmount: 1,
  };

  const createRes = http.post(
    `${BASE_URL}/api/activities`,
    JSON.stringify(activityPayload),
    { headers: getAuthHeaders(token) }
  );

  const createOk = check(createRes, {
    'activity created': (r) => r.status === 201,
    'activity has id': (r) => {
      try {
        return r.json('data.id') !== undefined;
      } catch (e) {
        return false;
      }
    },
  });

  if (!createOk) {
    errorRate.add(1);
    if (__ENV.DEBUG) {
      console.error(`Activity creation failed: ${createRes.status} - ${createRes.body}`);
    }
    return false;
  }

  const activityId = createRes.json('data.id');
  sleep(0.3);

  const getRes = http.get(
    `${BASE_URL}/api/activities/${activityId}`,
    { headers: getAuthHeaders(token) }
  );

  const getOk = check(getRes, {
    'activity retrieved': (r) => r.status === 200,
  });

  if (!getOk && __ENV.DEBUG) {
    console.error(`Get activity failed: ${getRes.status} - ${getRes.body}`);
  }

  sleep(0.3);

  const recordPayload = {
    notes: 'Load test completion',
  };

  const recordRes = http.post(
    `${BASE_URL}/api/activities/${activityId}/record`,
    JSON.stringify(recordPayload),
    { headers: getAuthHeaders(token) }
  );

  const recordOk = check(recordRes, {
    'activity recorded': (r) => r.status === 201,
  });

  if (!recordOk) {
    errorRate.add(1);
    if (__ENV.DEBUG) {
      console.error(`Activity record failed: ${recordRes.status} - ${recordRes.body}`);
    }
  }

  return createOk && getOk;
}

function testFinanceFlow(token) {
  const categoriesRes = http.get(`${BASE_URL}/api/finances/categories`, {
    headers: getAuthHeaders(token),
  });

  const categoriesOk = check(categoriesRes, {
    'categories retrieved': (r) => r.status === 200,
    'has categories array': (r) => {
      try {
        const data = r.json('data');
        return Array.isArray(data) && data.length > 0;
      } catch (e) {
        return false;
      }
    },
  });

  if (!categoriesOk) {
    errorRate.add(1);
    if (__ENV.DEBUG) {
      console.error(`Categories failed: ${categoriesRes.status} - ${categoriesRes.body}`);
    }
    return false;
  }

  const categories = categoriesRes.json('data');

  const variableCategory = categories.find(c =>
    (c.applicableToFreq === 'variable' || c.applicableToFreq === 'both') &&
    c.type === 'outcome'
  );

  if (!variableCategory) {
    if (__ENV.DEBUG) {
      console.error('No suitable category found');
      console.error('Available categories:', JSON.stringify(categories.map(c => ({
        id: c.id,
        name: c.name,
        type: c.type,
        freq: c.applicableToFreq
      }))));
    }
    errorRate.add(1);
    return false;
  }

  sleep(0.3);

  const now = new Date();
  const transactionPayload = {
    type: 'outcome',
    frequency: 'variable',
    amount: parseFloat((Math.random() * 100 + 10).toFixed(2)),
    categoryId: variableCategory.id,
    description: 'Load test transaction',
    date: now.toISOString(),
  };

  const createRes = http.post(
    `${BASE_URL}/api/finances/transactions`,
    JSON.stringify(transactionPayload),
    { headers: getAuthHeaders(token) }
  );

  const createOk = check(createRes, {
    'transaction created': (r) => r.status === 201,
    'transaction has id': (r) => {
      try {
        return r.json('data.id') !== undefined;
      } catch (e) {
        return false;
      }
    },
  });

  if (!createOk) {
    errorRate.add(1);
    if (__ENV.DEBUG) {
      console.error(`Transaction creation failed: ${createRes.status}`);
      console.error(`Payload: ${JSON.stringify(transactionPayload)}`);
      console.error(`Response: ${createRes.body}`);
    }
    return false;
  }

  sleep(0.3);

  const transactionId = createRes.json('data.id');
  const getRes = http.get(
    `${BASE_URL}/api/finances/transactions/${transactionId}`,
    { headers: getAuthHeaders(token) }
  );

  check(getRes, {
    'transaction retrieved': (r) => r.status === 200,
  });

  sleep(0.3);

  const startDate = new Date(now.getFullYear(), now.getMonth(), 1);
  const endDate = new Date(now.getFullYear(), now.getMonth() + 1, 0);

  const summaryRes = http.get(
    `${BASE_URL}/api/finances/summary?start_date=${formatDate(startDate)}&end_date=${formatDate(endDate)}`,
    { headers: getAuthHeaders(token) }
  );

  check(summaryRes, {
    'summary retrieved': (r) => r.status === 200,
  });

  return createOk;
}

function formatDate(date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

function testHealthCheck() {
  const res = http.get(`${BASE_URL}/health`);

  const ok = check(res, {
    'health check responds': (r) => r.status === 200,
    'postgres healthy': (r) => {
      try {
        return r.json('databases.postgresql') === 'ok';
      } catch (e) {
        return false;
      }
    },
    'mongodb healthy': (r) => {
      try {
        return r.json('databases.mongodb') === 'ok';
      } catch (e) {
        return false;
      }
    },
  });

  if (!ok && __ENV.DEBUG) {
    console.error(`Health check failed: ${res.status} - ${res.body}`);
  }

  return ok;
}

export default function () {
  if (!testHealthCheck()) {
    sleep(2);
    return;
  }

  const auth = registerAndLogin();
  if (!auth) {
    sleep(1);
    return;
  }

  sleep(0.5);

  const scenario = Math.random();
  let success = false;

  if (scenario < 0.5) {
    success = testActivityFlow(auth.accessToken);
  } else {
    success = testFinanceFlow(auth.accessToken);
  }

  if (success) {
    successfulFlows.add(1);
  }

  sleep(1);
}

export function handleSummary(data) {
  const metrics = data.metrics;

  const passed = Object.values(data.root_group.checks)
    .reduce((sum, check) => sum + check.passes, 0);
  const failed = Object.values(data.root_group.checks)
    .reduce((sum, check) => sum + check.fails, 0);
  const total = passed + failed;
  const passRate = total > 0 ? ((passed / total) * 100).toFixed(2) : '0.00';

  const getValue = (obj, path, defaultValue = 0) => {
    try {
      return obj && obj[path] !== undefined ? obj[path] : defaultValue;
    } catch {
      return defaultValue;
    }
  };

  console.log('\n' + '='.repeat(60));
  console.log('LOAD TEST SUMMARY');
  console.log('='.repeat(60));
  console.log(`Total Requests: ${getValue(metrics.http_reqs?.values, 'count')}`);
  console.log(`Failed Requests: ${(getValue(metrics.http_req_failed?.values, 'rate') * 100).toFixed(2)}%`);
  console.log(`Check Pass Rate: ${passRate}%`);
  console.log(`Successful Flows: ${getValue(metrics.successful_flows?.values, 'count')}`);
  console.log(`Error Rate: ${(getValue(metrics.errors?.values, 'rate') * 100).toFixed(2)}%`);
  console.log('\nResponse Times:');
  console.log(`  Average: ${getValue(metrics.http_req_duration?.values, 'avg').toFixed(2)}ms`);
  console.log(`  Median: ${getValue(metrics.http_req_duration?.values, 'med').toFixed(2)}ms`);
  console.log(`  p95: ${getValue(metrics.http_req_duration?.values, 'p(95)').toFixed(2)}ms`);
  console.log(`  p99: ${getValue(metrics.http_req_duration?.values, 'p(99)').toFixed(2)}ms`);
  console.log(`  Max: ${getValue(metrics.http_req_duration?.values, 'max').toFixed(2)}ms`);
  console.log('='.repeat(60) + '\n');

  return {
    'stdout': '',
    'summary.json': JSON.stringify(data, null, 2),
  };
}
