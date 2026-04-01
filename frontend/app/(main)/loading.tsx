export default function MainLoading() {
  return (
    <div className="flex h-screen items-center justify-center bg-zinc-950 text-zinc-400">
      <div className="flex flex-col items-center gap-3">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-zinc-600 border-t-indigo-500" />
        <p className="text-sm">Загрузка…</p>
      </div>
    </div>
  );
}
