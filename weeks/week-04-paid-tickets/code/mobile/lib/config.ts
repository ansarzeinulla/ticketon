import Constants from "expo-constants";

/**
 * Where the Go API lives.
 *
 * A phone on the same Wi-Fi cannot reach "localhost" - that would be the phone
 * itself. Expo already knows the development machine's LAN address, because
 * that is how the app was delivered, so the API host is derived from it. The
 * scanner therefore works on a real device with no configuration at all.
 *
 * Set EXPO_PUBLIC_API_BASE_URL to point somewhere else (a staging server, or a
 * tunnel) and it wins.
 */
function resolveBaseURL(): string {
  const explicit = process.env.EXPO_PUBLIC_API_BASE_URL;
  if (explicit) return explicit.replace(/\/+$/, "");

  // hostUri looks like "192.168.1.24:8081" or "localhost:8081".
  const hostUri =
    Constants.expoConfig?.hostUri ??
    (Constants.expoGoConfig as { debuggerHost?: string } | undefined)?.debuggerHost;

  const host = hostUri?.split(":")[0];
  if (host) return `http://${host}:8080/api/v1`;

  return "http://localhost:8080/api/v1";
}

export const API_BASE_URL = resolveBaseURL();

/** Shown on the login screen so a mis-pointed build is obvious immediately. */
export const API_HOST_LABEL = API_BASE_URL.replace(/^https?:\/\//, "").replace("/api/v1", "");
