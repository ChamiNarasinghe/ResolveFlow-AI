import Link from "next/link";
import { Badge } from "../ui/Badge";
import type { BadgeVariant } from "../ui/Badge";
import type { Ticket, TicketPriority, TicketStatus } from "../../types";

interface RecentTicketsProps {
  tickets: Ticket[];
}

const STATUS_BADGE: Record<TicketStatus, BadgeVariant> = {
  open: "default",
  in_progress: "info",
  waiting_for_customer: "warning",
  resolved: "success",
  closed: "neutral",
};

const STATUS_LABEL: Record<TicketStatus, string> = {
  open: "Open",
  in_progress: "In Progress",
  waiting_for_customer: "Waiting",
  resolved: "Resolved",
  closed: "Closed",
};

const PRIORITY_BADGE: Record<TicketPriority, BadgeVariant> = {
  urgent: "danger",
  high: "warning",
  medium: "default",
  low: "neutral",
};

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

function formatCategory(cat: string | null): string {
  if (!cat) return "—";
  return cat
    .split("_")
    .map((w) => w[0].toUpperCase() + w.slice(1))
    .join(" ");
}

export function RecentTickets({ tickets }: RecentTicketsProps) {
  if (tickets.length === 0) {
    return (
      <p className="py-8 text-center text-sm text-[var(--text-muted)]">
        No tickets yet.
      </p>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-[var(--border)]">
            {["#", "Subject", "Category", "Priority", "Status", "Created"].map(
              (h) => (
                <th
                  key={h}
                  className="pb-3 pr-4 text-left text-xs font-semibold uppercase tracking-wide text-[var(--text-muted)] last:pr-0"
                >
                  {h}
                </th>
              ),
            )}
          </tr>
        </thead>
        <tbody>
          {tickets.map((t) => (
            <tr
              key={t.id}
              className="border-b border-[var(--border)] last:border-0 hover:bg-[var(--bg-subtle)] transition-colors"
            >
              <td className="py-3 pr-4 font-mono text-xs text-[var(--text-muted)]">
                #{t.ticket_number}
              </td>
              <td className="py-3 pr-4 max-w-[200px]">
                <span
                  className="block truncate font-medium text-[var(--text-primary)]"
                  title={t.subject}
                >
                  {t.subject}
                </span>
              </td>
              <td className="py-3 pr-4 text-xs text-[var(--text-secondary)]">
                {formatCategory(t.category)}
              </td>
              <td className="py-3 pr-4">
                <Badge variant={PRIORITY_BADGE[t.priority]}>
                  {t.priority.charAt(0).toUpperCase() + t.priority.slice(1)}
                </Badge>
              </td>
              <td className="py-3 pr-4">
                <Badge variant={STATUS_BADGE[t.status]}>
                  {STATUS_LABEL[t.status]}
                </Badge>
              </td>
              <td className="py-3 text-xs text-[var(--text-muted)]">
                {formatDate(t.created_at)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <div className="mt-4 flex justify-end">
        <Link
          href="/tickets"
          className="text-xs font-medium text-[var(--accent)] hover:underline"
        >
          View all tickets →
        </Link>
      </div>
    </div>
  );
}
