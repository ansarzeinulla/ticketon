/**
 * The single place the browser talks to the Phase 2 Go API.
 *
 * Native fetch rather than Axios: everything needed here (a base URL, a bearer
 * header, JSON encoding and typed errors) is a few lines, and it keeps the
 * client bundle smaller.
 */

import { clearToken, getToken } from "@/lib/session";
import type {
  AcceptedResponse,
  Activation,
  ActivationSubmission,
  AdminSearchResponse,
  AuthResponse,
  BiletEvent,
  CheckoutInput,
  CheckoutResult,
  Campaign,
  CreateCampaignInput,
  CreateEventInput,
  CreateTicketTypeInput,
  PromoPreview,
  EventListResponse,
  EventOrder,
  EventResponse,
  AnalyticsResponse,
  ApiErrorBody,
  OpenCaseInput,
  PublicEventResponse,
  RefundResponse,
  SupportCase,
  SupportThread,
  TimelineEntry,
  TicketType,
  TicketTypeListResponse,
  TicketTypeResponse,
  UploadedImage,
  User,
  EventReport,
  EventReportsResponse,
  PlatformSetting,
  PlatformSettingsResponse,
  OrganizerProfile,
  ProfilePatch,
  CancelOrderResponse,
  SupportCaseResponse,
  StaffMember,
  StaffResponse,
  AttendeeTicket,
} from "@/lib/types";

/**
 * Where the browser reaches the API.
 *
 * NEXT_PUBLIC_ is required: this value is inlined into the client bundle at
 * build time, because the browser calls the API directly.
 */
export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080/api/v1";

/**
 * Where *this process* reaches the API when rendering on the server.
 *
 * These are not the same address once the app runs in a container. The browser
 * is on the host, so it wants http://localhost:8080; a Server Component is
 * inside the web container, where "localhost" is the web container itself and
 * the API lives at http://api:8080. Getting this wrong makes the public event
 * page and the catalogue fail while every client-side call keeps working -
 * which is exactly how it presented.
 *
 * API_INTERNAL_BASE_URL has no NEXT_PUBLIC_ prefix on purpose: it must never
 * be inlined into a bundle a browser downloads, since a browser cannot resolve
 * a container hostname anyway.
 */
const SERVER_API_BASE_URL =
  process.env.API_INTERNAL_BASE_URL || API_BASE_URL;

/** The base URL for wherever this code happens to be running. */
function baseURL(): string {
  return typeof window === "undefined" ? SERVER_API_BASE_URL : API_BASE_URL;
}

/**
 * The printable A4 PDF for one ticket.
 *
 * A direct API URL rather than a proxied route: the API answers with
 * `Content-Disposition: attachment`, which is what makes a plain link download
 * the file even across origins (the HTML `download` attribute is ignored
 * cross-origin, the header is not).
 */
export function ticketPDFURL(ticketID: string): string {
  return `${API_BASE_URL}/tickets/${ticketID}/pdf`;
}

/**
 * The admission QR as a PNG. The API renders it from the same token and with
 * the same encoder the PDF uses, so the preview on screen and the code on the
 * printed page can never drift apart.
 */
export function ticketQRURL(ticketID: string): string {
  return `${API_BASE_URL}/tickets/${ticketID}/qr.png`;
}

/**
 * The campaign QR as a PNG, for the organizer to put on a poster.
 *
 * Built here rather than from the `qr_image_url` the API returns: that field is
 * an absolute API path including /api/v1, and API_BASE_URL already ends in it.
 * Composing the two produced /api/v1/api/v1/... and a broken image.
 */
export function campaignQRURL(campaignID: string): string {
  return `${API_BASE_URL}/campaigns/${campaignID}/qr.png`;
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

  /**
   * True when a promo code stopped being usable between the preview and the
   * payment - most often because someone else took the last redemption.
   */
  get isPromoProblem(): boolean {
    return this.code.startsWith("promo_");
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
   * Overrides the stored token. Used straight after login, when the response
   * carries a token that has not been written to the cookie yet.
   */
  token?: string | null;
  /** Skip the Authorization header entirely (register and login). */
  anonymous?: boolean;
  signal?: AbortSignal;
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = "GET", body, token, anonymous = false, signal } = options;

  const headers: Record<string, string> = { Accept: "application/json" };
  if (body !== undefined) headers["Content-Type"] = "application/json";

  if (!anonymous) {
    const bearer = token ?? getToken();
    if (bearer) headers.Authorization = `Bearer ${bearer}`;
  }

  let response: Response;
  try {
    // baseURL(), not API_BASE_URL: this is the one call site that runs both in
    // the browser and in a Server Component. The URL builders below stay
    // public, because the browser is what loads a PDF, a QR image or a CSV.
    response = await fetch(`${baseURL()}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal,
      // The token travels in a header, so no cookies are sent cross-origin.
      credentials: "omit",
      cache: "no-store",
    });
  } catch (cause) {
    // An aborted request is the caller unmounting, not a failure to report.
    if (cause instanceof DOMException && cause.name === "AbortError") throw cause;

    throw new ApiError(
      0,
      "network_error",
      `Could not reach the API at ${baseURL()}. Is it running? Start it with: make api-run`,
    );
  }

  if (response.status === 204) return undefined as T;

  const raw = await response.text();
  let parsed: unknown = null;
  if (raw.length > 0) {
    try {
      parsed = JSON.parse(raw);
    } catch {
      throw new ApiError(
        response.status,
        "invalid_response",
        `The API returned a non-JSON response (HTTP ${response.status}).`,
      );
    }
  }

  if (!response.ok) {
    const envelope = parsed as ApiErrorBody | null;
    const error = envelope?.error;

    // A rejected token is stale: drop it so the app stops pretending to be
    // signed in. The AuthProvider notices and sends the user to /login.
    if (response.status === 401) clearToken();

    const remaining = (error as { remaining?: number } | undefined)?.remaining;

    throw new ApiError(
      response.status,
      error?.code ?? "unknown_error",
      error?.message ?? `Request failed with HTTP ${response.status}.`,
      error?.fields ?? {},
      typeof remaining === "number" ? remaining : undefined,
    );
  }

  return parsed as T;
}

export const api = {
  /** POST /auth/register - creates an account and returns a token with it. */
  register(input: {
    email: string;
    password: string;
    full_name?: string;
    phone?: string;
    locale?: string;
  }): Promise<AuthResponse> {
    return request<AuthResponse>("/auth/register", {
      method: "POST",
      body: input,
      anonymous: true,
    });
  },

  /** POST /auth/login */
  login(input: { email: string; password: string }): Promise<AuthResponse> {
    return request<AuthResponse>("/auth/login", {
      method: "POST",
      body: input,
      anonymous: true,
    });
  },

  /** GET /auth/me - also doubles as "is this token still good?". */
  async me(token?: string | null, signal?: AbortSignal): Promise<User> {
    const data = await request<{ user: User }>("/auth/me", { token, signal });
    return data.user;
  },

  /**
   * GET /events/mine - every event this organizer owns, drafts included.
   *
   * Not GET /events: that is the public catalogue and returns only published,
   * publicly visible events, so a newly created draft would be missing from
   * the dashboard that just created it.
   */
  listMyEvents(
    params: { limit?: number; offset?: number; status?: string; lifecycle?: string } = {},
    signal?: AbortSignal,
  ) {
    const query = new URLSearchParams();
    if (params.limit !== undefined) query.set("limit", String(params.limit));
    if (params.offset !== undefined) query.set("offset", String(params.offset));
    if (params.status) query.set("status", params.status);
    if (params.lifecycle) query.set("lifecycle", params.lifecycle);

    const suffix = query.size > 0 ? `?${query}` : "";
    return request<EventListResponse>(`/events/mine${suffix}`, { signal });
  },

  /** GET /events - the public catalogue. */
  listPublicEvents(params: { limit?: number; offset?: number } = {}, signal?: AbortSignal) {
    const query = new URLSearchParams();
    if (params.limit !== undefined) query.set("limit", String(params.limit));
    if (params.offset !== undefined) query.set("offset", String(params.offset));

    const suffix = query.size > 0 ? `?${query}` : "";
    return request<EventListResponse>(`/events${suffix}`, { signal });
  },

  /** GET /events/{id} - the organizer's own view, drafts included. */
  async getEvent(id: string, signal?: AbortSignal): Promise<BiletEvent> {
    const data = await request<EventResponse>(`/events/${id}`, { signal });
    return data.event;
  },

  /** POST /events - returns 201 with the created draft. */
  async createEvent(input: CreateEventInput): Promise<BiletEvent> {
    const data = await request<EventResponse>("/events", {
      method: "POST",
      body: input,
    });
    return data.event;
  },

  /** POST /events/{id}/publish */
  async publishEvent(id: string): Promise<BiletEvent> {
    const data = await request<EventResponse>(`/events/${id}/publish`, {
      method: "POST",
    });
    return data.event;
  },

  /**
   * POST /events/{id}/unpublish (SRS 4.2).
   *
   * Distinct from cancelling: unpublishing takes the page down while the
   * organizer reworks it, and it can be published again. Cancelling is final
   * and emails every ticket holder.
   */
  async unpublishEvent(id: string): Promise<BiletEvent> {
    const data = await request<EventResponse>(`/events/${id}/unpublish`, {
      method: "POST",
    });
    return data.event;
  },

  /** PATCH /events/{id} - edit an event (SRS 4.2). */
  async updateEvent(id: string, patch: Record<string, unknown>): Promise<BiletEvent> {
    const data = await request<EventResponse>(`/events/${id}`, {
      method: "PATCH",
      body: patch,
    });
    return data.event;
  },

  /** POST /events/{id}/cancel */
  async cancelEvent(id: string): Promise<BiletEvent> {
    const data = await request<EventResponse>(`/events/${id}/cancel`, {
      method: "POST",
    });
    return data.event;
  },

  // --- ticket types (organizer) ---------------------------------------------

  /** GET /events/{id}/ticket-types - includes hidden types. */
  async listTicketTypes(eventID: string, signal?: AbortSignal): Promise<TicketType[]> {
    const data = await request<TicketTypeListResponse>(
      `/events/${eventID}/ticket-types`,
      { signal },
    );
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
  async updateTicketType(
    id: string,
    input: Partial<CreateTicketTypeInput>,
  ): Promise<TicketType> {
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

  // --- analytics and history (organizer) --------------------------------------

  /**
   * GET /events/{id}/analytics - the dashboard figures.
   *
   * Optional filters: date range and ticket type (SRS 4.15).
   */
  eventAnalytics(
    eventID: string,
    filter: { from?: string; to?: string; ticketTypeID?: string } = {},
    signal?: AbortSignal,
  ): Promise<AnalyticsResponse> {
    const query = new URLSearchParams();
    if (filter.from) query.set("from", filter.from);
    if (filter.to) query.set("to", filter.to);
    if (filter.ticketTypeID) query.set("ticket_type_id", filter.ticketTypeID);

    const suffix = query.size > 0 ? `?${query}` : "";
    return request<AnalyticsResponse>(`/events/${eventID}/analytics${suffix}`, { signal });
  },

  /** GET /events/{id}/timeline - the chronological activity history. */
  async eventTimeline(
    eventID: string,
    filter: { from?: string; to?: string; type?: string; limit?: number } = {},
    signal?: AbortSignal,
  ): Promise<TimelineEntry[]> {
    const query = new URLSearchParams();
    if (filter.from) query.set("from", filter.from);
    if (filter.to) query.set("to", filter.to);
    if (filter.type) query.set("type", filter.type);
    if (filter.limit) query.set("limit", String(filter.limit));

    const suffix = query.size > 0 ? `?${query}` : "";
    const data = await request<{ entries: TimelineEntry[] }>(
      `/events/${eventID}/timeline${suffix}`,
      { signal },
    );
    return data.entries;
  },

  /** POST /events/{id}/duplicate - copies the setup into a new draft. */
  async duplicateEvent(
    eventID: string,
    overrides: { title?: string; starts_at?: string; ends_at?: string } = {},
  ): Promise<BiletEvent> {
    const data = await request<EventResponse>(`/events/${eventID}/duplicate`, {
      method: "POST",
      body: overrides,
    });
    return data.event;
  },

  // --- campaigns (organizer) --------------------------------------------------

  /** GET /events/{id}/campaigns */
  async listCampaigns(eventID: string, signal?: AbortSignal): Promise<Campaign[]> {
    const data = await request<{ campaigns: Campaign[] }>(
      `/events/${eventID}/campaigns`,
      { signal },
    );
    return data.campaigns;
  },

  /** POST /events/{id}/campaigns */
  async createCampaign(eventID: string, input: CreateCampaignInput): Promise<Campaign> {
    const data = await request<{ campaign: Campaign }>(`/events/${eventID}/campaigns`, {
      method: "POST",
      body: input,
    });
    return data.campaign;
  },

  /** PATCH /campaigns/{id} */
  async setCampaignActive(id: string, active: boolean): Promise<Campaign> {
    const data = await request<{ campaign: Campaign }>(`/campaigns/${id}`, {
      method: "PATCH",
      body: { active },
    });
    return data.campaign;
  },

  /** DELETE /campaigns/{id} */
  deleteCampaign(id: string): Promise<void> {
    return request<void>(`/campaigns/${id}`, { method: "DELETE" });
  },

  // --- attendee-facing --------------------------------------------------------

  /**
   * POST /events/{id}/promo/preview - prices a code against the basket.
   *
   * A preview only: nothing is reserved and no counter moves. The binding check
   * happens during checkout, because the last redemption can be taken between
   * seeing the discount and paying for it.
   */
  previewPromo(
    eventID: string,
    input: {
      code?: string;
      campaign_token?: string;
      items: { ticket_type_id: string; quantity: number }[];
    },
  ): Promise<PromoPreview> {
    return request<PromoPreview>(`/events/${eventID}/promo/preview`, {
      method: "POST",
      body: input,
      anonymous: true,
    });
  },

  /**
   * GET /public/events/{slug} - needs no token, so it works from a Server
   * Component as well as from the browser.
   */
  getPublicEvent(slug: string, signal?: AbortSignal): Promise<PublicEventResponse> {
    return request<PublicEventResponse>(`/public/events/${encodeURIComponent(slug)}`, {
      anonymous: true,
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

  /** GET /orders/{id} */
  getOrder(id: string, signal?: AbortSignal): Promise<CheckoutResult> {
    return request<CheckoutResult>(`/orders/${id}`, { anonymous: true, signal });
  },

  // --- support cases ----------------------------------------------------------

  /** GET /support/categories - the server owns the list, so it cannot drift. */
  async supportCategories(signal?: AbortSignal): Promise<string[]> {
    const data = await request<{ categories: string[] }>("/support/categories", {
      anonymous: true,
      signal,
    });
    return data.categories;
  },

  /** POST /support/cases - opens a case with its order/ticket context. */
  openSupportCase(input: OpenCaseInput): Promise<SupportThread> {
    return request<SupportThread>("/support/cases", { method: "POST", body: input });
  },

  /** GET /support/cases - the cases this account opened. */
  async mySupportCases(signal?: AbortSignal): Promise<SupportCase[]> {
    const data = await request<{ cases: SupportCase[] }>("/support/cases", { signal });
    return data.cases;
  },

  /** GET /events/{id}/support/cases - the organizer's inbox for one event. */
  async eventSupportCases(eventID: string, signal?: AbortSignal): Promise<SupportCase[]> {
    const data = await request<{ cases: SupportCase[] }>(
      `/events/${eventID}/support/cases`,
      { signal },
    );
    return data.cases;
  },

  /** GET /support/cases/{id} - the thread, filtered to what the caller may see. */
  getSupportThread(id: string, signal?: AbortSignal): Promise<SupportThread> {
    return request<SupportThread>(`/support/cases/${id}`, { signal });
  },

  /** POST /support/cases/{id}/messages */
  replyToSupportCase(
    id: string,
    message: string,
    internalNote = false,
  ): Promise<SupportThread> {
    return request<SupportThread>(`/support/cases/${id}/messages`, {
      method: "POST",
      body: { message, internal_note: internalNote },
    });
  },

  /** PATCH /support/cases/{id} - organizer only. */
  setSupportCaseStatus(id: string, status: string): Promise<SupportThread> {
    return request<SupportThread>(`/support/cases/${id}`, {
      method: "PATCH",
      body: { status },
    });
  },

  // --- Phase 10: refunds, activation ---------------------------------------

  /** GET /events/{id}/orders - the organizer's attendee view (SRS 4.9). */
  async eventOrders(eventID: string, signal?: AbortSignal): Promise<EventOrder[]> {
    const data = await request<{ orders: EventOrder[] }>(`/events/${eventID}/orders`, {
      signal,
    });
    return data.orders;
  },

  /** POST /orders/{id}/refund - organizer only, full refund. */
  refundOrder(orderID: string, reason?: string): Promise<RefundResponse> {
    return request<RefundResponse>(`/orders/${orderID}/refund`, {
      method: "POST",
      body: reason ? { reason } : {},
    });
  },

  // --- Phase 12: account recovery, the admin portal, uploads ---------------

  /** POST /auth/password-reset/request - answers the same either way. */
  requestPasswordReset(email: string): Promise<AcceptedResponse> {
    return request<AcceptedResponse>("/auth/password-reset/request", {
      method: "POST",
      body: { email },
    });
  },

  /** POST /auth/password-reset - consumes the emailed token. */
  resetPassword(token: string, password: string): Promise<AcceptedResponse> {
    return request<AcceptedResponse>("/auth/password-reset", {
      method: "POST",
      body: { token, password },
    });
  },

  /** POST /auth/verify-email - needs no session; the token is the proof. */
  verifyEmail(token: string): Promise<{ status: string; email: string; account_status: string }> {
    return request("/auth/verify-email", { method: "POST", body: { token } });
  },

  /** POST /auth/verify-email/request - re-send for the signed-in account. */
  requestEmailVerification(): Promise<AcceptedResponse> {
    return request<AcceptedResponse>("/auth/verify-email/request", { method: "POST" });
  },

  // --- Moderation (SRS 4.12) -------------------------------------------------

  /** POST /admin/events/{id}/suspend - stop a reported event selling. */
  suspendEvent(eventID: string, reason?: string): Promise<EventResponse> {
    return request(`/admin/events/${eventID}/suspend`, {
      method: "POST",
      body: reason ? { reason } : undefined,
    });
  },

  /** POST /admin/events/{id}/unsuspend - lift a suspension. */
  unsuspendEvent(eventID: string): Promise<EventResponse> {
    return request(`/admin/events/${eventID}/unsuspend`, { method: "POST" });
  },

  /** POST /admin/events/{id}/paid-sales/suspend - stop the money, keep the event. */
  suspendPaidSales(eventID: string, reason?: string): Promise<unknown> {
    return request(`/admin/events/${eventID}/paid-sales/suspend`, {
      method: "POST",
      body: reason ? { reason } : undefined,
    });
  },

  /** POST /admin/events/{id}/paid-sales/unsuspend - let it take money again. */
  restorePaidSales(eventID: string): Promise<unknown> {
    return request(`/admin/events/${eventID}/paid-sales/unsuspend`, { method: "POST" });
  },

  /** POST /admin/users/{id}/suspend - lock an account (SRS 4.12). */
  suspendUser(userID: string, reason?: string): Promise<{ user: User }> {
    return request(`/admin/users/${userID}/suspend`, {
      method: "POST",
      body: reason ? { reason } : undefined,
    });
  },

  /** POST /admin/users/{id}/unsuspend - restore an account. */
  unsuspendUser(userID: string): Promise<{ user: User }> {
    return request(`/admin/users/${userID}/unsuspend`, { method: "POST" });
  },

  /** GET /admin/event-reports - the moderation queue (SRS 4.12). */
  listEventReports(status?: string, signal?: AbortSignal): Promise<EventReportsResponse> {
    const suffix = status ? `?status=${encodeURIComponent(status)}` : "";
    return request<EventReportsResponse>(`/admin/event-reports${suffix}`, { signal });
  },

  /** PATCH /admin/event-reports/{id} - record a moderator's decision. */
  reviewEventReport(
    reportID: string,
    status: string,
    resolution?: string,
  ): Promise<{ report: EventReport }> {
    return request(`/admin/event-reports/${reportID}`, {
      method: "PATCH",
      body: { status, resolution },
    });
  },

  /** POST /events/{id}/report - anyone signed in may flag an event. */
  reportEvent(eventID: string, reason: string, details?: string): Promise<{ report: EventReport }> {
    return request(`/events/${eventID}/report`, {
      method: "POST",
      body: { reason, details },
    });
  },

  /** GET /admin/settings - the configurable platform values (SRS 4.12). */
  listSettings(signal?: AbortSignal): Promise<PlatformSettingsResponse> {
    return request<PlatformSettingsResponse>("/admin/settings", { signal });
  },

  /** PATCH /admin/settings/{key} - change one setting. */
  updateSetting(key: string, value: unknown): Promise<{ setting: PlatformSetting }> {
    return request(`/admin/settings/${encodeURIComponent(key)}`, {
      method: "PATCH",
      body: { value },
    });
  },

  // --- Organizer profile and password (SRS 4.1) -------------------------------

  /** GET /auth/profile - contact and masked payout information. */
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

  // --- Orders (SRS 4.9) --------------------------------------------------------

  /** POST /orders/{id}/cancel - withdraw a free registration. */
  cancelOrder(orderID: string, reason?: string): Promise<CancelOrderResponse> {
    return request<CancelOrderResponse>(`/orders/${orderID}/cancel`, {
      method: "POST",
      body: reason ? { reason } : undefined,
    });
  },

  // --- Support (SRS 4.13) ------------------------------------------------------

  /** POST /support/cases/{id}/assign - an empty email hands the case back. */
  assignCase(caseID: string, email: string): Promise<SupportCaseResponse> {
    return request<SupportCaseResponse>(`/support/cases/${caseID}/assign`, {
      method: "POST",
      body: { email },
    });
  },

  // --- Event staff (SRS 4.8) ---------------------------------------------------

  /** GET /events/{id}/staff - who can scan at the gate. */
  listStaff(eventID: string, signal?: AbortSignal): Promise<StaffResponse> {
    return request<StaffResponse>(`/events/${eventID}/staff`, { signal });
  },

  /** POST /events/{id}/staff - name a scanner by email. */
  assignStaff(eventID: string, email: string, role: string): Promise<{ assignment: StaffMember }> {
    return request(`/events/${eventID}/staff`, { method: "POST", body: { email, role } });
  },

  /** DELETE /events/{id}/staff/{assignmentId} - revoke the gate. */
  revokeStaff(eventID: string, assignmentID: string): Promise<void> {
    return request(`/events/${eventID}/staff/${assignmentID}`, { method: "DELETE" });
  },

  /** GET /admin/search - users, events, orders and payments (SRS 4.12). */
  adminSearch(query: string, signal?: AbortSignal): Promise<AdminSearchResponse> {
    const suffix = `?q=${encodeURIComponent(query)}`;
    return request<AdminSearchResponse>(`/admin/search${suffix}`, { signal });
  },

  /**
   * The operational report's URL (SRS 4.12).
   *
   * A download, not a fetch: the browser has to be the one asking so the file
   * lands in Downloads rather than in a JavaScript string. The token cannot
   * ride in a header on a plain navigation, so the caller fetches it and turns
   * the response into a blob - see the admin portal.
   */
  adminReportURL(): string {
    return `${API_BASE_URL}/admin/reports/events.csv`;
  },

  /** GET the CSV report as a blob, with the bearer token attached. */
  async adminReportBlob(): Promise<Blob> {
    const token = getToken();
    const response = await fetch(api.adminReportURL(), {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    });
    if (!response.ok) {
      throw new ApiError(response.status, "report_failed", "Could not build the report.");
    }
    return response.blob();
  },

  /** POST /uploads/images - multipart, for an event banner (SRS 4.2). */
  async uploadImage(file: File): Promise<UploadedImage> {
    const token = getToken();
    const form = new FormData();
    form.append("file", file);

    // Deliberately not through `request`: the browser must set its own
    // multipart Content-Type, boundary included, and a JSON header here would
    // make the upload unparseable on the server.
    const response = await fetch(`${API_BASE_URL}/uploads/images`, {
      method: "POST",
      headers: token ? { Authorization: `Bearer ${token}` } : {},
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

  /** GET /events/{id}/attendees - manual door search (SRS 4.8). */
  async eventAttendees(
    eventID: string,
    query: string,
    signal?: AbortSignal,
  ): Promise<AttendeeTicket[]> {
    const suffix = `?q=${encodeURIComponent(query)}`;
    const data = await request<{ attendees: AttendeeTicket[]; total: number }>(
      `/events/${eventID}/attendees${suffix}`,
      { signal },
    );
    return data.attendees;
  },

  /**
   * POST /events/{id}/check-in/manual - admit somebody without a QR (SRS 4.8).
   *
   * The same transaction as a camera scan, so it is neither weaker nor a way
   * around the one-admission-per-ticket rule: a second call is a 409.
   */
  checkInManually(eventID: string, ticketID: string, deviceLabel: string): Promise<unknown> {
    return request(`/events/${eventID}/check-in/manual`, {
      method: "POST",
      body: { ticket_id: ticketID, device_label: deviceLabel },
    });
  },

  /** GET /events/{id}/activation - the paid-sales checklist (SRS 4.5). */
  async eventActivation(eventID: string, signal?: AbortSignal): Promise<Activation> {
    const data = await request<{ activation: Activation }>(
      `/events/${eventID}/activation`,
      { signal },
    );
    return data.activation;
  },

  /** POST /events/{id}/activation - complete one or more checklist steps. */
  async advanceActivation(
    eventID: string,
    steps: ActivationSubmission,
  ): Promise<Activation> {
    const data = await request<{ activation: Activation }>(
      `/events/${eventID}/activation`,
      { method: "POST", body: steps },
    );
    return data.activation;
  },
};
