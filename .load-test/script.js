import http from 'k6/http';
import { sleep } from 'k6';

export const options = {
  iterations: 10,
  vus: 10,
  duration: '30s',
};

export default function () {
  http.get('http://localhost:8080/health/ready');
  sleep(1);
}