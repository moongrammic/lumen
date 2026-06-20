import { NextResponse } from "next/server";

/** Matches JWT TTL in backend/internal/service/auth.go */
export const AUTH_COOKIE_MAX_AGE = 60 * 60 * 24;

export function setAuthCookie(response: NextResponse, token: string) {
  response.cookies.set({
    name: "access_token",
    value: token,
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: AUTH_COOKIE_MAX_AGE,
  });
}

export function clearAuthCookie(response: NextResponse) {
  response.cookies.set({
    name: "access_token",
    value: "",
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: 0,
  });
}
