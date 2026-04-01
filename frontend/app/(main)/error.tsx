"use client";

export default function MainError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div className="flex h-screen flex-col items-center justify-center gap-4 bg-zinc-950 px-6 text-center">
      <h1 className="text-lg font-semibold text-zinc-100">Что-то пошло не так</h1>
      <p className="max-w-md text-sm text-zinc-400">{error.message || "Неизвестная ошибка"}</p>
      <button
        type="button"
        className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500"
        onClick={() => reset()}
      >
        Попробовать снова
      </button>
    </div>
  );
}
