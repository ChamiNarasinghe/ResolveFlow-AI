"use client";

import type { TicketPriority, TicketStatus } from "../../types";

export interface TicketFiltersValue {
  status: TicketStatus | "";
  priority: TicketPriority | "";
}

interface TicketFiltersProps {
  value: TicketFiltersValue;
  onChange: (next: TicketFiltersValue) => void;
  disabled?: boolean;
}

const STATUS_OPTIONS: { value: TicketStatus | ""; label: string }[] = [
  { value: "", label: "All statuses" },
  { value: "open", label: "Open" },
  { value: "in_progress", label: "In Progress" },
  { value: "waiting_for_customer", label: "Waiting for Customer" },
  { value: "resolved", label: "Resolved" },
  { value: "closed", label: "Closed" },
];

const PRIORITY_OPTIONS: { value: TicketPriority | ""; label: string }[] = [
  { value: "", label: "All priorities" },
  { value: "urgent", label: "Urgent" },
  { value: "high", label: "High" },
  { value: "medium", label: "Medium" },
  { value: "low", label: "Low" },
];

const selectClass =
  "h-9 rounded-[var(--radius)] border border-[var(--border)] bg-[var(--bg-surface)] px-3 pr-8 text-sm text-[var(--text-primary)] shadow-[var(--shadow-sm)] transition-colors hover:border-[var(--border-strong)] focus:outline-none focus:ring-2 focus:ring-[var(--accent-ring)] focus:border-[var(--accent)] disabled:opacity-50 disabled:cursor-not-allowed appearance-none cursor-pointer";

export function TicketFilters({ value, onChange, disabled }: TicketFiltersProps) {
  const hasActiveFilter = value.status !== "" || value.priority !== "";

  function handleStatus(e: React.ChangeEvent<HTMLSelectElement>) {
    onChange({ ...value, status: e.target.value as TicketStatus | "" });
  }

  function handlePriority(e: React.ChangeEvent<HTMLSelectElement>) {
    onChange({ ...value, priority: e.target.value as TicketPriority | "" });
  }

  function handleClear() {
    onChange({ status: "", priority: "" });
  }

  return (
    <div className="flex flex-wrap items-center gap-3">
      {/* Status */}
      <div className="relative">
        <select
          value={value.status}
          onChange={handleStatus}
          disabled={disabled}
          aria-label="Filter by status"
          className={selectClass}
        >
          {STATUS_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
        <ChevronIcon />
      </div>

      {/* Priority */}
      <div className="relative">
        <select
          value={value.priority}
          onChange={handlePriority}
          disabled={disabled}
          aria-label="Filter by priority"
          className={selectClass}
        >
          {PRIORITY_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
        <ChevronIcon />
      </div>

      {/* Clear */}
      {hasActiveFilter && (
        <button
          onClick={handleClear}
          disabled={disabled}
          className="inline-flex h-9 items-center gap-1.5 rounded-[var(--radius)] border border-[var(--border)] bg-[var(--bg-surface)] px-3 text-sm font-medium text-[var(--text-secondary)] shadow-[var(--shadow-sm)] transition-colors hover:bg-[var(--bg-subtle)] hover:text-[var(--text-primary)] disabled:opacity-50 disabled:cursor-not-allowed focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-ring)]"
        >
          <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-hidden="true">
            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
          Clear filters
        </button>
      )}
    </div>
  );
}

function ChevronIcon() {
  return (
    <span className="pointer-events-none absolute inset-y-0 right-2.5 flex items-center text-[var(--text-muted)]">
      <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-hidden="true">
        <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
      </svg>
    </span>
  );
}
