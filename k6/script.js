import { randomIntBetween, randomItem } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';
import http from 'k6/http';
import { sleep } from 'k6';

export const options = {
	scenarios: {
		constant_request_rate: {
			executor: 'constant-arrival-rate',
			rate: 24000,
			timeUnit: '1s',
			duration: '1m',
			preAllocatedVUs: 50,
			maxVUs: 10000,
		},
	},
};

let urlString = "http://127.0.0.1:8000/api/v1/ad";

const limits = [5, 10, 15];
const genders = ['M', 'F'];
const countries = ['US', 'TW', 'GB', 'AU', 'FR', 'DE'];
const platforms = ['android', 'ios', 'web'];

export default function () {
	const limit = randomItem(limits);
	const age = randomIntBetween(1, 100);
	const gender = randomItem(genders);
	const country = randomItem(countries);
	const platform = randomItem(platforms);

	for (let i = 0; i < 10; i++) {
		// 用 new URL 效能會變差
		http.get(
			`${urlString}?limit=${limit}&offset=${i}&age=${age}&gender=${gender}&country=${country}&platform=${platform}`,
			{
				tags: {
					name: 'GetAd'
				}
			}
		);
	}

	sleep(1);
}
