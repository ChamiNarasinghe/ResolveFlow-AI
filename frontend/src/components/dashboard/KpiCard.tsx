interface KpiCardProps {
  label: string;
  value: number;
  icon: React.ReactNode;
  accent?: boolean;
}

export function KpiCard({ label, value, icon, accent = false }: KpiCardProps) {
  return (
    <div
      className={[
        "flex items-center gap-4 rounded-[var(--radius-lg)] border p-5 shadow-[var(--shadow-sm)]",
        accent
          ? "border-[var(--accent)] bg-[var(--accent)] text-white"
          : "border-[var(--border)] bg-[var(--bg-surface)]",
      ].join(" ")}
    >
      <div
        className={[
          "flex h-11 w-11 shrink-0 items-center justify-center rounded-[var(--radius-md)]",
          accent
            ? "bg-white/20"
            : "bg-[var(--accent-light)] text-[var(--accent)]",
        ].join(" ")}
      >
        {icon}
      </div>
      <div className="min-w-0">
        <p
          className={[
            "text-2xl font-bold tabular-nums leading-none",
            accent ? "text-white" : "text-[var(--text-primary)]",
          ].join(" ")}
        >
          {value.toLocaleString()}
        </p>
        <p
          className={[
            "mt-1 text-xs font-medium",
            accent ? "text-white/80" : "text-[var(--text-muted)]",
          ].join(" ")}
        >
          {label}
        </p>
      </div>
    </div>
  );
}
