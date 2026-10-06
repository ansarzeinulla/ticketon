import { defineConfig, devices } from "@playwright/test";

/**
 * End-to-end tests against the real stack (SRS 9: "Automated API, interface,
 * payment, and QR-validation tests").
 *
 * Nothing is mocked. The suite drives the Next.js app in a browser, and that
 * app talks to the real Go API, which talks to the real PostgreSQL — the same
 * three processes a person uses. A suite that stubbed the API would prove the
 * components render, not that the platform works.
 *
 * Consequently it needs all three running:
 *   make up && make api-run && make web-dev
 *
 * Workers are limited to two. The specs create their own accounts and events
 * so they do not collide over data, but they do share one API process and one
 * database, and a stampede of parallel checkouts makes failures read as
 * flakiness rather than as bugs.
 */
export default defineConfig({
  testDir: "./e2e",
  outputDir: "./e2e/.results",
  fullyParallel: true,
  workers: process.env.CI ? 2 : 2,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
  timeout: 45_000,
  expect: { timeout: 10_000 },

  use: {
    baseURL: process.env.E2E_BASE_URL ?? "http://localhost:3000",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "off",
  },

  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
  ],
});
