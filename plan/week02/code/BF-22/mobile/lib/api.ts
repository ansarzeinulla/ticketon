import { API_BASE_URL } from "./config";
import { loadToken } from "./session";
import type { AuthResponse, User } from "./types";

/** A failed API call, carrying the code from the Go error envelope. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }

  get isNetworkError(): boolean {
    return this.status === 0;
  }
}

interface RequestOptions {
  method?: "GET" | "POST";
  body?: unknown;
  token?: string | null;
  anonymous?: boolean;
}

/** Staff at a door cannot wait on a hung request. */
const REQUEST_TIMEOUT_MS = 10_000;

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = "GET", body, token, anonymous = false } = options;

  const headers: Record<string, string> = { Accept: "application/json" };
  if (body !== undefined) headers["Content-Type"] = "application/json";

  if (!anonymous) {
    const bearer = token ?? (await loadToken());
    if (bearer) headers.Authorization = `Bearer ${bearer}`;
  }

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);

  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: controller.signal,
    });
  } catch {
    throw new ApiError(
      0,
      "network_error",
      `Cannot reach the BiletFlow API at ${API_BASE_URL}. Check the Wi-Fi and try again.`,
    );
  } finally {
    clearTimeout(timeout);
  }

  const parsed: unknown = await response.json().catch(() => null);

  if (!response.ok) {
    const error = (parsed as { error?: { code?: string; message?: string } } | null)?.error;
    throw new ApiError(
      response.status,
      error?.code ?? "unknown_error",
      error?.message ?? `Request failed with HTTP ${response.status}.`,
    );
  }

  return parsed as T;
}

export const api = {
  login(email: string, password: string): Promise<AuthResponse> {
    return request<AuthResponse>("/auth/login", {
      method: "POST",
      body: { email, password },
      anonymous: true,
    });
  },

  async me(token?: string): Promise<User> {
    const data = await request<{ user: User }>("/auth/me", { token });
    return data.user;
  },
};
