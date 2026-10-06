import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /**
   * Standalone output for the container image (SRS 8).
   *
   * Next traces which of node_modules the server actually reaches and copies
   * only those into .next/standalone, so the runtime image carries the server
   * and its real dependencies rather than the whole dependency tree. It has no
   * effect on `next dev`.
   */
  output: "standalone",
};

export default nextConfig;
