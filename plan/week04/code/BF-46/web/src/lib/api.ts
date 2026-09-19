/**
 * The single place the browser talks to the Go API.
 *
 * Native fetch rather than Axios: a base URL, a bearer header, JSON encoding
 * and typed errors are a few lines each.
 */

import { getToken } from "@/lib/session";
import type {
  AcceptedResponse,
  Activation,
  ActivationSubmission,
  ApiErrorBody,
  AuthResponse,
  BiletEvent,
  CheckoutInput,
  CheckoutResult,
  CreateEventInput,
  CreateTicketTypeInput,
  EventListResponse,
  EventOrder,
  EventResponse,
  Guest,
  OrganizerProfile,
  ProfilePatch,
  PublicEventResponse,
  SeatMap,
  TicketType,
  TicketTypeListResponse,
  TicketTypeResponse,
  UploadedImage,
  User,
} from "@/lib/types";

/**
 * Where the browser reaches the Go API. NEXT_PUBLIC_ variables are inlined at
 * build time, so this is a plain constant in the bundle.
 */
export const API_BASE_URL = (
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080/api/v1"
).replace(/\/+$/, "");

/**
 * The printable A4 PDF for one ticket.
 *
 * A plain link rather than a fetch: the API answers with
 * `Content-Disposition: attachment`, which is what makes the browser download
 * the file across origins (the HTML `download` attribute is ignored
 * cross-origin, the header is not).
 */
export function ticketPDFURL(ticketID: string): string {
  return `${API_BASE_URL}/tickets/${ticketID}/pdf`;
}

/**
 * The admission QR as a PNG. The API renders it with the same encoder the PDF
 * uses, so the preview on screen and the code on the printed page can never
 * drift apart.
 */
export function ticketQRURL(ticketID: string): string {
  return `${API_BASE_URL}/tickets/${ticketID}/qr.png`;
}

/**
 * A failed API call, carrying the pieces of the Go error envelope so the UI can
 * react to `code` and highlight the exact inputs named in `fields`.
 */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly fields: Record<string, string>;
  /** Set on an insufficient_inventory error: how many are actually left. */
  readonly remaining?: number;

  constructor(
    status: number,
    code: string,
    message: string,
    fields: Record<string, string> = {},
    remaining?: number,
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.fields = fields;
    this.remaining = remaining;
  }

  /** True when stock ran out between loading the page and checking out. */
  get isSoldOut(): boolean {
    return this.code === "insufficient_inventory";
  }

  /** True when the API could not be reached at all. */
  get isNetworkError(): boolean {
    return this.status === 0;
  }
}

interface RequestOptions {
  method?: "GET" | "POST" | "PATCH" | "DELETE";
  body?: unknown;
  /**
   * Overrides the stored token. Omitted, the cookie's token is used; null sends
   * no Authorization header at all (register and login).
   */
  token?: string | null;
  signal?: AbortSignal;
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = "GET", body, token = getToken(), signal } = options;

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
    const remaining = (error as { remaining?: number } | undefined)?.remaining;
    throw new ApiError(
      response.status,
      error?.code ?? "unknown_error",
      error?.message ?? `Request failed with HTTP ${response.status}.`,
      error?.fields ?? {},
      typeof remaining === "number" ? remaining : undefined,
    );
  }
  return payload as T;
}

export const api = {
  /** Create an account. The response already carries a token. */
  register(input: { email: string; password: string; full_name?: string }): Promise<AuthResponse> {
    return request<AuthResponse>("/auth/register", { method: "POST", body: input, token: null });
  },

  /** Exchange credentials for a token. */
  login(input: { email: string; password: string }): Promise<AuthResponse> {
    return request<AuthResponse>("/auth/login", { method: "POST", body: input, token: null });
  },

  /** Who the token belongs to. */
  async me(token: string, signal?: AbortSignal): Promise<User> {
    const data = await request<{ user: User }>("/auth/me", { token, signal });
    return data.user;
  },

  /** POST /auth/password-reset/request - answers the same either way. */
  requestPasswordReset(email: string): Promise<AcceptedResponse> {
    return request<AcceptedResponse>("/auth/password-reset/request", {
      method: "POST",
      body: { email },
      token: null,
    });
  },

  /** POST /auth/password-reset - consumes the emailed token. */
  resetPassword(token: string, password: string): Promise<AcceptedResponse> {
    return request<AcceptedResponse>("/auth/password-reset", {
      method: "POST",
      body: { token, password },
      token: null,
    });
  },

  /** POST /auth/verify-email - needs no session; the token is the proof. */
  verifyEmail(token: string): Promise<{ status: string; email: string; account_status: string }> {
    return request("/auth/verify-email", { method: "POST", body: { token }, token: null });
  },

  /** POST /auth/verify-email/request - re-send for the signed-in account. */
  requestEmailVerification(): Promise<AcceptedResponse> {
    return request<AcceptedResponse>("/auth/verify-email/request", { method: "POST" });
  },

  // --- the public catalogue -----------------------------------------------------

  /** GET /events - published, public events, soonest first. */
  listPublicEvents(params: { limit?: number; offset?: number } = {}, signal?: AbortSignal) {
    const query = new URLSearchParams();
    if (params.limit !== undefined) query.set("limit", String(params.limit));
    if (params.offset !== undefined) query.set("offset", String(params.offset));

    const suffix = query.size > 0 ? `?${query}` : "";
    return request<EventListResponse>(`/events${suffix}`, { token: null, signal });
  },

  /**
   * GET /public/events/{slug} - needs no token, so it works from a Server
   * Component as well as from the browser.
   */
  getPublicEvent(slug: string, signal?: AbortSignal): Promise<PublicEventResponse> {
    return request<PublicEventResponse>(`/public/events/${encodeURIComponent(slug)}`, {
      token: null,
      signal,
    });
  },

  /** POST /events/{id}/checkout - the simulated purchase. */
  checkout(eventID: string, input: CheckoutInput): Promise<CheckoutResult> {
    return request<CheckoutResult>(`/events/${eventID}/checkout`, {
      method: "POST",
      body: input,
    });
  },

  /** GET /events/{id}/seats - the live seat map of an assigned-seating event. */
  async seatMap(eventID: string, signal?: AbortSignal): Promise<SeatMap> {
    const data = await request<{ seat_map: SeatMap }>(`/events/${eventID}/seats`, {
      token: null,
      signal,
    });
    return data.seat_map;
  },

  /** GET /orders/{id} - the id is the capability, so no token is needed. */
  getOrder(id: string, signal?: AbortSignal): Promise<CheckoutResult> {
    return request<CheckoutResult>(`/orders/${id}`, { token: null, signal });
  },

  // --- events (organizer) -------------------------------------------------------

  /**
   * GET /events/mine - every event this organizer owns, drafts included.
   *
   * Not GET /events: that is the public catalogue and returns only published,
   * publicly visible events, so a newly created draft would be missing from
   * the dashboard that just created it.
   */
  listMyEvents(params: { limit?: number; offset?: number; status?: string } = {}, signal?: AbortSignal) {
    const query = new URLSearchParams();
    if (params.limit !== undefined) query.set("limit", String(params.limit));
    if (params.offset !== undefined) query.set("offset", String(params.offset));
    if (params.status) query.set("status", params.status);

    const suffix = query.size > 0 ? `?${query}` : "";
    return request<EventListResponse>(`/events/mine${suffix}`, { signal });
  },

  /** GET /events/{id} - the organizer's own view, drafts included. */
  async getEvent(id: string, signal?: AbortSignal): Promise<BiletEvent> {
    const data = await request<EventResponse>(`/events/${id}`, { signal });
    return data.event;
  },

  /** POST /events - returns 201 with the created draft. */
  async createEvent(input: CreateEventInput): Promise<BiletEvent> {
    const data = await request<EventResponse>("/events", { method: "POST", body: input });
    return data.event;
  },

  /** PATCH /events/{id} - edit an event (SRS 4.2). */
  async updateEvent(id: string, patch: Record<string, unknown>): Promise<BiletEvent> {
    const data = await request<EventResponse>(`/events/${id}`, { method: "PATCH", body: patch });
    return data.event;
  },

  /** POST /events/{id}/publish */
  async publishEvent(id: string): Promise<BiletEvent> {
    const data = await request<EventResponse>(`/events/${id}/publish`, { method: "POST" });
    return data.event;
  },

  /**
   * POST /events/{id}/unpublish - takes the page down while the organizer
   * reworks it; it can be published again. Cancelling is final.
   */
  async unpublishEvent(id: string): Promise<BiletEvent> {
    const data = await request<EventResponse>(`/events/${id}/unpublish`, { method: "POST" });
    return data.event;
  },

  /** POST /events/{id}/cancel */
  async cancelEvent(id: string): Promise<BiletEvent> {
    const data = await request<EventResponse>(`/events/${id}/cancel`, { method: "POST" });
    return data.event;
  },

  // --- ticket types (organizer) -------------------------------------------------

  /** GET /events/{id}/ticket-types - includes hidden types. */
  async listTicketTypes(eventID: string, signal?: AbortSignal): Promise<TicketType[]> {
    const data = await request<TicketTypeListResponse>(`/events/${eventID}/ticket-types`, {
      signal,
    });
    return data.ticket_types;
  },

  /** POST /events/{id}/ticket-types */
  async createTicketType(eventID: string, input: CreateTicketTypeInput): Promise<TicketType> {
    const data = await request<TicketTypeResponse>(`/events/${eventID}/ticket-types`, {
      method: "POST",
      body: input,
    });
    return data.ticket_type;
  },

  /** PATCH /ticket-types/{id} */
  async updateTicketType(id: string, input: Partial<CreateTicketTypeInput>): Promise<TicketType> {
    const data = await request<TicketTypeResponse>(`/ticket-types/${id}`, {
      method: "PATCH",
      body: input,
    });
    return data.ticket_type;
  },

  /** DELETE /ticket-types/{id} */
  deleteTicketType(id: string): Promise<void> {
    return request<void>(`/ticket-types/${id}`, { method: "DELETE" });
  },

  // --- organizer profile (SRS 4.1) ---------------------------------------------

  /** GET /auth/profile - the caller's organizer profile, with masked payouts. */
  getProfile(signal?: AbortSignal): Promise<{ profile: OrganizerProfile }> {
    return request("/auth/profile", { signal });
  },

  /** PATCH /auth/profile - an absent key is left alone, an explicit null clears. */
  updateProfile(patch: ProfilePatch): Promise<{ profile: OrganizerProfile }> {
    return request("/auth/profile", { method: "PATCH", body: patch });
  },

  /** POST /auth/password - change the signed-in account's password. */
  changePassword(currentPassword: string, newPassword: string): Promise<void> {
    return request("/auth/password", {
      method: "POST",
      body: { current_password: currentPassword, new_password: newPassword },
    });
  },

  // --- paid-sales activation (SRS 4.5) -------------------------------------------

  /** GET /events/{id}/activation - the checklist and what is outstanding. */
  async eventActivation(eventID: string, signal?: AbortSignal): Promise<Activation> {
    const data = await request<{ activation: Activation }>(`/events/${eventID}/activation`, {
      signal,
    });
    return data.activation;
  },

  /** POST /events/{id}/activation - complete one or more checklist steps. */
  async advanceActivation(eventID: string, steps: ActivationSubmission): Promise<Activation> {
    const data = await request<{ activation: Activation }>(`/events/${eventID}/activation`, {
      method: "POST",
      body: steps,
    });
    return data.activation;
  },

  // --- who is coming (organizer) -----------------------------------------------

  /** GET /events/{id}/orders - newest first. */
  async eventOrders(eventID: string, signal?: AbortSignal): Promise<EventOrder[]> {
    const data = await request<{ orders: EventOrder[] }>(`/events/${eventID}/orders`, { signal });
    return data.orders;
  },

  /** GET /events/{id}/attendees - every issued ticket, optionally searched. */
  async eventGuests(eventID: string, query: string, signal?: AbortSignal): Promise<Guest[]> {
    const suffix = query ? `?q=${encodeURIComponent(query)}` : "";
    const data = await request<{ attendees: Guest[] }>(`/events/${eventID}/attendees${suffix}`, {
      signal,
    });
    return data.attendees;
  },

  /** POST /uploads/images - an event banner (SRS 4.2). */
  async uploadImage(file: File): Promise<UploadedImage> {
    const form = new FormData();
    form.append("file", file);

    // Deliberately not through `request`: the browser must set its own
    // multipart Content-Type, boundary included, and a JSON header here would
    // make the upload unparseable on the server.
    const headers: Record<string, string> = {};
    const token = getToken();
    if (token) headers.Authorization = `Bearer ${token}`;

    const response = await fetch(`${API_BASE_URL}/uploads/images`, {
      method: "POST",
      headers,
      body: form,
    });

    const payload = await response.json().catch(() => null);
    if (!response.ok) {
      const body = payload as ApiErrorBody | null;
      throw new ApiError(
        response.status,
        body?.error?.code ?? "upload_failed",
        body?.error?.message ?? "Could not upload that image.",
      );
    }
    return payload as UploadedImage;
  },
};
