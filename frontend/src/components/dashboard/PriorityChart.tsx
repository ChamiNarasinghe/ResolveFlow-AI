import type { Ticket, TicketPriority } from "../../types";

interface PriorityChartProps {
  tickets: Ticket[];
}

const PRIORITY_META: Record<
  TicketPriority,
  { label: string; color: string; dot: string }
> = {
  urgent: {
    label: "Urgent",
    color: "bg-[var(--danger)]",
    dot: "bg-[var(--danger)]",
  },
  high: {
    label: "High",
    color: "bg-[var(--warning)]",
    dot: "bg-[var(--warning)]",
  },
  medium: {
    label: "Medium",
    color: "bg-[var(--accent)]",
    dot: "bg-[var(--accent)]",
  },
  low: {
    label: "Low",
    color: "bg-[var(--success)]",
    dot: "bg-[var(--success)]",
  },
};

export function PriorityChart({ tickets }: PriorityChartProps) {
  const priorities = Object.keys(PRIORITY_META) as TicketPriority[];

  const counts = priorities.reduce<Record<TicketPriority, number>>(
    (acc, p) => {
      acc[p] = tickets.filter((t) => t.priority === p).length;
      return acc;
    },
    { urgent: 0, high: 0, medium: 0, low: 0 },
  );

  const max = Math.max(...Object.values(counts), 1);

  return (
    <div className="flex flex-col gap-3">
      {priorities.map((p) => {
        const count = counts[p];
        const pct = (count / max) * 100;
        const meta = PRIORITY_META[p];
        return (
          <div key={p} className="flex items-center gap-3">
            <div className="flex w-20 shrink-0 items-center gap-1.5">
              <span
                className={`h-2 w-2 shrink-0 rounded-full ${meta.dot}`}
              />
              <span className="text-xs text-[var(--text-secondary)]">
                {meta.label}
              </span>
            </div>
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
      <p className="mt-1 text-xs text-[var(--text-muted)]">
        Based on the most recent {tickets.length} tickets
      </p>
    </div>
  );
}
