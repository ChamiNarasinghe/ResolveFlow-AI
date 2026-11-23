import { forwardRef } from "react";

interface TextareaProps
  extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  label?: string;
  error?: string;
  hint?: string;
  maxLength?: number;
  showCount?: boolean;
}

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  (
    {
      label,
      error,
      hint,
      maxLength,
      showCount = false,
      className = "",
      id,
      value,
      ...props
    },
    ref,
  ) => {
    const inputId = id ?? label?.toLowerCase().replace(/\s+/g, "-");
    const currentLength =
      typeof value === "string" ? value.length : 0;

    return (
      <div className="flex flex-col gap-1.5">
        {label && (
          <label
            htmlFor={inputId}
            className="text-sm font-medium text-[var(--text-primary)]"
          >
            {label}
          </label>
        )}
        <textarea
          ref={ref}
          id={inputId}
          value={value}
          maxLength={maxLength}
          className={[
            "w-full rounded-[var(--radius)] border bg-white px-3 py-2 text-sm text-[var(--text-primary)] placeholder:text-[var(--text-muted)]",
            "resize-none transition-colors duration-150",
            "focus:outline-none focus:ring-2 focus:ring-[var(--accent-ring)] focus:border-[var(--accent)]",
            "disabled:cursor-not-allowed disabled:bg-[var(--bg-subtle)] disabled:opacity-60",
            error
              ? "border-[var(--danger)] focus:ring-red-200"
              : "border-[var(--border)] hover:border-[var(--border-strong)]",
            className,
          ]
            .filter(Boolean)
            .join(" ")}
          aria-invalid={error ? "true" : undefined}
          aria-describedby={
            error
              ? `${inputId}-error`
              : hint
                ? `${inputId}-hint`
                : undefined
          }
          {...props}
        />
        <div className="flex items-start justify-between gap-2">
          <div>
            {error && (
              <p
                id={`${inputId}-error`}
                className="text-xs text-[var(--danger)]"
                role="alert"
              >
                {error}
              </p>
            )}
            {!error && hint && (
              <p
                id={`${inputId}-hint`}
                className="text-xs text-[var(--text-muted)]"
              >
                {hint}
              </p>
            )}
          </div>
          {showCount && maxLength !== undefined && (
            <p
              className={`shrink-0 text-xs tabular-nums ${
                currentLength >= maxLength
                  ? "text-[var(--danger)]"
                  : currentLength >= maxLength * 0.9
                    ? "text-[var(--warning)]"
                    : "text-[var(--text-muted)]"
              }`}
              aria-live="polite"
            >
              {currentLength}/{maxLength}
            </p>
          )}
        </div>
      </div>
    );
  },
);

Textarea.displayName = "Textarea";
