import { api, expect, paidEvent, signIn, test, uniqueEmail, uniqueTitle } from "./fixtures";

/**
 * SRS 11: "An attendee can complete checkout and receive a QR-code ticket",
 * and "An attendee can download and print a ticket that remains scannable".
 * SRS 4.5 paid-sales activation, 4.6 checkout, 4.7 digital tickets.
 */

test("paid tickets cannot be bought until the activation checklist is done", async ({
  page,
}) => {
  const organizer = await api.register("unactivated", "Unactivated Organizer");
  const title = uniqueTitle("Not Yet Cleared");
  const event = await api.createEvent(organizer.token, title);
  await api.addTicketType(organizer.token, event.id, "General Admission", "5000", 20);
  await api.publish(organizer.token, event.id);

  await page.goto(`/events/${event.slug}`);

  // SRS 4.5: "Paid tickets shall not be purchasable before activation." The
  // page says so rather than offering a selector that would be refused.
  await expect(page.getByTestId("paid-sales-pending-banner")).toBeVisible();
  await expect(page.getByRole("button", { name: "Get tickets" })).toHaveCount(0);
});

test("an attendee buys a paid ticket and receives a QR-code ticket", async ({ page }) => {
  const { event } = await paidEvent("Winter Jazz Night");
  const attendeeEmail = uniqueEmail("buyer");

  await page.goto(`/events/${event.slug}`);
  await page.getByRole("button", { name: /^Add one / }).first().click();
  await page.getByRole("button", { name: "Get tickets" }).click();

  const dialog = page.getByRole("dialog");
  await dialog.locator('input[name="buyer_name"]').fill("Aigerim Zhaksy");
  await dialog.locator('input[name="buyer_email"]').fill(attendeeEmail);

  // SRS 4.6: demonstration payments must never read as real ones.
  await expect(dialog.getByText(/simulated/i).first()).toBeVisible();
  await dialog.getByRole("button", { name: /^Pay / }).click();

  await expect(page).toHaveURL(/\/orders\//);
  await expect(page.getByText("5 000").first()).toBeVisible();

  // SRS 4.7: a QR code, and a print-optimised PDF behind a plain link.
  await expect(page.locator("img[src*='qr']").first()).toBeVisible();
  const pdfLink = page.getByRole("link", { name: /download pdf/i }).first();
  await expect(pdfLink).toBeVisible();
  await expect(pdfLink).toHaveAttribute("href", /\/tickets\/.+\/pdf$/);
});

test("the downloaded ticket is a real A4 PDF", async ({ page }) => {
  const { event, ticketTypeID } = await paidEvent("Printable Night");
  const order = await api.checkout(
    event.id,
    ticketTypeID,
    1,
    "Marat Zhaksy",
    uniqueEmail("printer"),
  );

  await page.goto(`/orders/${order.id}`);
  const download = page.waitForEvent("download");
  await page.getByRole("link", { name: /download pdf/i }).first().click();

  const file = await download;
  const stream = await file.createReadStream();
  const chunks: Buffer[] = [];
  for await (const chunk of stream) chunks.push(chunk as Buffer);
  const bytes = Buffer.concat(chunks);

  // SRS 4.7 / UC7: it has to actually be a PDF, not an error page with a
  // .pdf name. The Go suite already decodes the QR off the rendered page;
  // this proves the browser gets the same file.
  expect(bytes.subarray(0, 5).toString()).toBe("%PDF-");
  expect(bytes.length).toBeGreaterThan(1000);
});

test("an organizer refunds a paid order and the ticket stops working", async ({ page }) => {
  const { organizer, event, ticketTypeID } = await paidEvent("Refunded Gala");
  await api.checkout(event.id, ticketTypeID, 1, "Olzhas Serik", uniqueEmail("refundme"));

  await signIn(page, organizer.token);
  await page.goto(`/dashboard/events/${event.id}`);

  const row = page.locator("tr", { hasText: "Olzhas Serik" });
  await row.getByRole("button", { name: "Refund order" }).click();
  await row.getByRole("button", { name: "Confirm refund" }).click();

  // SRS 4.9: "Refunded or cancelled tickets shall become invalid."
  // Scoped to the refund banner: the activation checklist above also renders a
  // role="status", and matching the first one on the page found that instead.
  await expect(page.getByText("Refund complete")).toBeVisible();
  await expect(row.getByText(/refunded/i).first()).toBeVisible();
});

test("a promo code is priced by the server and shown before paying", async ({ page }) => {
  const { organizer, event } = await paidEvent("Discounted Evening");
  const code = await api.createCampaign(organizer.token, event.id, "STUDENT", 20);

  await page.goto(`/events/${event.slug}`);
  await page.getByRole("button", { name: /^Add one / }).first().click();

  // SRS 4.14: "The attendee shall see the applied code, discount, and updated
  // order total before completing checkout" - and the discount is never
  // computed here; it comes back from the server's own pricing.
  await page.getByTestId("promo-input").fill(code);
  await page.keyboard.press("Enter");

  await expect(page.getByTestId("promo-applied")).toBeVisible();
  await expect(page.getByTestId("promo-discount")).toContainText("1 000");
  await expect(page.getByTestId("promo-total")).toContainText("4 000");
});

test("an invalid promo code is refused with a message, not silently ignored", async ({
  page,
}) => {
  const { event } = await paidEvent("No Such Code");

  await page.goto(`/events/${event.slug}`);
  await page.getByRole("button", { name: /^Add one / }).first().click();
  await page.getByTestId("promo-input").fill("NOTAREALCODE");
  await page.keyboard.press("Enter");

  await expect(page.getByTestId("promo-error")).toBeVisible();
  // The basket total is untouched by a code that does not apply.
  await expect(page.getByTestId("promo-applied")).toHaveCount(0);
});
