"use client";

import { useEffect, useRef } from "react";
import { usePathname, useRouter } from "next/navigation";
import { useAuthStore, type AuthState } from "@/store/useAuthStore";

type AuthProviderProps = {
  children: React.ReactNode;
};

export function AuthProvider({ children }: AuthProviderProps) {
  const router = useRouter();
  const pathname = usePathname();
  const bootstrap = useAuthStore((state: AuthState) => state.bootstrap);
  const isBootstrapped = useAuthStore((state: AuthState) => state.isBootstrapped);
  const isAuthenticated = useAuthStore((state: AuthState) => state.isAuthenticated);
  const bootstrapStartedRef = useRef(false);

  useEffect(() => {
    if (bootstrapStartedRef.current) {
      return;
    }
    bootstrapStartedRef.current = true;
    void bootstrap();
  }, [bootstrap]);

  useEffect(() => {
    if (!isBootstrapped) return;

    const isProtected = pathname.startsWith("/guilds") || pathname.startsWith("/channels");
    const isAuthPage = pathname === "/login" || pathname === "/register";

    if (!isAuthenticated && isProtected) {
      router.replace("/login");
      return;
    }
    if (isAuthenticated && isAuthPage) {
      router.replace("/guilds");
    }
  }, [isAuthenticated, isBootstrapped, pathname, router]);

  if (!isBootstrapped) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-zinc-950 text-zinc-400">
        Loading...
      </div>
    );
  }

  return <>{children}</>;
}
