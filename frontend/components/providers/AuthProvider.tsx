"use client";

import { useEffect } from "react";
import { useAuthStore, type AuthState } from "@/store/useAuthStore";

type AuthProviderProps = {
  children: React.ReactNode;
};

export function AuthProvider({ children }: AuthProviderProps) {
  const bootstrap = useAuthStore((state: AuthState) => state.bootstrap);

  useEffect(() => {
    bootstrap();
  }, [bootstrap]);

  return <>{children}</>;
}
