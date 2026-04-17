import http from "k6/http";
import { check } from "k6";

export const options = {
  vus: 20,
  duration: "30s",
};

export default function () {
  const res = http.get("http://localhost:8080/fail");
  check(res, {
    "gateway returns 503 when upstream errors trigger circuit logic": (r) =>
      r.status === 503,
  });
}
