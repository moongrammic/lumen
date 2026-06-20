/**
 * Backend origin for server-side Next.js routes (auth proxy, API catch-all).
 * In Docker Compose use BACKEND_URL=http://backend:8080.
 * Local dev without Docker: http://localhost:8080
 */
export function getBackendUrl(): string {
  return (
    process.env.BACKEND_URL ??
    process.env.NEXT_PUBLIC_API_URL?.replace(/\/api$/, "") ??
    "http://localhost:8080"
  );
}
