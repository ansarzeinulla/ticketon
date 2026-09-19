"use client";

/**
 * Holds the signed-in user for the whole app.
 *
 * The browser cannot see its own session any more: the token lives in an
 * httpOnly cookie (SRS 7), so there is nothing to read during render. The
 * provider therefore starts in "loading" and asks the server who is signed in.
 *
 * That single question is also the validity check. proxy.ts can see that a
 * cookie exists; only the API can say whether the token inside it is still
 * good, or whether the account has since been suspended.
 */

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

import { ApiError, api } from "@/lib/api";
import type { User } from "@/lib/types";

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

/** Set once the session is settled, either by asking or by an explicit action. */
interface Session {
  status: Exclude<AuthStatus, "loading">;
  user: User | null;
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(null);

  // Ask the server who is signed in.
  //
  // This runs once on mount and is the only way the browser learns about its
  // own session. Every setState happens after an await, so the effect never
  // triggers a cascading render.
  useEffect(() => {
    const controller = new AbortController();

    api
      .me(null, controller.signal)
      .then((me) => setSession({ status: "authenticated", user: me }))
      .catch((error: unknown) => {
        if (error instanceof DOMException && error.name === "AbortError") return;

        // A network blip should not sign the user out - but with no token to
        // hold on to, the honest thing is to report signed-out and let the
        // next successful call correct it.
        if (error instanceof ApiError && error.isNetworkError) {
          setSession({ status: "unauthenticated", user: null });
          return;
        }
        setSession({ status: "unauthenticated", user: null });
      });

    return () => controller.abort();
  }, []);

  const status: AuthStatus = session ? session.status : "loading";
  const user = session?.user ?? null;

  const login = useCallback(async (email: string, password: string) => {
    // The route handler sets the cookie; only the user comes back.
    const me = await api.login({ email, password });
    setSession({ status: "authenticated", user: me });
    return me;
  }, []);

  const register = useCallback(
    async (input: { email: string; password: string; full_name?: string }) => {
      const me = await api.register(input);
      setSession({ status: "authenticated", user: me });
      return me;
    },
    [],
  );

  const logout = useCallback(() => {
    // Optimistic locally, authoritative on the server: the cookie is httpOnly,
    // so only the route handler can actually remove it.
    setSession({ status: "unauthenticated", user: null });
    void api.logout().catch(() => {
      // Nothing useful to do if the sign-out request fails; the cookie expires
      // on its own, and the UI already reflects the intent.
    });
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
  if (!context) {
    throw new Error("useAuth must be used inside an AuthProvider");
  }
  return context;
}
