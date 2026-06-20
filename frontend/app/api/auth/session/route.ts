import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import { clearAuthCookie } from "@/lib/auth-cookie";
import { getBackendUrl } from "@/lib/server-backend";

export async function GET() {
  const backendUrl = getBackendUrl();
  const cookieStore = await cookies();
  const token = cookieStore.get("access_token")?.value;

  if (!token) {
    return NextResponse.json({ error: "unauthenticated" }, { status: 401 });
  }

  const upstream = await fetch(`${backendUrl}/api/me`, {
    headers: { Authorization: `Bearer ${token}` },
    cache: "no-store",
  });

  const user = await upstream.json().catch(() => null);
  if (!upstream.ok) {
    const response = NextResponse.json(user ?? { error: "session_invalid" }, { status: upstream.status });
    // Clear cookie only when token is truly invalid/forbidden.
    // Do not clear on transient upstream failures (429/5xx), otherwise users get logged out.
    if (upstream.status === 401 || upstream.status === 403) {
      clearAuthCookie(response);
    }
    return response;
  }

  return NextResponse.json({ token, user });
}
