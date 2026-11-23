import type { Ticket } from "../../types";

interface TrendChartProps {
  tickets: Ticket[];
}

// Build a local-calendar date key "YYYY-MM-DD" from a Date object using the
// browser's local timezone — avoids the UTC-shift that toISOString() produces.
function localDateKey(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

function getLast7Days(): { label: string; dateStr: string }[] {
  const days: { label: string; dateStr: string }[] = [];
  for (let i = 6; i >= 0; i--) {
    const d = new Date();
    d.setDate(d.getDate() - i);
    days.push({
      // Label and dateStr both derived from the same local Date — no UTC shift.
      label: d.toLocaleDateString("en-US", { weekday: "short" }),
      dateStr: localDateKey(d),
    });
  }
  return days;
}

export function TrendChart({ tickets }: TrendChartProps) {
  const days = getLast7Days();

  const counts = days.map(({ dateStr }) =>
    tickets.filter((t) => {
      // Parse the UTC ISO string from the backend into a local calendar date
      // so the bucketing boundary matches what the user sees on their clock.
      return localDateKey(new Date(t.created_at)) === dateStr;
    }).length,
  );

  const max = Math.max(...counts, 1);
  const W = 280;
  const H = 64;
  const pad = 4;
  const step = (W - pad * 2) / (counts.length - 1);

  const points = counts.map((c, i) => ({
    x: pad + i * step,
    y: H - pad - ((c / max) * (H - pad * 2)),
  }));

  const polyline = points.map((p) => `${p.x},${p.y}`).join(" ");
  const area = [
    `M ${points[0].x},${H - pad}`,
    ...points.map((p) => `L ${p.x},${p.y}`),
    `L ${points[points.length - 1].x},${H - pad}`,
    "Z",
  ].join(" ");

  return (
    <div className="flex flex-col gap-3">
      <svg
        viewBox={`0 0 ${W} ${H}`}
        className="w-full"
        aria-hidden="true"
        preserveAspectRatio="none"
      >
        <defs>
          <linearGradient id="trendGrad" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#838236" stopOpacity="0.25" />
            <stop offset="100%" stopColor="#838236" stopOpacity="0" />
          </linearGradient>
        </defs>
        <path d={area} fill="url(#trendGrad)" />
        <polyline
          points={polyline}
          fill="none"
          stroke="#838236"
          strokeWidth="2"
          strokeLinejoin="round"
          strokeLinecap="round"
        />
        {points.map((p, i) => (
          <circle key={i} cx={p.x} cy={p.y} r="3" fill="#838236" />
        ))}
      </svg>
      <div className="flex justify-between">
        {days.map(({ label }, i) => (
          <span
            key={i}
            className="text-center text-xs text-[var(--text-muted)]"
            style={{ width: `${100 / days.length}%` }}
          >
            {label}
          </span>
        ))}
      </div>
      <p className="text-xs text-[var(--text-muted)]">
        Showing activity from the most recent {tickets.length} tickets.
        Days with high volume may be understated if the total exceeds 100.
      </p>
    </div>
  );
}
