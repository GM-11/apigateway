import http from "k6/http";
import { check } from "k6";

export const options = {
  vus: 1,
  duration: "15s",
};

export default function () {
  const res = http.get("http://localhost:8080/auth");
  check(res, {
    "unauthorized (401)": (r) => r.status === 401,
    "not rate limited without auth (not 429)": (r) => r.status !== 429,
  });
}
