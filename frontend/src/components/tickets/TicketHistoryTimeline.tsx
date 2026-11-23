"use client";

import { useEffect, useReducer, useState } from "react";
import { ticketsService } from "../../services/tickets";
import { Card, CardHeader, CardBody } from "../ui/Card";
import { Skeleton } from "../ui/Skeleton";
import { EmptyState } from "../ui/EmptyState";
import { Button } from "../ui/Button";
import type { Pagination, TicketHistory } from "../../types";

// ─── State ────────────────────────────────────────────────────────────────────

interface HistoryState {
  records: TicketHistory[];
  pagination: Pagination | null;
  loading: boolean;
  error: string | null;
}

type HistoryAction =
  | { type: "LOADING" }
  | { type: "SUCCESS"; records: TicketHistory[]; pagination: Pagination }
  | { type: "ERROR"; message: string };

function historyReducer(state: HistoryState, action: HistoryAction): HistoryState {
  switch (action.type) {
    case "LOADING": return { ...state, loading: true, error: null };
    case "SUCCESS": return { records: action.records, pagination: action.pagination, loading: false, error: null };
    case "ERROR":   return { ...state, loading: false, error: action.message };
  }
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

const ACTION_LABEL: Record<string, string> = {
  "ticket.updated":    "Updated",
  "ticket.assigned":   "Team assigned",
  "ticket.unassigned": "Team unassigned",
  "ticket.deleted":    "Deleted",
};

const FIELD_LABEL: Record<string, string> = {
  status:           "Status",
  priority:         "Priority",
  category:         "Category",
  assigned_team_id: "Assigned team",
  assigned_to:      "Assigned agent",
  deleted_at:       "Deleted at",
};

function formatValue(key: string, value: unknown): string {
  if (value === null || value === undefined) return "Unassigned";
  if (typeof value === "string") {
    // Format snake_case status/priority values
    if (key === "status" || key === "priority" || key === "category") {
      return value.split("_").map((w) => w[0].toUpperCase() + w.slice(1)).join(" ");
    }
    // Truncate UUIDs
    if (value.length === 36 && value.includes("-")) return value.slice(0, 8) + "…";
    return value;
  }
  return String(value);
}

function FieldDiff({ oldVal, newVal }: { oldVal: Record<string, unknown>; newVal: Record<string, unknown> }) {
  const keys = Object.keys(newVal);
  if (keys.length === 0) return null;
  return (
    <ul className="mt-1.5 flex flex-col gap-1">
      {keys.map((key) => (
        <li key={key} className="flex flex-wrap items-center gap-1 text-xs text-[var(--text-secondary)]">
          <span className="font-medium text-[var(--text-primary)]">
            {FIELD_LABEL[key] ?? key}
          </span>
          <span className="text-[var(--text-muted)]">
            {formatValue(key, oldVal[key])}
          </span>
          <svg className="h-3 w-3 shrink-0 text-[var(--text-muted)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-hidden="true">
            <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
          </svg>
          <span className="font-medium text-[var(--text-primary)]">
            {formatValue(key, newVal[key])}
          </span>
        </li>
      ))}
    </ul>
  );
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString("en-US", {
    month: "short", day: "numeric", year: "numeric",
    hour: "numeric", minute: "2-digit",
  });
}

function HistoryEventIcon() {
  return (
    <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full border border-[var(--border)] bg-[var(--bg-subtle)]">
      <svg className="h-3.5 w-3.5 text-[var(--text-muted)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-hidden="true">
        <path strokeLinecap="round" strokeLinejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    </span>
  );
}

// ─── Component ────────────────────────────────────────────────────────────────

const PAGE_SIZE = 20;

export function TicketHistoryTimeline({ ticketId }: { ticketId: string }) {
  const [state, dispatch] = useReducer(historyReducer, {
    records: [], pagination: null, loading: true, error: null,
  });
  const [page, setPage] = useState(1);
  const [fetchKey, setFetchKey] = useState(0);

  useEffect(() => {
    let cancelled = false;

    dispatch({ type: "LOADING" });

    ticketsService
      .getHistory(ticketId, { page, page_size: PAGE_SIZE })
      .then((res) => {
        if (!cancelled)
          dispatch({ type: "SUCCESS", records: res.data, pagination: res.pagination });
      })
      .catch((err: unknown) => {
        if (!cancelled)
          dispatch({
            type: "ERROR",
            message: err instanceof Error ? err.message : "Failed to load history.",
          });
      });

    return () => { cancelled = true; };
  }, [ticketId, page, fetchKey]);

  const { records, pagination, loading, error } = state;
  const isEmpty = !loading && !error && records.length === 0;
  const canPrev = (pagination?.page ?? 1) > 1;
  const canNext = pagination ? pagination.page < pagination.total_pages && pagination.total > 0 : false;

  return (
    <Card padding={false}>
      <CardHeader>
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold text-[var(--text-primary)]">History</h2>
          {pagination && pagination.total > 0 && (
            <span className="text-xs text-[var(--text-muted)]">{pagination.total} event{pagination.total !== 1 ? "s" : ""}</span>
          )}
        </div>
      </CardHeader>

      <CardBody>
        {loading ? (
          <div className="flex flex-col gap-4">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="flex gap-3">
                <Skeleton className="mt-0.5 h-7 w-7 shrink-0 rounded-full" />
                <div className="flex-1">
                  <Skeleton className="mb-1.5 h-3 w-48" />
                  <Skeleton className="h-3 w-32" />
                </div>
              </div>
            ))}
          </div>
        ) : error ? (
          <div className="flex flex-col items-center gap-2 py-4 text-center">
            <p className="text-xs text-[var(--danger)]">{error}</p>
            <button
              onClick={() => setFetchKey((k) => k + 1)}
              className="text-xs font-medium text-[var(--accent)] hover:underline focus-visible:outline-none"
            >
              Try again
            </button>
          </div>
        ) : isEmpty ? (
          <EmptyState
            title="No history yet"
            description="Changes to this ticket will appear here."
            className="border-0 bg-transparent py-6 shadow-none"
          />
        ) : (
          <>
            <ol className="flex flex-col gap-4">
              {records.map((record) => {
                const label = ACTION_LABEL[record.action] ?? record.action;
                const oldVal = (record.old_value ?? {}) as Record<string, unknown>;
                const newVal = (record.new_value ?? {}) as Record<string, unknown>;
                const actor = record.actor_id
                  ? `Agent ${record.actor_id.slice(0, 8)}…`
                  : "Unknown actor";

                return (
                  <li key={record.id} className="flex gap-3">
                    <HistoryEventIcon />
                    <div className="flex-1 min-w-0">
                      <div className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
                        <span className="text-xs font-semibold text-[var(--text-primary)]">{label}</span>
                        <span className="text-[10px] text-[var(--text-muted)]">by {actor}</span>
                        <span className="text-[10px] text-[var(--text-muted)]">· {formatDate(record.created_at)}</span>
                      </div>
                      <FieldDiff oldVal={oldVal} newVal={newVal} />
                    </div>
                  </li>
                );
              })}
            </ol>

            {/* Pagination */}
            {pagination && pagination.total > PAGE_SIZE && (
              <div className="mt-5 flex items-center justify-between border-t border-[var(--border)] pt-4">
                <span className="text-xs text-[var(--text-muted)]">
                  Page {pagination.page} of {pagination.total_pages}
                </span>
                <div className="flex gap-2">
                  <Button
                    variant="secondary"
                    size="sm"
                    disabled={!canPrev || loading}
                    onClick={() => setPage((p) => p - 1)}
                  >
                    Previous
                  </Button>
                  <Button
                    variant="secondary"
                    size="sm"
                    disabled={!canNext || loading}
                    onClick={() => setPage((p) => p + 1)}
                  >
                    Next
                  </Button>
                </div>
              </div>
            )}
          </>
        )}
      </CardBody>
    </Card>
  );
}
