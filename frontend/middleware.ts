/**
 * Должен лежать в корне приложения Next (`frontend/middleware.ts`), рядом с `app/`, не внутри `app/`.
 */
import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const PROTECTED_PREFIXES = ["/guilds", "/channels"];
const AUTH_PAGES = ["/login", "/register"];

export function middleware(request: NextRequest) {
  const token = request.cookies.get("access_token")?.value;
  const { pathname } = request.nextUrl;

  const isProtected = PROTECTED_PREFIXES.some((prefix) => pathname.startsWith(prefix));
  const isAuthPage = AUTH_PAGES.some((path) => pathname.startsWith(path));

  if (isProtected && !token) {
    return NextResponse.redirect(new URL("/login", request.url));
  }

  if (isAuthPage && token) {
    return NextResponse.redirect(new URL("/guilds", request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/guilds/:path*", "/channels/:path*", "/login", "/register"],
};
