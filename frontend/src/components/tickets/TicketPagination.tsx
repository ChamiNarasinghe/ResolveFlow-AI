"use client";

import type { Pagination } from "../../types";

const PAGE_SIZE_OPTIONS = [20, 50, 100] as const;

interface TicketPaginationProps {
  pagination: Pagination;
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
  disabled?: boolean;
}

export function TicketPagination({
  pagination,
  onPageChange,
  onPageSizeChange,
  disabled,
}: TicketPaginationProps) {
  const { page, page_size, total, total_pages } = pagination;
  const canPrev = page > 1;
  const canNext = page < total_pages;

  const start = total === 0 ? 0 : (page - 1) * page_size + 1;
  const end = Math.min(page * page_size, total);

  const btnBase =
    "inline-flex h-8 w-8 items-center justify-center rounded-[var(--radius-sm)] border border-[var(--border)] bg-[var(--bg-surface)] text-sm text-[var(--text-secondary)] shadow-[var(--shadow-sm)] transition-colors hover:bg-[var(--bg-subtle)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-ring)] disabled:opacity-40 disabled:cursor-not-allowed";

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 pt-4">
      {/* Left: count summary */}
      <p className="text-xs text-[var(--text-muted)]">
        {total === 0 ? (
          "No tickets"
        ) : (
          <>
            Showing <span className="font-medium text-[var(--text-secondary)]">{start}–{end}</span> of{" "}
            <span className="font-medium text-[var(--text-secondary)]">{total}</span> ticket{total !== 1 ? "s" : ""}
          </>
        )}
      </p>

      {/* Right: controls */}
      <div className="flex items-center gap-3">
        {/* Page size */}
        <div className="flex items-center gap-2">
          <label htmlFor="page-size-select" className="text-xs text-[var(--text-muted)]">
            Rows
          </label>
          <select
            id="page-size-select"
            value={page_size}
            disabled={disabled}
            onChange={(e) => onPageSizeChange(Number(e.target.value))}
            className="h-8 rounded-[var(--radius-sm)] border border-[var(--border)] bg-[var(--bg-surface)] px-2 pr-6 text-xs text-[var(--text-primary)] shadow-[var(--shadow-sm)] appearance-none focus:outline-none focus:ring-2 focus:ring-[var(--accent-ring)] disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
            aria-label="Rows per page"
          >
            {PAGE_SIZE_OPTIONS.map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
          </select>
        </div>

        {/* Prev / page indicator / Next */}
        <div className="flex items-center gap-1.5">
          <button
            onClick={() => onPageChange(page - 1)}
            disabled={!canPrev || disabled}
            aria-label="Previous page"
            className={btnBase}
          >
            <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-hidden="true">
              <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7" />
            </svg>
          </button>

          <span className="min-w-[5rem] text-center text-xs text-[var(--text-secondary)]">
            Page <span className="font-medium">{total_pages === 0 ? 0 : page}</span> of{" "}
            <span className="font-medium">{total_pages}</span>
          </span>

          <button
            onClick={() => onPageChange(page + 1)}
            disabled={!canNext || disabled}
            aria-label="Next page"
            className={btnBase}
          >
            <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-hidden="true">
              <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  );
}
