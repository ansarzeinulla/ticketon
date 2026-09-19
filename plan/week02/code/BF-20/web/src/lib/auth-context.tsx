"use client";

/**
 * Holds the signed-in user for the whole app.
 *
 * The token lives in a cookie (see session.ts). On mount the provider reads it
 * and asks the API who it belongs to: a cookie can outlive the token inside
 * it, and only the API can say whether it is still good.
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
import { clearToken, getToken, setToken } from "@/lib/session";
import type { AuthResponse, User } from "@/lib/types";

type AuthStatus = "loading" | "authenticated" | "unauthenticated";

interface AuthContextValue {
  user: User | null;
  status: AuthStatus;
  login: (email: string, password: string) => Promise<User>;
  register: (input: { email: string; password: string; full_name?: string }) => Promise<User>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

interface Session {
  status: Exclude<AuthStatus, "loading">;
  user: User | null;
}

const signedOut: Session = { status: "unauthenticated", user: null };

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(null);

  useEffect(() => {
    const controller = new AbortController();

    // Async so every setState lands after the effect has returned.
    void (async () => {
      const token = getToken();
      if (!token) {
        setSession(signedOut);
        return;
      }
      try {
        const me = await api.me(token, controller.signal);
        setSession({ status: "authenticated", user: me });
      } catch (error) {
        if (error instanceof DOMException && error.name === "AbortError") return;
        // A rejected token is stale: drop it so the app stops pretending.
        if (error instanceof ApiError && error.status === 401) clearToken();
        setSession(signedOut);
      }
    })();

    return () => controller.abort();
  }, []);

  const signIn = useCallback((auth: AuthResponse) => {
    setToken(auth.access_token, auth.expires_at);
    setSession({ status: "authenticated", user: auth.user });
    return auth.user;
  }, []);

  const login = useCallback(
    async (email: string, password: string) => signIn(await api.login({ email, password })),
    [signIn],
  );

  const register = useCallback(
    async (input: { email: string; password: string; full_name?: string }) =>
      signIn(await api.register(input)),
    [signIn],
  );

  const logout = useCallback(() => {
    clearToken();
    setSession(signedOut);
  }, []);

  const status: AuthStatus = session ? session.status : "loading";
  const user = session?.user ?? null;

  const value = useMemo<AuthContextValue>(
    () => ({ user, status, login, register, logout }),
    [user, status, login, register, logout],
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
