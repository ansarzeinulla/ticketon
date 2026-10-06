import path from "node:path";


import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

/**
 * Unit tests for the pure logic the end-to-end suite covers poorly.
 *
 * Playwright proves the flows work; these prove the arithmetic and the
 * timezone conversion are right for inputs a browser walkthrough would never
 * think to try - a half-tiyn rounding, a DST boundary, a null venue.
 */
export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    include: ["src/**/*.test.ts", "src/**/*.test.tsx"],
    setupFiles: ["./vitest.setup.ts"],
  },
  resolve: {
    alias: { "@": path.resolve(import.meta.dirname, "./src") },
  },
});
