export type BadgeVariant =
  | "default"
  | "success"
  | "warning"
  | "danger"
  | "info"
  | "neutral";

interface BadgeProps {
  variant?: BadgeVariant;
  children: React.ReactNode;
  className?: string;
}

const variants: Record<BadgeVariant, string> = {
  default:
    "bg-[var(--accent-light)] text-[var(--accent)] border border-[var(--accent-ring)]",
  success:
    "bg-[var(--success-bg)] text-[var(--success)] border border-green-200",
  warning:
    "bg-[var(--warning-bg)] text-[var(--warning)] border border-amber-200",
  danger: "bg-[var(--danger-bg)] text-[var(--danger)] border border-red-200",
  info: "bg-[var(--info-bg)] text-[var(--info)] border border-sky-200",
  neutral:
    "bg-[var(--bg-subtle)] text-[var(--text-secondary)] border border-[var(--border)]",
};

export function Badge({
  variant = "default",
  children,
  className = "",
}: BadgeProps) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${variants[variant]} ${className}`}
    >
      {children}
    </span>
  );
}
