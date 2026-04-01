import { NextResponse } from "next/server";

const backendUrl = process.env.BACKEND_URL ?? "http://localhost:8080";

export async function POST(request: Request) {
  const body = await request.json();

  const upstream = await fetch(`${backendUrl}/api/auth/login`, {
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
  if (!accessToken) {
    return NextResponse.json({ error: "missing_token" }, { status: 502 });
  }

  response.cookies.set({
    name: "access_token",
    value: accessToken,
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: 60 * 60 * 24 * 7,
  });

  return response;
}
