/**
 * The single place the browser talks to the Go API.
 *
 * Native fetch rather than Axios: a base URL, a bearer header, JSON encoding
 * and typed errors are a few lines each.
 */

import { env } from "@/lib/env";
import type { ApiErrorBody, AuthResponse, User } from "@/lib/types";

export const API_BASE_URL = env.apiBaseUrl;

/**
 * A failed API call, carrying the pieces of the Go error envelope so the UI can
 * react to `code` and highlight the exact inputs named in `fields`.
 */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly fields: Record<string, string>;

  constructor(status: number, code: string, message: string, fields: Record<string, string> = {}) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.fields = fields;
  }

  /** True when the API could not be reached at all. */
  get isNetworkError(): boolean {
    return this.status === 0;
  }
}

interface RequestOptions {
  method?: "GET" | "POST" | "PATCH" | "DELETE";
  body?: unknown;
  /** Sent as `Authorization: Bearer <token>` when present. */
  token?: string | null;
  signal?: AbortSignal;
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = "GET", body, token, signal } = options;

  const headers: Record<string, string> = { Accept: "application/json" };
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (token) headers.Authorization = `Bearer ${token}`;

  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal,
      cache: "no-store",
    });
  } catch (cause) {
    if (cause instanceof DOMException && cause.name === "AbortError") throw cause;
    throw new ApiError(0, "network_error", `Could not reach the API at ${API_BASE_URL}.`);
  }

  if (response.status === 204) return undefined as T;

  const payload: unknown = await response.json().catch(() => null);
  if (!response.ok) {
    const error = (payload as ApiErrorBody | null)?.error;
    throw new ApiError(
      response.status,
      error?.code ?? "unknown_error",
      error?.message ?? `Request failed with HTTP ${response.status}.`,
      error?.fields ?? {},
    );
  }
  return payload as T;
}

export const api = {
  /** Create an account. The response already carries a token. */
  register(input: { email: string; password: string; full_name?: string }): Promise<AuthResponse> {
    return request<AuthResponse>("/auth/register", { method: "POST", body: input });
  },

  /** Exchange credentials for a token. */
  login(input: { email: string; password: string }): Promise<AuthResponse> {
    return request<AuthResponse>("/auth/login", { method: "POST", body: input });
  },

  /** Who the token belongs to. */
  async me(token: string, signal?: AbortSignal): Promise<User> {
    const data = await request<{ user: User }>("/auth/me", { token, signal });
    return data.user;
  },
};
