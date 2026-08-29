"use client";

/**
 * Holds the signed-in user for the whole app.
 *
 * The token cookie is read during render via useSyncExternalStore, which is the
 * supported way to read browser-only state without tripping hydration or
 * calling setState inside an effect. Whether that token is still *valid* only
 * the API can say, so the effect confirms it with GET /auth/me. proxy.ts can
 * see that a cookie exists; it cannot see that the account was suspended.
 */

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  useSyncExternalStore,
  type ReactNode,
} from "react";

import { ApiError, api } from "@/lib/api";
import { clearToken, getToken, setToken } from "@/lib/session";
import type { AuthResponse, User } from "@/lib/types";

type AuthStatus = "loading" | "authenticated" | "unauthenticated";

interface AuthContextValue {
  user: User | null;
  status: AuthStatus;
  login: (email: string, password: string) => Promise<User>;
  register: (input: {
    email: string;
    password: string;
    full_name?: string;
  }) => Promise<User>;
  logout: () => void;
  refresh: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

/** The cookie only changes through this provider, so there is nothing to subscribe to. */
const subscribeToToken = () => () => {};

/**
 * The snapshot during SSR and the hydration render, when `document.cookie` is
 * not readable yet.
 *
 * It has to be distinct from `null`: returning null would make the first client
 * render look definitively signed out, and any route gate watching for that
 * would redirect a perfectly valid session away before hydration finished.
 * UNRESOLVED means "not known yet", which keeps the app in its loading state
 * until React re-renders with the real cookie value.
 */
const UNRESOLVED = Symbol("token-unresolved");
type TokenSnapshot = string | null | typeof UNRESOLVED;

const serverToken = (): TokenSnapshot => UNRESOLVED;

/** Set once the session is settled, either by validation or by an explicit action. */
interface Session {
  status: Exclude<AuthStatus, "loading">;
  user: User | null;
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const tokenAtMount = useSyncExternalStore<TokenSnapshot>(
    subscribeToToken,
    getToken,
    serverToken,
  );
  const [session, setSession] = useState<Session | null>(null);

  const resolvedToken = tokenAtMount === UNRESOLVED ? null : tokenAtMount;

  // Confirm the stored token with the API. Every setState here happens after an
  // await, so the effect never triggers a cascading render.
  useEffect(() => {
    if (!resolvedToken) return;

    const controller = new AbortController();

    api
      .me(resolvedToken, controller.signal)
      .then((me) => setSession({ status: "authenticated", user: me }))
      .catch((error: unknown) => {
        if (error instanceof DOMException && error.name === "AbortError") return;

        // A network blip should not sign the user out; a rejected token should.
        if (error instanceof ApiError && !error.isNetworkError) clearToken();
        setSession({ status: "unauthenticated", user: null });
      });

    return () => controller.abort();
  }, [resolvedToken]);

  // Until the session settles: loading while the cookie is unknown or a token
  // is being checked, signed out only once we have actually looked and found
  // nothing. No effect is needed for the signed-out case.
  const status: AuthStatus = session
    ? session.status
    : tokenAtMount === UNRESOLVED || tokenAtMount
      ? "loading"
      : "unauthenticated";

  const user = session?.user ?? null;

  /** Store the token from an auth response and adopt the user. */
  const adopt = useCallback((response: AuthResponse): User => {
    setToken(response.access_token, response.expires_at);
    setSession({ status: "authenticated", user: response.user });
    return response.user;
  }, []);

  const login = useCallback(
    async (email: string, password: string) => adopt(await api.login({ email, password })),
    [adopt],
  );

  const register = useCallback(
    async (input: { email: string; password: string; full_name?: string }) =>
      adopt(await api.register(input)),
    [adopt],
  );

  const logout = useCallback(() => {
    clearToken();
    setSession({ status: "unauthenticated", user: null });
  }, []);

  /** Re-read the account, e.g. after creating an event grants a new role. */
  const refresh = useCallback(async () => {
    try {
      const me = await api.me();
      setSession({ status: "authenticated", user: me });
    } catch {
      // Leave the current session alone; the next API call will surface it.
    }
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({ user, status, login, register, logout, refresh }),
    [user, status, login, register, logout, refresh],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) throw new Error("useAuth must be used inside an <AuthProvider>");
  return context;
}
