import { isAxiosError } from "axios";

type ApiErrorBody = { message?: string; error?: string };

export function getApiErrorMessage(error: unknown, fallback: string): string {
  if (isAxiosError(error)) {
    const data = error.response?.data as ApiErrorBody | undefined;
    return data?.message ?? data?.error ?? fallback;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return fallback;
}
