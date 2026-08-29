import { API_BASE_URL } from "./config";
import { loadToken } from "./session";
import type {
  AttendeeTicket,
  AuthResponse,
  CheckInStats,
  CheckInSuccess,
  ScanErrorCode,
  ScannableEvent,
  User,
} from "./types";

/**
 * A failed API call. `code` is what the scanner switches on to choose between
 * the green and the red screen, and the extra fields let the red screen say
 * something specific instead of "denied".
 */
export class ApiError extends Error {
  readonly status: number;
  readonly code: ScanErrorCode;
  readonly attendeeName?: string;
  readonly checkedInAt?: string;
  readonly stats?: CheckInStats;

  constructor(
    status: number,
    code: ScanErrorCode,
    message: string,
    extra: { attendeeName?: string; checkedInAt?: string; stats?: CheckInStats } = {},
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.attendeeName = extra.attendeeName;
    this.checkedInAt = extra.checkedInAt;
    this.stats = extra.stats;
  }

  get isNetworkError(): boolean {
    return this.status === 0;
  }

  /** A session that the API no longer accepts; the app returns to sign-in. */
  get isSessionExpired(): boolean {
    return this.status === 401;
  }
}

interface RequestOptions {
  method?: "GET" | "POST";
  body?: unknown;
  token?: string | null;
  anonymous?: boolean;
  signal?: AbortSignal;
}

/** A scan should fail fast: staff cannot wait at a gate. */
const REQUEST_TIMEOUT_MS = 10_000;

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = "GET", body, token, anonymous = false, signal } = options;

  const headers: Record<string, string> = { Accept: "application/json" };
  if (body !== undefined) headers["Content-Type"] = "application/json";

  if (!anonymous) {
    const bearer = token ?? (await loadToken());
    if (bearer) headers.Authorization = `Bearer ${bearer}`;
  }

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  if (signal) signal.addEventListener("abort", () => controller.abort());

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
      `Cannot reach the BiletFlow API at ${API_BASE_URL}. Check the venue Wi-Fi and try again.`,
    );
  } finally {
    clearTimeout(timeout);
  }

  const raw = await response.text();
  let parsed: unknown = null;
  if (raw.length > 0) {
    try {
      parsed = JSON.parse(raw);
    } catch {
      throw new ApiError(response.status, "invalid_response",
        `The API returned an unexpected response (HTTP ${response.status}).`);
    }
  }

  if (!response.ok) {
    const envelope = parsed as
      | {
          error?: {
            code?: string;
            message?: string;
            attendee_name?: string;
            checked_in_at?: string;
            stats?: CheckInStats;
          };
        }
      | null;
    const error = envelope?.error;

    throw new ApiError(
      response.status,
      error?.code ?? "unknown_error",
      error?.message ?? `Request failed with HTTP ${response.status}.`,
      {
        attendeeName: error?.attendee_name,
        checkedInAt: error?.checked_in_at,
        stats: error?.stats,
      },
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

  async me(token?: string | null): Promise<User> {
    const data = await request<{ user: User }>("/auth/me", { token });
    return data.user;
  },

  /** The events this account may run check-in for - the event selector. */
  async scannableEvents(): Promise<ScannableEvent[]> {
    const data = await request<{ events: ScannableEvent[] }>("/events/scannable");
    return data.events;
  },

  /** The scan itself. Throws ApiError with a scanner code on refusal. */
  async checkIn(eventID: string, qrToken: string, device: string): Promise<CheckInSuccess> {
    const data = await request<{ result: string; check_in: CheckInSuccess }>(
      `/events/${eventID}/check-in`,
      { method: "POST", body: { qr_token: qrToken, device_label: device } },
    );
    return data.check_in;
  },

  /** Manual attendee lookup for when a QR will not scan (SRS 4.8). */
  async searchAttendees(eventID: string, query: string): Promise<AttendeeTicket[]> {
    const data = await request<{ attendees: AttendeeTicket[] }>(
      `/events/${eventID}/attendees?q=${encodeURIComponent(query)}`,
    );
    return data.attendees;
  },

  /**
   * Admit somebody found by name rather than by camera (SRS 4.8).
   *
   * The server runs the identical transaction as a scan - same row lock, same
   * duplicate protection - so this is a real check-in, not a lesser one.
   */
  async checkInManually(
    eventID: string,
    ticketID: string,
    device: string,
  ): Promise<CheckInSuccess> {
    const data = await request<{ result: string; check_in: CheckInSuccess }>(
      `/events/${eventID}/check-in/manual`,
      { method: "POST", body: { ticket_id: ticketID, device_label: device } },
    );
    return data.check_in;
  },

  async stats(eventID: string): Promise<CheckInStats> {
    const data = await request<{ stats: CheckInStats }>(`/events/${eventID}/check-in/stats`);
    return data.stats;
  },

  /** Undo an accidental admission (SRS 4.8). */
  async reverseCheckIn(ticketID: string, reason: string): Promise<CheckInStats> {
    const data = await request<{ stats: CheckInStats }>(
      `/tickets/${ticketID}/check-in/reverse`,
      { method: "POST", body: { reason } },
    );
    return data.stats;
  },
};
