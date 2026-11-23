import Link from "next/link";
import { Skeleton } from "../ui/Skeleton";
import { EmptyState } from "../ui/EmptyState";
import { TicketStatusBadge } from "./TicketStatusBadge";
import { TicketPriorityBadge } from "./TicketPriorityBadge";
import type { Ticket } from "../../types";

interface TicketTableProps {
  tickets: Ticket[];
  teamMap: Record<string, string>;
  loading: boolean;
  /** Number of skeleton rows to show while loading */
  skeletonRows?: number;
}

const COLUMNS = [
  "#",
  "Subject",
  "Customer",
  "Category",
  "Priority",
  "Status",
  "Assigned Team",
  "Created",
];

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

function resolveTeam(ticket: Ticket, teamMap: Record<string, string>): string {
  if (!ticket.assigned_team_id) return "Unassigned";
  return teamMap[ticket.assigned_team_id] ?? "Unknown team";
}

function SkeletonRow() {
  return (
    <tr className="border-b border-[var(--border)]">
      {COLUMNS.map((col) => (
        <td key={col} className="py-3 pr-4 last:pr-0">
          <Skeleton className="h-4 w-full max-w-[120px]" />
        </td>
      ))}
    </tr>
  );
}

export function TicketTable({
  tickets,
  teamMap,
  loading,
  skeletonRows = 8,
}: TicketTableProps) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-[var(--border)]">
            {COLUMNS.map((col) => (
              <th
                key={col}
                scope="col"
                className="pb-3 pr-4 text-left text-xs font-semibold uppercase tracking-wide text-[var(--text-muted)] last:pr-0"
              >
                {col}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {loading ? (
            Array.from({ length: skeletonRows }).map((_, i) => (
              <SkeletonRow key={i} />
            ))
          ) : tickets.length === 0 ? (
            <tr>
              <td colSpan={COLUMNS.length} className="py-0">
                <EmptyState
                  title="No tickets found"
                  description="No tickets match the current filters. Try adjusting or clearing them."
                  className="my-6 border-0 shadow-none bg-transparent"
                  icon={
                    <svg fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5} aria-hidden="true">
                      <path strokeLinecap="round" strokeLinejoin="round" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
                    </svg>
                  }
                />
              </td>
            </tr>
          ) : (
            tickets.map((ticket) => (
              <tr
                key={ticket.id}
                className="border-b border-[var(--border)] last:border-0 transition-colors hover:bg-[var(--bg-subtle)]"
              >
                {/* # */}
                <td className="py-3 pr-4 font-mono text-xs text-[var(--text-muted)] whitespace-nowrap">
                  #{ticket.ticket_number}
                </td>

                {/* Subject */}
                <td className="py-3 pr-4 max-w-[220px]">
                  <Link
                    href={`/tickets/${ticket.id}`}
                    className="block truncate font-medium text-[var(--text-primary)] hover:text-[var(--accent)] hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-ring)] rounded"
                    title={ticket.subject}
                  >
                    {ticket.subject}
                  </Link>
                </td>

                {/* Customer */}
                <td className="py-3 pr-4 min-w-[140px]">
                  <span className="block truncate font-medium text-[var(--text-primary)] text-xs" title={ticket.customer_name}>
                    {ticket.customer_name}
                  </span>
                  <span className="block truncate text-xs text-[var(--text-muted)]" title={ticket.customer_email}>
                    {ticket.customer_email}
                  </span>
                </td>

                {/* Category */}
                <td className="py-3 pr-4 text-xs text-[var(--text-secondary)] whitespace-nowrap">
                  {formatCategory(ticket.category)}
                </td>

                {/* Priority */}
                <td className="py-3 pr-4 whitespace-nowrap">
                  <TicketPriorityBadge priority={ticket.priority} />
                </td>

                {/* Status */}
                <td className="py-3 pr-4 whitespace-nowrap">
                  <TicketStatusBadge status={ticket.status} />
                </td>

                {/* Assigned team */}
                <td className="py-3 pr-4 text-xs text-[var(--text-secondary)] whitespace-nowrap">
                  {resolveTeam(ticket, teamMap)}
                </td>

                {/* Created */}
                <td className="py-3 text-xs text-[var(--text-muted)] whitespace-nowrap">
                  {formatDate(ticket.created_at)}
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}
