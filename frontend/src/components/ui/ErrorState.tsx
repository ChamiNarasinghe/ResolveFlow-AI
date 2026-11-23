interface ErrorStateProps {
  title?: string;
  message: string;
  onRetry?: () => void;
  className?: string;
}

export function ErrorState({
  title = "Something went wrong",
  message,
  onRetry,
  className = "",
}: ErrorStateProps) {
  return (
    <div
      role="alert"
      className={`flex flex-col items-center justify-center gap-3 rounded-[var(--radius-lg)] border border-[var(--danger-bg)] bg-[var(--danger-bg)] px-6 py-10 text-center ${className}`}
    >
      <svg
        className="h-8 w-8 text-[var(--danger)]"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        strokeWidth={1.5}
        aria-hidden="true"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z"
        />
      </svg>
      <div className="space-y-1">
        <p className="text-sm font-semibold text-[var(--danger)]">{title}</p>
        <p className="text-sm text-[var(--text-secondary)]">{message}</p>
      </div>
      {onRetry && (
        <button
          onClick={onRetry}
          className="mt-1 rounded-[var(--radius-sm)] border border-[var(--danger)] px-3 py-1.5 text-xs font-medium text-[var(--danger)] transition-colors hover:bg-red-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-300"
        >
          Try again
        </button>
      )}
    </div>
  );
}
