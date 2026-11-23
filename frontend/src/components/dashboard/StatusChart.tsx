import type { StatusCounts } from "../../services/dashboard";
import type { TicketStatus } from "../../types";

interface StatusChartProps {
  counts: StatusCounts;
  total: number;
}

const STATUS_META: Record<TicketStatus, { label: string; color: string }> = {
  open: { label: "Open", color: "bg-[var(--accent)]" },
  in_progress: { label: "In Progress", color: "bg-[var(--info)]" },
  waiting_for_customer: { label: "Waiting", color: "bg-[var(--warning)]" },
  resolved: { label: "Resolved", color: "bg-[var(--success)]" },
  closed: { label: "Closed", color: "bg-[var(--text-muted)]" },
};

export function StatusChart({ counts, total }: StatusChartProps) {
  const statuses = Object.keys(STATUS_META) as TicketStatus[];

  return (
    <div className="flex flex-col gap-3">
      {statuses.map((status) => {
        const count = counts[status];
        const pct = total > 0 ? (count / total) * 100 : 0;
        const meta = STATUS_META[status];
        return (
          <div key={status} className="flex items-center gap-3">
            <span className="w-28 shrink-0 text-xs text-[var(--text-secondary)]">
              {meta.label}
            </span>
            <div className="flex-1 overflow-hidden rounded-full bg-[var(--bg-subtle)] h-2">
              <div
                className={`h-2 rounded-full transition-all duration-500 ${meta.color}`}
                style={{ width: `${pct}%` }}
              />
            </div>
            <span className="w-8 shrink-0 text-right text-xs font-medium tabular-nums text-[var(--text-primary)]">
              {count}
            </span>
          </div>
        );
      })}
    </div>
  );
}
