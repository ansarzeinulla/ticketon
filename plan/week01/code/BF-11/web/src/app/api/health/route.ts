/**
 * Pings the Go API from the web server, so the browser can tell whether the
 * backend is reachable without knowing its address.
 */

const API_BASE_URL = process.env.API_BASE_URL ?? "http://localhost:8080";

export async function GET() {
  try {
    const res = await fetch(`${API_BASE_URL}/health`, { cache: "no-store" });
    const body = await res.json().catch(() => null);
    return Response.json({ web: "ok", api: res.ok ? "ok" : "degraded", details: body }, {
      status: res.ok ? 200 : 503,
    });
  } catch {
    return Response.json({ web: "ok", api: "unreachable" }, { status: 503 });
  }
}
