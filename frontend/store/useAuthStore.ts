"use client";

import { create } from "zustand";
import { toast } from "sonner";

export type AuthState = {
  token: string | null;
  userId: string | null;
  isAuthenticated: boolean;
  bootstrap: () => void;
  login: (email: string, password: string) => Promise<boolean>;
  register: (email: string, password: string) => Promise<boolean>;
  logout: () => Promise<void>;
};

export const useAuthStore = create<AuthState>((set, get) => ({
  token: null,
  userId: null,
  isAuthenticated: false,
  bootstrap: () => {
    set((state) => ({ ...state }));
  },
  login: async (email, password) => {
    try {
      const response = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ email, password }),
      });

      const data = await response.json().catch(() => null);
      if (!response.ok) throw new Error(data?.error ?? "login_failed");

      const token = data?.token as string | undefined;
      set({ token: token ?? null, userId: data?.user?.id ?? null, isAuthenticated: Boolean(token) });
      return true;
    } catch {
      toast.error("Login failed");
      return false;
    }
  },
  register: async (email, password) => {
    try {
      const response = await fetch("/api/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ email, password }),
      });
      if (!response.ok) throw new Error("register_failed");
      toast.success("Account created");

      return await get().login(email, password);
    } catch {
      toast.error("Registration failed");
      return false;
    }
  },
  logout: async () => {
    try {
      await fetch("/api/auth/logout", { method: "POST", credentials: "include" });
    } catch {
      // ignore network errors during logout
    }
    set({ token: null, userId: null, isAuthenticated: false });
  },
}));
