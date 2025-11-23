import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
    vus: 5,
    duration: '1m',
    thresholds: {
    http_req_duration: ['p(95)<300'],
    http_req_failed: ['rate<0.001'],
    },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export function setup() {
    const teamName = `loadtest-${Date.now()}`;
    const membersCount = 5;
    const members = [];

    for (let i = 1; i <= membersCount; i++) {
    const id = `${teamName}-u${i}`;
    members.push({
        user_id: id,
        username: `load-${i}`,
        is_active: true,
    });
    }

    const payload = JSON.stringify({
    team_name: teamName,
    members,
    });

    const res = http.post(`${BASE_URL}/team/add`, payload, {
    headers: { 'Content-Type': 'application/json' },
    });

    check(res, {
    'team created 201': (r) => r.status === 201,
    });

    return {
    teamName,
    userIds: members.map((m) => m.user_id),
    };
}

export default function (data) {
    const { userIds } = data;
  const authorId = userIds[Math.floor(Math.random() * userIds.length)];

  const prId = `pr-${__VU}-${Date.now()}-${Math.floor(Math.random() * 1e6)}`;
    const prName = 'loadtest-pr';

    const createRes = http.post(
    `${BASE_URL}/pullRequest/create`,
    JSON.stringify({
        pull_request_id: prId,
        pull_request_name: prName,
        author_id: authorId,
    }),
    {
        headers: { 'Content-Type': 'application/json' },
    },
    );

    check(createRes, {
    'create PR 201': (r) => r.status === 201,
    });

    const reviewRes = http.get(
    `${BASE_URL}/users/getReview?user_id=${encodeURIComponent(authorId)}`,
    );
    check(reviewRes, {
    'getReview 200': (r) => r.status === 200,
    });

    const statsRes = http.get(`${BASE_URL}/stats/reviewers`);
    check(statsRes, {
    'stats 200': (r) => r.status === 200,
    });

    sleep(1);
}
