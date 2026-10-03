// Phase 1 のベースライン: キャッシュのない状態で、全リクエストがオリジンに届くときの数字を取る。
//
// BASE_URL でリクエスト先を切り替える(既定はオリジン直)。エッジができたら BASE_URL=http://edge:8080 で同じシナリオを流す。
// アクセスする id は 1〜ITEMS の一様分布。人気URLに偏るロングテール分布は Phase 3 で入れる。
import http from "k6/http";
import { check } from "k6";

const BASE_URL = __ENV.BASE_URL || "http://origin:8080";
const ITEMS = parseInt(__ENV.ITEMS || "1000", 10);

export const options = {
  scenarios: {
    baseline: {
      executor: "constant-arrival-rate",
      rate: parseInt(__ENV.RATE || "200", 10), // 毎秒のリクエスト数
      timeUnit: "1s",
      duration: __ENV.DURATION || "1m",
      preAllocatedVUs: 50,
      maxVUs: 500,
    },
  },
};

export default function () {
  const id = Math.floor(Math.random() * ITEMS) + 1;
  // name タグでURLをまとめ、メトリクスの系列がidごとに増えないようにする
  const res = http.get(`${BASE_URL}/api/items/${id}`, { tags: { name: "/api/items/{id}" } });
  check(res, { "status is 200": (r) => r.status === 200 });
}
