// k6 → Prometheus → Grafana の経路を確かめるためのシナリオ。
// Phase 1 でオリジンができたら baseline.js を追加する。
import http from "k6/http";
import { check } from "k6";

export const options = {
  vus: 2,
  duration: "10s",
};

export default function () {
  const res = http.get("http://prometheus:9090/-/healthy");
  check(res, { "status is 200": (r) => r.status === 200 });
}
