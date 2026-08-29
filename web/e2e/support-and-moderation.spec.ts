import {
  api,
  expect,
  freeEvent,
  paidEvent,
  platformAdmin,
  signIn,
  test,
  uniqueEmail,
} from "./fixtures";

/**
 * SRS 11: "An attendee can open a contextual support case and exchange
 * asynchronous messages with authorized organizer staff", and "Administrators
 * can suspend a suspicious event and stop further sales".
 * SRS 4.12 moderation, 4.13 support.
 */

test("an attendee opens a support case from their order and the organizer replies", async ({
  browser,
}) => {
  const { organizer, event, ticketTypeID } = await freeEvent("Support Lecture");
  const attendee = await api.register("supportattendee", "Nurlan Sagyndyk");
  const order = await api.checkout(
    event.id,
    ticketTypeID,
    1,
    "Nurlan Sagyndyk",
    attendee.email,
    attendee.token,
  );

  // Two contexts, because the point of SRS 4.13 is a conversation between two
  // different people who each see only what they are entitled to see.
  const attendeeContext = await browser.newContext();
  const attendeePage = await attendeeContext.newPage();
  await signIn(attendeePage, attendee.token);

  await attendeePage.goto(`/orders/${order.id}`);
  // The compose form is behind a button: an order page that opened straight
  // into a support form would suggest something had gone wrong.
  await attendeePage.getByRole("button", { name: "Contact the organizer" }).click();
  await attendeePage.getByTestId("support-category").selectOption("ticket_delivery");
  await attendeePage.locator('input[name="subject"]').fill("Ticket not received");
  await attendeePage.getByTestId("support-message").fill("My ticket has not arrived.");
  await attendeePage.getByTestId("support-submit").click();

  await expect(attendeePage.getByTestId("case-messages")).toContainText(
    "My ticket has not arrived.",
  );

  // The organizer sees it in their inbox and answers.
  const organizerContext = await browser.newContext();
  const organizerPage = await organizerContext.newPage();
  await signIn(organizerPage, organizer.token);

  await organizerPage.goto(`/dashboard/events/${event.id}`);
  await organizerPage.getByTestId("inbox-case").first().click();
  await expect(organizerPage.getByTestId("case-messages")).toContainText(
    "My ticket has not arrived.",
  );

  await organizerPage.getByTestId("reply-input").fill("Resent - please check your inbox.");
  await organizerPage.getByTestId("reply-send").click();
  await expect(organizerPage.getByTestId("case-messages")).toContainText("Resent");

  // SRS 4.13: the attendee sees the reply on their own thread.
  await attendeePage.reload();
  await attendeePage.getByTestId("open-case").first().click();
  await expect(attendeePage.getByTestId("case-messages")).toContainText("Resent");

  await attendeeContext.close();
  await organizerContext.close();
});

test("an internal note never reaches the attendee", async ({ browser }) => {
  const { organizer, event, ticketTypeID } = await freeEvent("Internal Note Lecture");
  const attendee = await api.register("noteattendee", "Olzhas Serik");
  const order = await api.checkout(
    event.id,
    ticketTypeID,
    1,
    "Olzhas Serik",
    attendee.email,
    attendee.token,
  );

  const attendeeContext = await browser.newContext();
  const attendeePage = await attendeeContext.newPage();
  await signIn(attendeePage, attendee.token);
  await attendeePage.goto(`/orders/${order.id}`);
  await attendeePage.getByRole("button", { name: "Contact the organizer" }).click();
  await attendeePage.getByTestId("support-category").selectOption("payment");
  await attendeePage.locator('input[name="subject"]').fill("Possible double charge");
  await attendeePage.getByTestId("support-message").fill("Was I charged twice?");
  await attendeePage.getByTestId("support-submit").click();
  await expect(attendeePage.getByTestId("case-messages")).toBeVisible();

  const organizerContext = await browser.newContext();
  const organizerPage = await organizerContext.newPage();
  await signIn(organizerPage, organizer.token);
  await organizerPage.goto(`/dashboard/events/${event.id}`);
  await organizerPage.getByTestId("inbox-case").first().click();

  await organizerPage.getByTestId("reply-input").fill("Checking with finance, do not send.");
  await organizerPage.getByRole("checkbox").check();
  await organizerPage.getByTestId("reply-send").click();
  await expect(organizerPage.getByTestId("case-messages")).toContainText("finance");

  // SRS 4.13: internal notes exist so staff can talk among themselves.
  await attendeePage.reload();
  await attendeePage.getByTestId("open-case").first().click();
  await expect(attendeePage.getByTestId("case-messages")).not.toContainText("finance");

  await attendeeContext.close();
  await organizerContext.close();
});

test("a platform admin suspends an event and it stops selling", async ({ page }) => {
  const admin = await platformAdmin();
  const { event, title } = await paidEvent("Suspicious Gala");

  await signIn(page, admin.token);
  await page.goto("/admin");
  await page.getByRole("searchbox").fill(title);

  // Scoped to the Events section: the activation-fee payment carries the same
  // title as its reference, so an unscoped row locator matches twice.
  const eventsSection = page.locator("section", { hasText: "Events" }).last();
  const row = eventsSection.locator("tr", { hasText: title });
  await expect(row).toBeVisible();

  // SRS 11: "Administrators can suspend a suspicious event and stop further
  // sales." Every moderation action asks first and records a reason.
  await row.getByRole("button", { name: "Suspend", exact: true }).click();
  await row.getByRole("textbox").fill("Reported as misleading");
  await row.getByRole("button", { name: "Suspend this event" }).click();

  await expect(row.getByRole("button", { name: "Lift suspension" })).toBeVisible();

  // The public page says so, and the ticket selector is gone.
  await page.goto(`/events/${event.slug}`);
  await expect(page.getByTestId("suspended-banner")).toBeVisible();
  await expect(page.getByRole("button", { name: "Get tickets" })).toHaveCount(0);
});

test("a platform admin suspends a user, whose session ends at once", async ({ browser }) => {
  const admin = await platformAdmin();
  const target = await api.register("suspendme", "Rogue Organizer");

  const targetContext = await browser.newContext();
  const targetPage = await targetContext.newPage();
  await signIn(targetPage, target.token);
  await targetPage.goto("/dashboard");
  await expect(targetPage.getByRole("heading", { name: "Your events" })).toBeVisible();

  const adminContext = await browser.newContext();
  const adminPage = await adminContext.newPage();
  await signIn(adminPage, admin.token);
  await adminPage.goto("/admin");
  await adminPage.getByRole("searchbox").fill(target.email);

  const row = adminPage.locator("tr", { hasText: target.email });
  await row.getByRole("button", { name: "Suspend", exact: true }).click();
  await row.getByRole("textbox").fill("Policy violation");
  await row.getByRole("button", { name: "Suspend this account" }).click();
  await expect(row.getByRole("button", { name: "Restore" })).toBeVisible();

  // SRS 4.12: the effect is immediate, because every authorised request
  // re-reads the account rather than trusting the token.
  await targetPage.goto("/dashboard");
  await expect(targetPage).toHaveURL(/\/login/);

  await targetContext.close();
  await adminContext.close();
});

test("the admin portal is refused to an ordinary account", async ({ page }) => {
  const ordinary = await api.register("notadmin");
  await signIn(page, ordinary.token);

  await page.goto("/admin");

  // The page says so plainly; the API is the actual boundary.
  await expect(page.getByText(/platform administrator/i).first()).toBeVisible();
  await expect(page.getByRole("searchbox")).toHaveCount(0);
});

test("an attendee reports an event and it reaches the moderation queue", async ({
  browser,
}) => {
  const admin = await platformAdmin();
  const { event, title } = await freeEvent("Reportable Lecture");
  const reporter = await api.register("reporter");

  // SRS 4.12: "Review reported events" needs something to review. Filed
  // through the API, since the report control lives on the public page.
  const response = await fetch(
    `${process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080/api/v1"}/events/${event.id}/report`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${reporter.token}`,
      },
      body: JSON.stringify({ reason: "misleading", details: "The venue does not exist." }),
    },
  );
  expect(response.status).toBe(201);

  const adminContext = await browser.newContext();
  const adminPage = await adminContext.newPage();
  await signIn(adminPage, admin.token);
  await adminPage.goto("/admin/reports");

  const row = adminPage.locator("li, tr", { hasText: title }).first();
  await expect(row).toBeVisible();
  await expect(row).toContainText(/misleading/i);

  await adminContext.close();
  expect(uniqueEmail("unused")).toContain("@");
});
