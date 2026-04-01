"use client";

import Link from "next/link";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore, type AuthState } from "@/store/useAuthStore";

export default function RegisterPage() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const register = useAuthStore((state: AuthState) => state.register);
  const router = useRouter();

  const onSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const ok = await register(email, password);
    if (ok) router.push("/guilds");
  };

  return (
    <main className="flex min-h-screen items-center justify-center bg-zinc-950 p-6">
      <form onSubmit={onSubmit} className="w-full max-w-sm space-y-4 rounded-lg bg-zinc-900 p-6">
        <h1 className="text-xl font-semibold text-white">Register</h1>
        <input
          className="w-full rounded-md border border-zinc-700 bg-zinc-800 p-2 text-white"
          placeholder="Email"
          type="email"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
        />
        <input
          className="w-full rounded-md border border-zinc-700 bg-zinc-800 p-2 text-white"
          placeholder="Password"
          type="password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
        />
        <button className="w-full rounded-md bg-indigo-600 p-2 font-medium text-white" type="submit">
          Create account
        </button>
        <p className="text-sm text-zinc-300">
          Already have an account?{" "}
          <Link href="/login" className="text-indigo-400">
            Login
          </Link>
        </p>
      </form>
    </main>
  );
}
