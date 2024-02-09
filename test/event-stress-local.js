import http from 'k6/http';
import { check, sleep } from 'k6';
import { htmlReport } from "https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js";
import { textSummary } from "https://jslib.k6.io/k6-summary/0.0.1/index.js";

export let options = {
    stages: [
        { duration: '5s', target: 2000 }, // Ramp up to 2000 users over 5 seconds
        { duration: '1m', target: 2000 }, // Keep at 2000 users for 10 seconds
        { duration: '5s', target: 0 }, // Ramp down to 0 users over 5 seconds
    ],
    thresholds: {
        'http_req_duration': ['p(95)<500'], // 95% of requests should be below 500ms
    },
};

export default function () {
    let id = Math.floor(Math.random() * 97) + 1; // Generate a random id within the range [1, 97]
    let res = http.get(`http://localhost:8000/events/${id}`); // Send a GET request with the random id
    check(res, { 'status was 200': (r) => r.status === 200 }); // Check for a successful response
    sleep(1); // Pause for 1 second between iterations (optional)
}

export function handleSummary(data) {
    return {
        "event-stress-local.html": htmlReport(data),
        stdout: textSummary(data, { indent: " ", enableColors: true }),
    };
}
