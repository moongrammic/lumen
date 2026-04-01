import { NextResponse } from "next/server";

const backendUrl = process.env.BACKEND_URL ?? "http://localhost:8080";

export async function POST(request: Request) {
  const body = await request.json();

  const upstream = await fetch(`${backendUrl}/api/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });

  const data = await upstream.json().catch(() => null);
  return NextResponse.json(data ?? { error: "invalid_upstream" }, { status: upstream.status });
}
