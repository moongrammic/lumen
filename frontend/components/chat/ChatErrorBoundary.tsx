"use client";

import { Component, type ErrorInfo, type ReactNode } from "react";

type Props = {
  children: ReactNode;
  fallback?: ReactNode;
};

type State = { hasError: boolean };

export class ChatErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false };

  static getDerivedStateFromError(): State {
    return { hasError: true };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("ChatErrorBoundary:", error, info.componentStack);
  }

  render() {
    if (this.state.hasError) {
      return (
        this.props.fallback ?? (
          <div className="flex flex-1 flex-col items-center justify-center gap-2 p-6 text-center text-sm text-zinc-400">
            <p>Не удалось отобразить чат.</p>
            <button
              type="button"
              className="rounded-md bg-zinc-800 px-3 py-1.5 text-zinc-100 hover:bg-zinc-700"
              onClick={() => this.setState({ hasError: false })}
            >
              Попробовать снова
            </button>
          </div>
        )
      );
    }
    return this.props.children;
  }
}
