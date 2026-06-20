"use client";

import { create } from "zustand";
import { toast } from "sonner";

type AuthUser = {
  id: string;
  username: string;
  email: string;
};

export type AuthState = {
  token: string | null;
  userId: string | null;
  user: AuthUser | null;
  isAuthenticated: boolean;
  isBootstrapped: boolean;
  bootstrap: () => Promise<void>;
  login: (email: string, password: string) => Promise<boolean>;
  register: (username: string, email: string, password: string) => Promise<boolean>;
  logout: () => Promise<void>;
};

function applySession(
  set: (partial: Partial<AuthState>) => void,
  token: string | null,
  user: AuthUser | null,
) {
  set({
    token,
    user,
    userId: user?.id ?? null,
    isAuthenticated: Boolean(token),
    isBootstrapped: true,
  });
}

function readApiError(data: { message?: string; error?: string } | null, fallback: string) {
  return data?.message ?? data?.error ?? fallback;
}

let bootstrapInFlight: Promise<void> | null = null;

export const useAuthStore = create<AuthState>((set, get) => ({
  token: null,
  userId: null,
  user: null,
  isAuthenticated: false,
  isBootstrapped: false,
  bootstrap: async () => {
    if (get().isBootstrapped) {
      return;
    }
    if (bootstrapInFlight) {
      await bootstrapInFlight;
      return;
    }

    bootstrapInFlight = (async () => {
      try {
        const response = await fetch("/api/auth/session", { credentials: "include" });
        if (!response.ok) {
          applySession(set, null, null);
          return;
        }
        const data = await response.json();
        applySession(set, data.token ?? null, data.user ?? null);
      } catch {
        applySession(set, null, null);
      } finally {
        bootstrapInFlight = null;
      }
    })();

    await bootstrapInFlight;
  },
  login: async (email, password) => {
    try {
      const response = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ email: email.trim(), password }),
      });

      const data = await response.json().catch(() => null);
      if (!response.ok) throw new Error(readApiError(data, "login_failed"));

      const token = data?.token as string | undefined;
      applySession(set, token ?? null, data?.user ?? null);
      return true;
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Login failed");
      return false;
    }
  },
  register: async (username, email, password) => {
    try {
      const response = await fetch("/api/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          username: username.trim(),
          email: email.trim(),
          password,
        }),
      });

      const data = await response.json().catch(() => null);
      if (!response.ok) throw new Error(readApiError(data, "register_failed"));

      const token = data?.token as string | undefined;
      applySession(set, token ?? null, data?.user ?? null);
      toast.success("Account created");
      return true;
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Registration failed");
      return false;
    }
  },
  logout: async () => {
    try {
      await fetch("/api/auth/logout", { method: "POST", credentials: "include" });
    } catch {
      // ignore network errors during logout
    }
    applySession(set, null, null);
  },
}));
