import http from 'k6/http';
import { check, sleep } from 'k6';
import { htmlReport } from "https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js";
import { textSummary } from "https://jslib.k6.io/k6-summary/0.0.1/index.js";

export let options = {
    stages: [
        // Adjust the stages to fit your load testing requirements
        { duration: '30s', target: 100 }, // Ramp up to 100 users over 30 seconds
        { duration: '1m', target: 100 },  // Stay at 100 users for 1 minute
        { duration: '30s', target: 0 },   // Ramp down to 0 users over 30 seconds
    ],
    thresholds: {
        // You can define thresholds to specify acceptable performance
        'http_req_duration': ['p(95)<500'], // 95% of requests should be below 500ms
    },
};

export default function () {

    // Send a GET request to the specified URL with the random id
    let res = http.get("http://localhost:8000/events?limit=100");
    // TODO: query events of user

    // Check if the response was successful (HTTP status 200)
    check(res, { 'status was 200': (r) => r.status === 200 });

    // Optionally, you can pause for a short time between iterations
    sleep(1);
}

export function handleSummary(data) {
    return {
        "user-events-local.html": htmlReport(data),
        stdout: textSummary(data, { indent: " ", enableColors: true }),
    };
}