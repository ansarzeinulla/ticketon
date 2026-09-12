import { PASSWORD, api, expect, signIn, test, uniqueEmail } from "./fixtures";

/**
 * SRS 4.1: registration, sign-in, sign-out, password reset and change.
 *
 * Fields are addressed by their `name` attribute rather than their label. The
 * shared FieldShell appends a required marker to every label, so the accessible
 * name of the password input is "Password *" — matching on the name attribute
 * says what is meant and does not break when that marker changes.
 */

test("a visitor registers, lands on the dashboard, and signs out", async ({ page }) => {
  const email = uniqueEmail("register");

  await page.goto("/register");
  await page.locator('input[name="full_name"]').fill("Dana Amirova");
  await page.locator('input[name="email"]').fill(email);
  await page.locator('input[name="password"]').fill(PASSWORD);
  await page.locator('input[name="confirm_password"]').fill(PASSWORD);
  await page.getByRole("button", { name: /create account/i }).click();

  await expect(page).toHaveURL(/\/dashboard/);
  await expect(page.getByRole("heading", { name: "Your events" })).toBeVisible();

  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page).toHaveURL(/\/login/);
});

test("signing in with the wrong password says so without revealing anything", async ({
  page,
}) => {
  const account = await api.register("wrongpass");

  await page.goto("/login");
  await page.locator('input[name="email"]').fill(account.email);
  await page.locator('input[name="password"]').fill("not the right password");
  await page.getByRole("button", { name: "Sign in" }).click();

  // The message must not distinguish "no such account" from "wrong password",
  // or the form becomes a way to enumerate who is registered.
  await expect(page.getByRole("alert").first()).toContainText(/incorrect/i);
  await expect(page).toHaveURL(/\/login/);
});

test("the dashboard is not reachable without signing in", async ({ page }) => {
  await page.goto("/dashboard");

  // proxy.ts redirects before the page renders, and carries the destination so
  // signing in returns you where you were going.
  await expect(page).toHaveURL(/\/login\?next=%2Fdashboard/);
});

test("forgotten password answers identically whether or not the account exists", async ({
  page,
}) => {
  const known = await api.register("forgot");

  for (const email of [known.email, "definitely-not-registered@biletflow.test"]) {
    await page.goto("/forgot-password");
    await page.locator('input[name="email"]').fill(email);
    await page.getByRole("button", { name: /send|reset/i }).click();

    // Identical wording either way: SRS 4.1's reset must not enumerate users.
    await expect(page.getByRole("heading", { name: /check your inbox/i })).toBeVisible();
  }
});

test("an organizer changes their own password and the old one stops working", async ({
  page,
}) => {
  const account = await api.register("changepass");
  await signIn(page, account.token);

  await page.goto("/dashboard/profile");
  await page.locator('input[name="current_password"]').fill(PASSWORD);
  await page.locator('input[name="new_password"]').fill("an entirely different passphrase");
  await page.locator('input[name="confirm_password"]').fill("an entirely different passphrase");
  await page.getByTestId("change-password").click();

  await expect(page.getByRole("status").first()).toContainText(/changed/i);

  // Proven by using it, not by trusting the banner: SRS 4.1 is about the
  // credential actually changing, not about a message saying it did.
  const refused = await fetch("http://localhost:8080/api/v1/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email: account.email, password: PASSWORD }),
  });
  expect(refused.status).toBe(401);
});

test("an organizer fills in their profile and it survives a reload", async ({ page }) => {
  const account = await api.register("profile");
  await signIn(page, account.token);

  await page.goto("/dashboard/profile");
  await page.locator('input[name="display_name"]').fill("Dana Events");
  await page.locator('input[name="contact_email"]').fill("hello@danaevents.kz");
  await page.locator('input[name="website_url"]').fill("https://danaevents.kz");
  await page.getByTestId("save-profile").click();

  await expect(page.getByRole("status").first()).toContainText(/saved/i);

  await page.reload();
  await expect(page.locator('input[name="display_name"]')).toHaveValue("Dana Events");
  await expect(page.locator('input[name="contact_email"]')).toHaveValue(
    "hello@danaevents.kz",
  );
});

test("the profile refuses an address that is not one, on the field itself", async ({
  page,
}) => {
  const account = await api.register("profileinvalid");
  await signIn(page, account.token);

  await page.goto("/dashboard/profile");
  await page.locator('input[name="contact_email"]').fill("not-an-address");
  await page.getByTestId("save-profile").click();

  // The API's 422 field map is rendered on the input that caused it.
  const field = page.locator('input[name="contact_email"]');
  await expect(field).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByText(/valid email address/i)).toBeVisible();
});
