"use client";

import { useCallback, useEffect, useReducer, useState } from "react";
import { useRouter } from "next/navigation";
import { ticketsService } from "../../services/tickets";
import { teamsService } from "../../services/teams";
import { ErrorState } from "../ui/ErrorState";
import { Button } from "../ui/Button";
import { TicketFilters } from "./TicketFilters";
import { TicketTable } from "./TicketTable";
import { TicketPagination } from "./TicketPagination";
import { TicketCreateModal } from "./TicketCreateModal";
import type { Pagination, Ticket, TicketPriority, TicketStatus } from "../../types";
import type { TicketFiltersValue } from "./TicketFilters";

const DEFAULT_PAGE_SIZE = 20;

const EMPTY_PAGINATION: Pagination = {
  page: 1,
  page_size: DEFAULT_PAGE_SIZE,
  total: 0,
  total_pages: 0,
};

// ─── Fetch state ──────────────────────────────────────────────────────────────

interface FetchState {
  tickets: Ticket[];
  pagination: Pagination;
  loading: boolean;
  error: string | null;
}

type FetchAction =
  | { type: "LOADING" }
  | { type: "SUCCESS"; tickets: Ticket[]; pagination: Pagination }
  | { type: "ERROR"; message: string };

function fetchReducer(state: FetchState, action: FetchAction): FetchState {
  switch (action.type) {
    case "LOADING":
      return { ...state, loading: true, error: null };
    case "SUCCESS":
      return { loading: false, error: null, tickets: action.tickets, pagination: action.pagination };
    case "ERROR":
      return { ...state, loading: false, error: action.message };
  }
}

const initialFetchState: FetchState = {
  tickets: [],
  pagination: EMPTY_PAGINATION,
  loading: true,
  error: null,
};

// ─── Component ────────────────────────────────────────────────────────────────

export function TicketsClient() {
  const router = useRouter();
  const [state, dispatch] = useReducer(fetchReducer, initialFetchState);
  const [teamMap, setTeamMap] = useState<Record<string, string>>({});
  const [filters, setFilters] = useState<TicketFiltersValue>({ status: "", priority: "" });
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [fetchKey, setFetchKey] = useState(0);
  const [modalOpen, setModalOpen] = useState(false);

  // Fetch teams once on mount — non-fatal if it fails.
  useEffect(() => {
    teamsService
      .list()
      .then((res) => {
        const map: Record<string, string> = {};
        for (const team of res.data) map[team.id] = team.name;
        setTeamMap(map);
      })
      .catch(() => undefined);
  }, []);

  // Fetch tickets. All state transitions happen inside async callbacks so no
  // setState is called synchronously in the effect body.
  useEffect(() => {
    let cancelled = false;

    dispatch({ type: "LOADING" });

    ticketsService
      .list({
        page,
        page_size: pageSize,
        ...(filters.status ? { status: filters.status as TicketStatus } : {}),
        ...(filters.priority ? { priority: filters.priority as TicketPriority } : {}),
      })
      .then((res) => {
        if (!cancelled)
          dispatch({ type: "SUCCESS", tickets: res.data, pagination: res.pagination });
      })
      .catch((err: unknown) => {
        if (!cancelled)
          dispatch({
            type: "ERROR",
            message: err instanceof Error ? err.message : "Failed to load tickets.",
          });
      });

    return () => {
      cancelled = true;
    };
  }, [page, pageSize, filters, fetchKey]);

  const refresh = useCallback(() => setFetchKey((k) => k + 1), []);

  function handleCreateSuccess(ticket: Ticket) {
    setModalOpen(false);
    router.push(`/tickets/${ticket.id}`);
  }

  function handleFiltersChange(next: TicketFiltersValue) {
    setFilters(next);
    setPage(1);
  }

  function handlePageSizeChange(next: number) {
    setPageSize(next);
    setPage(1);
  }

  const { tickets, pagination, loading, error } = state;

  return (
    <div className="flex flex-col gap-6">
      {/* Page header */}
      <div>
        <h1 className="text-xl font-bold text-[var(--text-primary)]">Tickets</h1>
        <p className="mt-0.5 text-sm text-[var(--text-muted)]">
          Browse, filter, and manage all customer support tickets.
        </p>
      </div>

      {/* Card */}
      <div className="rounded-[var(--radius-lg)] border border-[var(--border)] bg-[var(--bg-surface)] shadow-[var(--shadow-sm)]">
        {/* Toolbar */}
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--border)] px-5 py-4">
          <TicketFilters
            value={filters}
            onChange={handleFiltersChange}
            disabled={loading}
          />

          <div className="flex items-center gap-2">
            <button
              onClick={refresh}
              disabled={loading}
              aria-label="Refresh tickets"
              className="inline-flex h-9 items-center gap-2 rounded-[var(--radius)] border border-[var(--border)] bg-[var(--bg-surface)] px-3 text-xs font-medium text-[var(--text-secondary)] shadow-[var(--shadow-sm)] transition-colors hover:bg-[var(--bg-subtle)] disabled:opacity-50 disabled:cursor-not-allowed focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-ring)]"
            >
              <svg
                className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`}
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2}
                aria-hidden="true"
              >
                <path strokeLinecap="round" strokeLinejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              Refresh
            </button>

            <Button
              variant="primary"
              size="md"
              onClick={() => setModalOpen(true)}
            >
              <svg
                className="h-4 w-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2}
                aria-hidden="true"
              >
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
              </svg>
              Create Ticket
            </Button>
          </div>
        </div>

        {/* Body */}
        <div className="p-5">
          {error ? (
            <ErrorState message={error} onRetry={refresh} />
          ) : (
            <>
              <TicketTable
                tickets={tickets}
                teamMap={teamMap}
                loading={loading}
              />
              {!loading && (
                <TicketPagination
                  pagination={pagination}
                  onPageChange={setPage}
                  onPageSizeChange={handlePageSizeChange}
                  disabled={loading}
                />
              )}
            </>
          )}
        </div>
      </div>
      <TicketCreateModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        onSuccess={handleCreateSuccess}
      />
    </div>
  );
}
