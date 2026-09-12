import { api, expect, freeEvent, signIn, test, uniqueEmail, uniqueTitle } from "./fixtures";

/**
 * SRS 11: "An organizer can publish a free event and distribute valid tickets."
 * SRS 4.4 free registration, SRS 1.2 discovery, and SRS 4.9's cancellation.
 */

test("an organizer creates, publishes and unpublishes a free event", async ({ page }) => {
  const organizer = await api.register("freeorg", "Dana Amirova");
  await signIn(page, organizer.token);

  const title = uniqueTitle("Almaty Open Lecture");

  await page.goto("/events/new");
  await page.locator('input[name="title"]').fill(title);
  await page.getByRole("button", { name: "Create event" }).click();

  await expect(page).toHaveURL(/\/dashboard/);
  const card = page.locator("li", { hasText: title });
  await expect(card).toBeVisible();

  await card.getByRole("button", { name: "Publish" }).click();

  // The badge prints the raw status and capitalises it in CSS, so the assertion
  // is on the action that becomes available - which is the behaviour anyway.
  await expect(card.getByTestId("unpublish-event")).toBeVisible();
  await expect(card.getByRole("link", { name: "Public page" })).toBeVisible();

  // SRS 4.2 lists unpublish alongside publish: it is the reversible way to
  // take a page down, as opposed to cancelling, which is final.
  await card.getByTestId("unpublish-event").click();
  await expect(card.getByRole("button", { name: "Publish" })).toBeVisible();
  await expect(card.getByRole("link", { name: "Public page" })).toHaveCount(0);
});

test("an attendee finds a free event in the catalogue and registers", async ({ page }) => {
  // The catalogue is ordered soonest-first, so this event is given a start
  // date early enough to land on the first page however many others exist.
  const { event, title } = await freeEvent("Catalogue Free Lecture", {
    starts_at: "2026-10-01T15:00:00Z",
    ends_at: "2026-10-01T19:00:00Z",
  });
  const attendeeEmail = uniqueEmail("attendee");

  // SRS 1.2: attendees discover events. The catalogue is the only route to an
  // event for somebody who was not sent a direct link.
  await page.goto("/events");
  await page.getByRole("link", { name: title }).click();
  await expect(page).toHaveURL(new RegExp(`/events/${event.slug}`));

  await page.getByRole("button", { name: /^Add one / }).first().click();
  await page.getByRole("button", { name: "Get tickets" }).click();

  const dialog = page.getByRole("dialog");
  await dialog.locator('input[name="buyer_name"]').fill("Nurlan Sagyndyk");
  await dialog.locator('input[name="buyer_email"]').fill(attendeeEmail);
  await dialog.getByRole("button", { name: /^Pay / }).click();

  // SRS 4.4: a zero-value order, a confirmation, and a QR-code ticket.
  await expect(page).toHaveURL(/\/orders\//);
  await expect(page.locator("img[src*='qr']").first()).toBeVisible();
});

test("an organizer cancels a free registration and the place goes back on sale", async ({
  page,
}) => {
  const { organizer, event, ticketTypeID } = await freeEvent("Cancellable Lecture");
  await api.checkout(event.id, ticketTypeID, 1, "Olzhas Serik", uniqueEmail("cancelme"));

  await signIn(page, organizer.token);
  await page.goto(`/dashboard/events/${event.id}`);

  // SRS 4.9: "Organizers shall be able to cancel free registrations." Before
  // this existed, the only button offered was Refund, and it returned a 500.
  const row = page.locator("tr", { hasText: "Olzhas Serik" });
  await expect(row).toBeVisible();
  await expect(row.getByRole("button", { name: "Cancel registration" })).toBeVisible();
  await row.getByRole("button", { name: "Cancel registration" }).click();
  await row.getByRole("button", { name: "Cancel this registration" }).click();

  await expect(page.getByRole("status").first()).toContainText(/cancelled/i);

  // The place came back: 50 total, 1 sold then cancelled, so 50 available.
  await page.reload();
  const counts = page.getByTestId("ticket-type-counts").first();
  await expect(counts).toContainText("Available");
  await expect(counts).toContainText("50");
});

test("a free order is never offered a refund, because there is nothing to refund", async ({
  page,
}) => {
  const { organizer, event, ticketTypeID } = await freeEvent("No Refund Here");
  await api.checkout(event.id, ticketTypeID, 1, "Aigerim Zhaksy", uniqueEmail("norefund"));

  await signIn(page, organizer.token);
  await page.goto(`/dashboard/events/${event.id}`);

  const row = page.locator("tr", { hasText: "Aigerim Zhaksy" });
  await expect(row.getByRole("button", { name: "Refund order" })).toHaveCount(0);
  await expect(row.getByRole("button", { name: "Cancel registration" })).toBeVisible();
});
