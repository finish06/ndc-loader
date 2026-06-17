// rx-dag (ndc-loader) k6 test harness — full API surface
//
// Covers all query endpoints, openFDA compat, admin, and public endpoints.
//
// Usage:
//   k6 run tests/k6/staging.js                          # default (smoke + load)
//   k6 run tests/k6/staging.js --env SCENARIO=smoke     # smoke only
//   k6 run tests/k6/staging.js --env SCENARIO=load      # load only
//   k6 run tests/k6/staging.js --env SCENARIO=spike     # spike only
//   k6 run tests/k6/staging.js --env SCENARIO=soak      # 5-minute soak

import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

const BASE_URL = __ENV.BASE_URL || 'http://192.168.1.145:8081';
const API_KEY  = __ENV.API_KEY;
const SCENARIO = __ENV.SCENARIO || 'all';

if (!API_KEY) {
  throw new Error('API_KEY env var required — pass it with `--env API_KEY=...` (no default; never hardcode a key in source)');
}

const authHeaders = { 'X-API-Key': API_KEY };

// ---------------------------------------------------------------------------
// Custom metrics
// ---------------------------------------------------------------------------

const errorRate          = new Rate('error_rate');
const ndcLookupDuration  = new Trend('ndc_lookup_p95', true);
const searchDuration     = new Trend('search_p95', true);
const openfdaDuration    = new Trend('openfda_p95', true);
const packagesDuration   = new Trend('packages_p95', true);
const endpointHits       = new Counter('endpoint_hits');

// ---------------------------------------------------------------------------
// Scenarios
// ---------------------------------------------------------------------------

function buildScenarios() {
  const scenarios = {};

  if (SCENARIO === 'all' || SCENARIO === 'smoke') {
    scenarios.smoke = {
      executor: 'shared-iterations',
      vus: 1,
      iterations: 1,
      exec: 'smokeTest',
      startTime: '0s',
    };
  }

  if (SCENARIO === 'all' || SCENARIO === 'load') {
    scenarios.load = {
      executor: 'constant-vus',
      vus: 10,
      duration: '30s',
      exec: 'loadTest',
      startTime: SCENARIO === 'all' ? '10s' : '0s',
    };
  }

  if (SCENARIO === 'all' || SCENARIO === 'spike') {
    scenarios.spike = {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 25 },
        { duration: '15s', target: 25 },
        { duration: '5s', target: 0 },
      ],
      exec: 'spikeTest',
      startTime: SCENARIO === 'all' ? '45s' : '0s',
    };
  }

  if (SCENARIO === 'soak') {
    scenarios.soak = {
      executor: 'constant-vus',
      vus: 5,
      duration: '5m',
      exec: 'loadTest',
      startTime: '0s',
    };
  }

  return scenarios;
}

export const options = {
  scenarios: buildScenarios(),
  thresholds: {
    http_req_duration: ['p(95)<500'],     // overall p95 < 500ms
    error_rate:        ['rate<0.05'],      // < 5% errors
    ndc_lookup_p95:    ['p(95)<50'],       // NDC lookup p95 < 50ms
    search_p95:        ['p(95)<200'],      // search p95 < 200ms
    openfda_p95:       ['p(95)<200'],      // openFDA p95 < 200ms
  },
};

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function getJSON(url, hdrs) {
  const res = http.get(url, { headers: hdrs || {} });
  endpointHits.add(1);
  return res;
}

function postJSON(url, body, hdrs) {
  const res = http.post(url, JSON.stringify(body), {
    headers: Object.assign({ 'Content-Type': 'application/json' }, hdrs || {}),
  });
  endpointHits.add(1);
  return res;
}

function safeParseJSON(body) {
  try { return JSON.parse(body); }
  catch { return null; }
}

// ---------------------------------------------------------------------------
// Scenario: Smoke Test — verify every endpoint works
// ---------------------------------------------------------------------------

export function smokeTest() {
  // ── Public endpoints ──────────────────────────────────────────────────

  group('public endpoints', () => {
    let res = getJSON(`${BASE_URL}/health`);
    check(res, {
      'health: 200': (r) => r.status === 200,
      'health: status field': (r) => {
        const b = safeParseJSON(r.body);
        return b && (b.status === 'ok' || b.status === 'degraded');
      },
      'health: has dependencies': (r) => {
        const b = safeParseJSON(r.body);
        return b && Array.isArray(b.dependencies) && b.dependencies.length > 0;
      },
      'health: has uptime': (r) => safeParseJSON(r.body)?.uptime !== undefined,
      'health: has version': (r) => safeParseJSON(r.body)?.version !== undefined,
    });

    res = getJSON(`${BASE_URL}/version`);
    check(res, {
      'version: 200': (r) => r.status === 200,
      'version: has all fields': (r) => {
        const b = safeParseJSON(r.body);
        return b && b.version && b.git_commit && b.git_branch && b.go_version && b.os && b.arch;
      },
    });

    res = getJSON(`${BASE_URL}/metrics`);
    check(res, {
      'metrics: 200': (r) => r.status === 200,
      'metrics: has ndc_loader counters': (r) => r.body.includes('ndc_loader_'),
    });

    res = getJSON(`${BASE_URL}/swagger/index.html`);
    check(res, {
      'swagger: 200': (r) => r.status === 200,
    });
  });

  // ── Auth rejection ────────────────────────────────────────────────────

  group('auth', () => {
    const res = getJSON(`${BASE_URL}/api/ndc/stats`);
    check(res, {
      'auth: 401 without key': (r) => r.status === 401,
      'auth: error body': (r) => {
        const b = safeParseJSON(r.body);
        return b && b.error === 'unauthorized';
      },
    });
  });

  // ── NDC Lookup ────────────────────────────────────────────────────────

  group('NDC lookup', () => {
    // 2-segment product lookup
    let res = getJSON(`${BASE_URL}/api/ndc/0591-0405`, authHeaders);
    check(res, {
      'ndc lookup: 200': (r) => r.status === 200,
      'ndc lookup: has product_ndc': (r) => safeParseJSON(r.body)?.product_ndc === '0591-0405',
      'ndc lookup: has packages': (r) => safeParseJSON(r.body)?.packages?.length > 0,
      'ndc lookup: has pharm_classes_structured': (r) => safeParseJSON(r.body)?.pharm_classes_structured !== undefined,
    });

    // 3-segment package lookup
    res = getJSON(`${BASE_URL}/api/ndc/0591-0405-01`, authHeaders);
    check(res, {
      'ndc package: 200': (r) => r.status === 200,
      'ndc package: matched_package set': (r) => safeParseJSON(r.body)?.matched_package !== null,
    });

    // Invalid NDC
    res = getJSON(`${BASE_URL}/api/ndc/abc`, authHeaders);
    check(res, {
      'ndc invalid: 400': (r) => r.status === 400,
    });

    // Not found
    res = getJSON(`${BASE_URL}/api/ndc/9999-9999`, authHeaders);
    check(res, {
      'ndc not found: 404': (r) => r.status === 404,
    });
  });

  // ── Search ────────────────────────────────────────────────────────────

  group('search', () => {
    let res = getJSON(`${BASE_URL}/api/ndc/search?q=metformin&limit=5`, authHeaders);
    check(res, {
      'search: 200': (r) => r.status === 200,
      'search: has results': (r) => safeParseJSON(r.body)?.results?.length > 0,
      'search: has total': (r) => safeParseJSON(r.body)?.total > 0,
      'search: limit respected': (r) => safeParseJSON(r.body)?.results?.length <= 5,
    });

    // Prefix matching
    res = getJSON(`${BASE_URL}/api/ndc/search?q=metfor&limit=3`, authHeaders);
    check(res, {
      'search prefix: returns results': (r) => safeParseJSON(r.body)?.total > 0,
    });

    // Missing query
    res = getJSON(`${BASE_URL}/api/ndc/search`, authHeaders);
    check(res, {
      'search no query: 400': (r) => r.status === 400,
    });
  });

  // ── Packages ──────────────────────────────────────────────────────────

  group('packages', () => {
    const res = getJSON(`${BASE_URL}/api/ndc/0591-0405/packages`, authHeaders);
    check(res, {
      'packages: 200': (r) => r.status === 200,
      'packages: has array': (r) => Array.isArray(safeParseJSON(r.body)?.packages),
      'packages: non-empty': (r) => safeParseJSON(r.body)?.packages?.length > 0,
    });
  });

  // ── Stats ─────────────────────────────────────────────────────────────

  group('stats', () => {
    const res = getJSON(`${BASE_URL}/api/ndc/stats`, authHeaders);
    check(res, {
      'stats: 200': (r) => r.status === 200,
      'stats: products > 100k': (r) => safeParseJSON(r.body)?.products > 100000,
      'stats: packages > 200k': (r) => safeParseJSON(r.body)?.packages > 200000,
      'stats: applications > 20k': (r) => safeParseJSON(r.body)?.applications > 20000,
    });
  });

  // ── openFDA compat ────────────────────────────────────────────────────

  group('openFDA compat', () => {
    let res = getJSON(`${BASE_URL}/api/openfda/ndc.json?search=brand_name:lisinopril&limit=3`, authHeaders);
    check(res, {
      'openfda: 200': (r) => r.status === 200,
      'openfda: has meta': (r) => safeParseJSON(r.body)?.meta?.results?.total > 0,
      'openfda: has results': (r) => safeParseJSON(r.body)?.results?.length > 0,
      'openfda: active_ingredients array': (r) => Array.isArray(safeParseJSON(r.body)?.results?.[0]?.active_ingredients),
      'openfda: packaging array': (r) => Array.isArray(safeParseJSON(r.body)?.results?.[0]?.packaging),
      'openfda: route array': (r) => Array.isArray(safeParseJSON(r.body)?.results?.[0]?.route),
      'openfda: pharm_class array': (r) => Array.isArray(safeParseJSON(r.body)?.results?.[0]?.pharm_class),
      'openfda: openfda nested': (r) => safeParseJSON(r.body)?.results?.[0]?.openfda?.manufacturer_name !== undefined,
    });

    // Product NDC exact lookup
    res = getJSON(`${BASE_URL}/api/openfda/ndc.json?search=product_ndc:"0591-0405"&limit=1`, authHeaders);
    check(res, {
      'openfda ndc: 200': (r) => r.status === 200,
      'openfda ndc: exact match': (r) => safeParseJSON(r.body)?.results?.[0]?.product_ndc === '0591-0405',
    });

    // Not found
    res = getJSON(`${BASE_URL}/api/openfda/ndc.json?search=product_ndc:"9999-9999"&limit=1`, authHeaders);
    check(res, {
      'openfda 404: NOT_FOUND': (r) => r.status === 404 && safeParseJSON(r.body)?.error?.code === 'NOT_FOUND',
    });

    // Missing search
    res = getJSON(`${BASE_URL}/api/openfda/ndc.json`, authHeaders);
    check(res, {
      'openfda no search: 400': (r) => r.status === 400,
    });
  });

  // ── Admin ─────────────────────────────────────────────────────────────

  group('admin', () => {
    // Check load status (non-existent ID)
    const res = getJSON(`${BASE_URL}/api/admin/load/nonexistent-id`, authHeaders);
    check(res, {
      'admin load status: 404': (r) => r.status === 404,
    });
  });
}

// ---------------------------------------------------------------------------
// Scenario: Load Test — sustained traffic across key endpoints
// ---------------------------------------------------------------------------

const loadEndpoints = [
  // Fast endpoints (indexed lookups)
  { path: '/api/ndc/0591-0405', trend: ndcLookupDuration, weight: 3 },
  { path: '/api/ndc/0002-1433', trend: ndcLookupDuration, weight: 3 },
  { path: '/api/ndc/0078-0401', trend: ndcLookupDuration, weight: 2 },
  // Search (tsvector)
  { path: '/api/ndc/search?q=metformin&limit=10', trend: searchDuration, weight: 3 },
  { path: '/api/ndc/search?q=lisinopril&limit=10', trend: searchDuration, weight: 2 },
  { path: '/api/ndc/search?q=atorvastatin&limit=10', trend: searchDuration, weight: 2 },
  // openFDA compat
  { path: '/api/openfda/ndc.json?search=brand_name:metformin&limit=5', trend: openfdaDuration, weight: 2 },
  { path: '/api/openfda/ndc.json?search=brand_name:lisinopril&limit=5', trend: openfdaDuration, weight: 2 },
  // Packages
  { path: '/api/ndc/0591-0405/packages', trend: packagesDuration, weight: 1 },
  // Stats
  { path: '/api/ndc/stats', trend: null, weight: 1 },
  // Public
  { path: '/health', trend: null, weight: 1, public: true },
  { path: '/version', trend: null, weight: 1, public: true },
];

const weightedPool = [];
for (const ep of loadEndpoints) {
  for (let i = 0; i < ep.weight; i++) {
    weightedPool.push(ep);
  }
}

export function loadTest() {
  const ep = weightedPool[Math.floor(Math.random() * weightedPool.length)];
  const hdrs = ep.public ? {} : authHeaders;

  const res = getJSON(`${BASE_URL}${ep.path}`, hdrs);
  if (ep.trend) ep.trend.add(res.timings.duration);

  const ok = check(res, {
    'load: status 2xx': (r) => r.status >= 200 && r.status < 300,
  });
  errorRate.add(!ok);

  sleep(0.05 + Math.random() * 0.1);
}

// ---------------------------------------------------------------------------
// Scenario: Spike Test — burst across all endpoints
// ---------------------------------------------------------------------------

const spikeEndpoints = [
  { path: '/api/ndc/0591-0405', auth: true },
  { path: '/api/ndc/0002-1433', auth: true },
  { path: '/api/ndc/search?q=aspirin&limit=5', auth: true },
  { path: '/api/ndc/search?q=warfarin&limit=5', auth: true },
  { path: '/api/openfda/ndc.json?search=metformin&limit=3', auth: true },
  { path: '/api/ndc/0591-0405/packages', auth: true },
  { path: '/api/ndc/stats', auth: true },
  { path: '/health', auth: false },
  { path: '/version', auth: false },
  { path: '/metrics', auth: false },
];

export function spikeTest() {
  const ep = spikeEndpoints[Math.floor(Math.random() * spikeEndpoints.length)];
  const hdrs = ep.auth ? authHeaders : {};

  const res = getJSON(`${BASE_URL}${ep.path}`, hdrs);

  const ok = check(res, {
    'spike: status 2xx': (r) => r.status >= 200 && r.status < 300,
  });
  errorRate.add(!ok);

  sleep(0.02 + Math.random() * 0.05);
}
