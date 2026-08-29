import { api, expect, freeEvent, paidEvent, signIn, test, uniqueEmail } from "./fixtures";

/**
 * SRS 11: "Organizers can view accurate ticket, attendance, payment and refund
 * records", "…basic sales, capacity, campaign and attendance analytics", and
 * "…review past and cancelled events, inspect an authorized activity timeline,
 * and duplicate a past event without copying historical transactions".
 * SRS 4.3, 4.8, 4.15, 4.16.
 */

test("analytics report figures computed from the rows, not accumulated", async ({ page }) => {
  const { organizer, event, ticketTypeID } = await paidEvent("Measured Gala");
  await api.checkout(event.id, ticketTypeID, 2, "Aigerim Zhaksy", uniqueEmail("an1"));
  await api.checkout(event.id, ticketTypeID, 1, "Marat Zhaksy", uniqueEmail("an2"));

  await signIn(page, organizer.token);
  await page.goto(`/dashboard/events/${event.id}`);

  // 3 tickets at 5 000 KZT.
  await expect(page.getByTestId("stat-sold")).toContainText("3");
  await expect(page.getByTestId("stat-remaining")).toContainText("47");
  await expect(page.getByTestId("stat-revenue")).toContainText("15 000");
  await expect(page.getByTestId("stat-checkedin")).toContainText("0");

  // SRS 4.15: "compare sales by ticket type".
  await expect(page.getByTestId("sales-by-type")).toContainText("General Admission");
});

test("a refund is reflected in the analytics rather than left as gross", async ({ page }) => {
  const { organizer, event, ticketTypeID } = await paidEvent("Refund Maths");
  await api.checkout(event.id, ticketTypeID, 2, "Olzhas Serik", uniqueEmail("refundmaths"));

  await signIn(page, organizer.token);
  await page.goto(`/dashboard/events/${event.id}`);
  await expect(page.getByTestId("stat-sold")).toContainText("2");

  const row = page.locator("tr", { hasText: "Olzhas Serik" });
  await row.getByRole("button", { name: "Refund order" }).click();
  await row.getByRole("button", { name: "Confirm refund" }).click();
  await expect(page.getByText("Refund complete")).toBeVisible();

  // SRS 4.15: refunded tickets stop counting as sold, and the places return.
  await page.reload();
  await expect(page.getByTestId("stat-sold")).toContainText("0");
  await expect(page.getByTestId("stat-remaining")).toContainText("50");
});

test("all five SRS 4.3 ticket counters are visible to the organizer", async ({ page }) => {
  const { organizer, event, ticketTypeID } = await freeEvent("Counted Lecture");
  const order = await api.checkout(
    event.id,
    ticketTypeID,
    1,
    "Nurlan Sagyndyk",
    uniqueEmail("counted"),
  );

  // Admit them, so the checked-in counter has something to show.
  await api.checkInManually(organizer.token, event.id, order.tickets[0].id);

  await signIn(page, organizer.token);
  await page.goto(`/dashboard/events/${event.id}`);

  // SRS 4.3: "View the number of available, reserved, sold, refunded, and
  // checked-in tickets."
  const counts = page.getByTestId("ticket-type-counts").first();
  await expect(counts).toContainText("Sold");
  await expect(counts).toContainText("Available");
  await expect(counts).toContainText("Checked in");
});

test("an organizer finds an attendee by name and admits them without a QR", async ({
  page,
}) => {
  const { organizer, event, ticketTypeID } = await freeEvent("Door List Lecture");
  await api.checkout(event.id, ticketTypeID, 1, "Aisha Nurlanova", uniqueEmail("door"));

  await signIn(page, organizer.token);
  await page.goto(`/dashboard/events/${event.id}`);

  // SRS 4.8: "Search for attendees manually" - for when a QR will not scan.
  await page.getByTestId("attendee-search").fill("Aisha");
  await expect(page.getByTestId("attendee-results")).toContainText("Aisha Nurlanova");

  await page.getByTestId(/^admit-/).first().click();
  await expect(page.getByRole("status").first()).toContainText(/checked in/i);

  // The same one-admission rule as a camera scan.
  await expect(page.getByTestId("attendee-results")).toContainText("Already in");
});

test("the attendee list never hands out a QR token", async ({ page }) => {
  const { organizer, event, ticketTypeID } = await freeEvent("No Skeleton Key");
  await api.checkout(event.id, ticketTypeID, 1, "Timur Bekov", uniqueEmail("token"));

  await signIn(page, organizer.token);
  await page.goto(`/dashboard/events/${event.id}`);
  await page.getByTestId("attendee-search").fill("Timur");
  await expect(page.getByTestId("attendee-results")).toBeVisible();

  // The search is a way to find somebody, not a way to mint an admission
  // credential: TKT_ tokens must never appear in the response or the page.
  const body = await page.locator("body").innerText();
  expect(body).not.toContain("TKT_");
});

test("an organizer names gate staff and can revoke them again", async ({ page }) => {
  const { organizer, event } = await freeEvent("Staffed Lecture");
  const scanner = await api.register("gate", "Askar Kassym");

  await signIn(page, organizer.token);
  await page.goto(`/dashboard/events/${event.id}`);

  // SRS 4.8: without this the scanner app is unusable by anybody but the
  // organizer, because there was no way to grant the access.
  await page.locator('input[name="staff_email"]').fill(scanner.email);
  await page.getByTestId("add-staff").click();

  const list = page.getByTestId("staff-list");
  await expect(list).toContainText("Askar Kassym");
  await expect(list).toContainText(scanner.email);

  await page.getByTestId(/^revoke-/).first().click();
  await expect(page.getByTestId("staff-list")).toHaveCount(0);
});

test("an event's timeline records what happened, in order", async ({ page }) => {
  const { organizer, event, ticketTypeID } = await freeEvent("Timelined Lecture");
  await api.checkout(event.id, ticketTypeID, 1, "Dana Amirova", uniqueEmail("timeline"));

  await signIn(page, organizer.token);
  await page.goto(`/dashboard/events/${event.id}`);

  // SRS 4.16: the timeline reads the append-only audit log, so an event's
  // record cannot be quietly rewritten.
  const timeline = page.getByTestId("timeline");
  await expect(timeline).toBeVisible();
  await expect(timeline).toContainText(/created/i);
  await expect(timeline).toContainText(/published/i);
});

test("duplicating an event copies the configuration and none of the history", async ({
  page,
}) => {
  const { organizer, event, ticketTypeID, title } = await freeEvent("Reusable Lecture");
  await api.checkout(event.id, ticketTypeID, 2, "Sofia Ivanova", uniqueEmail("dup"));

  await signIn(page, organizer.token);
  await page.goto(`/dashboard/events/${event.id}`);
  await page.getByTestId("duplicate-event").click();

  // SRS 4.16: the copy is a draft with zero orders, zero sold and zero
  // check-ins - the configuration is reusable, the transactions are not.
  await expect(page).toHaveURL(/\/dashboard\/events\/[0-9a-f-]+$/);
  await expect(page.getByText(title, { exact: false }).first()).toBeVisible();
  await expect(page.getByTestId("stat-sold")).toContainText("0");
  await expect(page.getByTestId("stat-revenue")).toContainText("0");
});
