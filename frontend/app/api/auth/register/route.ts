import { NextResponse } from "next/server";
import { setAuthCookie } from "@/lib/auth-cookie";
import { getBackendUrl } from "@/lib/server-backend";

export async function POST(request: Request) {
  const backendUrl = getBackendUrl();
  const body = await request.json();

  const upstream = await fetch(`${backendUrl}/api/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });

  const data = await upstream.json().catch(() => null);
  const response = NextResponse.json(data ?? { error: "invalid_upstream" }, { status: upstream.status });

  if (!upstream.ok) {
    return response;
  }

  const accessToken = data?.token as string | undefined;
  if (accessToken) {
    setAuthCookie(response, accessToken);
  }

  return response;
}
