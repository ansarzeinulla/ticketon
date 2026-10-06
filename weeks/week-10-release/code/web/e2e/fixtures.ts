import { test as base, expect, type Page } from "@playwright/test";

/**
 * Shared setup for the end-to-end suite.
 *
 * Accounts and events are created through the API rather than by clicking
 * through the UI. Two reasons: a spec about refunds should fail when refunds
 * break, not when the create-event form changes; and `make seed` produces no
 * usable logins, so there is no fixture user to sign in as.
 */

export const BASE_URL = process.env.E2E_BASE_URL ?? "http://localhost:3000";

export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080/api/v1";

/** The password every test account uses. Long enough to pass validation. */
export const PASSWORD = "correct horse battery staple";

export interface Account {
  id: string;
  email: string;
  password: string;
  token: string;
}

let counter = 0;

/**
 * A unique suffix per call.
 *
 * Date.now() alone is not enough: Playwright runs each spec file in its own
 * worker process, so two workers start with counter 0 and can call this in the
 * same millisecond - which showed up as a 409 on registration. The random part
 * is what makes it unique across processes.
 */
function unique(): string {
  counter += 1;
  return `${Date.now().toString(36)}${counter}${Math.random().toString(36).slice(2, 8)}`;
}

/** A unique address per call, so parallel workers cannot collide. */
export function uniqueEmail(prefix: string): string {
  return `e2e.${prefix}.${unique()}@biletflow.test`;
}

/**
 * A unique event title.
 *
 * The database is not reset between runs, so a fixed title accumulates: the
 * second run finds two events called "Catalogue Free Lecture" and a locator
 * that matched one yesterday matches two today. Suffixing makes every spec's
 * event findable by name.
 */
export function uniqueTitle(base: string): string {
  return `${base} ${unique()}`;
}

async function apiCall<T>(
  path: string,
  init: { method?: string; token?: string; body?: unknown } = {},
): Promise<T> {
  const headers: Record<string, string> = { Accept: "application/json" };
  if (init.body !== undefined) headers["Content-Type"] = "application/json";
  if (init.token) headers.Authorization = `Bearer ${init.token}`;

  const response = await fetch(`${API_BASE_URL}${path}`, {
    method: init.method ?? "GET",
    headers,
    body: init.body === undefined ? undefined : JSON.stringify(init.body),
  });

  const text = await response.text();
  if (!response.ok) {
    throw new Error(`${init.method ?? "GET"} ${path} → ${response.status}: ${text}`);
  }
  return (text ? JSON.parse(text) : undefined) as T;
}

export const api = {
  async register(prefix: string, fullName = "E2E Tester"): Promise<Account> {
    const email = uniqueEmail(prefix);
    const data = await apiCall<{ user: { id: string }; access_token: string }>(
      "/auth/register",
      { method: "POST", body: { email, password: PASSWORD, full_name: fullName } },
    );
    return { id: data.user.id, email, password: PASSWORD, token: data.access_token };
  },

  async login(email: string): Promise<string> {
    const data = await apiCall<{ access_token: string }>("/auth/login", {
      method: "POST",
      body: { email, password: PASSWORD },
    });
    return data.access_token;
  },

  async createEvent(
    token: string,
    title: string,
    overrides: Record<string, unknown> = {},
  ): Promise<{ id: string; slug: string }> {
    const data = await apiCall<{ event: { id: string; slug: string } }>("/events", {
      method: "POST",
      token,
      body: {
        title,
        starts_at: "2027-09-01T15:00:00Z",
        ends_at: "2027-09-01T19:00:00Z",
        timezone: "Asia/Almaty",
        venue_name: "Almaty Demo Hall",
        venue_address: "Abay Avenue 44, Almaty",
        capacity: 200,
        ...overrides,
      },
    });
    return data.event;
  },

  async addTicketType(
    token: string,
    eventID: string,
    name: string,
    priceKZT: string,
    quantity: number,
  ): Promise<string> {
    const data = await apiCall<{ ticket_type: { id: string } }>(
      `/events/${eventID}/ticket-types`,
      {
        method: "POST",
        token,
        body: { name, price_kzt: priceKZT, quantity_total: quantity },
      },
    );
    return data.ticket_type.id;
  },

  /** Complete the four-step checklist, which is what clears an event to take money. */
  activatePaidSales(token: string, eventID: string) {
    return apiCall(`/events/${eventID}/activation`, {
      method: "POST",
      token,
      body: {
        confirm_identity: true,
        confirm_payout: true,
        accept_terms: true,
        pay_activation_fee: true,
      },
    });
  },

  publish(token: string, eventID: string) {
    return apiCall(`/events/${eventID}/publish`, { method: "POST", token });
  },

  async checkout(
    eventID: string,
    ticketTypeID: string,
    quantity: number,
    buyerName: string,
    buyerEmail: string,
    token?: string,
  ): Promise<{ id: string; tickets: { id: string }[] }> {
    const data = await apiCall<{ order: { id: string }; tickets: { id: string }[] }>(
      `/events/${eventID}/checkout`,
      {
        method: "POST",
        token,
        body: {
          buyer_name: buyerName,
          buyer_email: buyerEmail,
          items: [{ ticket_type_id: ticketTypeID, quantity }],
        },
      },
    );
    return { id: data.order.id, tickets: data.tickets };
  },

  /** Admit somebody by ticket id, the way the door search does (SRS 4.8). */
  checkInManually(token: string, eventID: string, ticketID: string) {
    return apiCall(`/events/${eventID}/check-in/manual`, {
      method: "POST",
      token,
      body: { ticket_id: ticketID, device_label: "e2e fixture" },
    });
  },

  /**
   * Create a campaign, returning the code it was actually given.
   *
   * Promo codes are unique across the whole platform (a citext unique index),
   * not per event, so a fixed code collides on the second run of the suite.
   * The caller passes a prefix and gets the real code back.
   */
  async createCampaign(
    token: string,
    eventID: string,
    prefix: string,
    percent: number,
  ): Promise<string> {
    const code = `${prefix}${unique().toUpperCase().replace(/[^A-Z0-9]/g, "")}`.slice(0, 32);
    await apiCall<{ campaign: { id: string; promo_code: string } }>(
      `/events/${eventID}/campaigns`,
      {
        method: "POST",
        token,
        body: {
          name: `${prefix} campaign`,
          code,
          discount_type: "percentage",
          // The API takes money and percentages as decimal strings, never as
          // numbers - the same rule that keeps KZT out of a float.
          discount_value: String(percent),
          max_redemptions: 50,
        },
      },
    );
    return code;
  },
};

/**
 * A published free event with one ticket type, ready to register for.
 */
export async function freeEvent(base: string, overrides: Record<string, unknown> = {}) {
  const organizer = await api.register("organizer", "Demo Organizer");
  const title = uniqueTitle(base);
  const event = await api.createEvent(organizer.token, title, overrides);
  const ticketTypeID = await api.addTicketType(
    organizer.token,
    event.id,
    "Free Entry",
    "0",
    50,
  );
  await api.publish(organizer.token, event.id);
  return { organizer, event, ticketTypeID, title };
}

/**
 * A published paid event that has completed its activation checklist, so it is
 * genuinely able to sell.
 */
export async function paidEvent(base: string, priceKZT = "5000") {
  const organizer = await api.register("organizer", "Demo Organizer");
  const title = uniqueTitle(base);
  const event = await api.createEvent(organizer.token, title);
  const ticketTypeID = await api.addTicketType(
    organizer.token,
    event.id,
    "General Admission",
    priceKZT,
    50,
  );
  await api.activatePaidSales(organizer.token, event.id);
  await api.publish(organizer.token, event.id);
  return { organizer, event, ticketTypeID, title };
}

/**
 * Sign a browser context in without going through the form.
 *
 * The token lives in a plain cookie the proxy can read (see lib/session.ts), so
 * seeding it is exactly what a real sign-in does. Specs that are *about*
 * signing in use the form instead.
 */
export async function signIn(page: Page, token: string) {
  // `url` and `path` are mutually exclusive in Playwright's cookie API; the
  // url form is enough, since the cookie lib.session.ts writes is path "/".
  await page.context().addCookies([
    { name: "biletflow_token", value: token, url: BASE_URL },
  ]);
}

/** Promote an account to platform admin and return a token carrying the role. */
export async function platformAdmin(): Promise<Account> {
  const admin = await api.register("admin", "Platform Admin");

  // There is deliberately no endpoint that hands out this role - anybody who
  // could call it would already have to be an admin - so the grant goes
  // straight to the database, the way an operator would do it.
  const { execSync } = await import("node:child_process");
  execSync(
    `docker compose exec -T db psql -U biletflow -d biletflow -qtAX -c ` +
      `"INSERT INTO user_roles (user_id, role) VALUES ('${admin.id}', 'platform_admin') ON CONFLICT DO NOTHING;"`,
    { cwd: "..", stdio: "ignore" },
  );

  // Re-login so the token carries the new role alongside the account.
  return { ...admin, token: await api.login(admin.email) };
}

export const test = base;
export { expect };
