"use client";

import { useState } from "react";
import { ticketsService } from "../../services/tickets";
import { Dropdown } from "../ui/Dropdown";
import { Button } from "../ui/Button";
import { TicketStatusBadge } from "./TicketStatusBadge";
import { TicketPriorityBadge } from "./TicketPriorityBadge";
import type { Ticket, TicketPriority, TicketStatus } from "../../types";
import { ApiError } from "../../lib/api-client";

const STATUS_OPTIONS: { value: TicketStatus; label: string }[] = [
  { value: "open", label: "Open" },
  { value: "in_progress", label: "In Progress" },
  { value: "waiting_for_customer", label: "Waiting for Customer" },
  { value: "resolved", label: "Resolved" },
  { value: "closed", label: "Closed" },
];

const PRIORITY_OPTIONS: { value: TicketPriority; label: string }[] = [
  { value: "urgent", label: "Urgent" },
  { value: "high", label: "High" },
  { value: "medium", label: "Medium" },
  { value: "low", label: "Low" },
];

interface TicketUpdateControlsProps {
  ticket: Ticket;
  onSuccess: (updated: Ticket) => void;
}

type UpdateField = "status" | "priority";

interface UpdateState {
  field: UpdateField | null;
  error: string | null;
}

export function TicketUpdateControls({ ticket, onSuccess }: TicketUpdateControlsProps) {
  const [updating, setUpdating] = useState<UpdateState>({ field: null, error: null });

  async function handleUpdate(field: UpdateField, value: string) {
    if (updating.field !== null) return;

    setUpdating({ field, error: null });
    try {
      const updated = await ticketsService.update(ticket.id, { [field]: value });
      setUpdating({ field: null, error: null });
      onSuccess(updated);
    } catch (err) {
      let message = "Update failed. Please try again.";
      if (err instanceof ApiError) {
        if (err.status === 401) message = "Session expired. Please log in again.";
        else if (err.status === 400) message = err.message;
        else if (err.status === 404) message = "Ticket not found.";
      }
      setUpdating({ field: null, error: message });
    }
  }

  const isUpdating = updating.field !== null;

  return (
    <div className="flex flex-col gap-3">
      {/* Status */}
      <div className="flex items-center justify-between gap-3">
        <span className="text-xs font-medium text-[var(--text-muted)]">Status</span>
        <Dropdown
          align="right"
          trigger={
            <Button
              variant="ghost"
              size="sm"
              loading={updating.field === "status"}
              disabled={isUpdating}
              aria-label="Change status"
              className="gap-1.5"
            >
              <TicketStatusBadge status={ticket.status} />
              <svg className="h-3 w-3 text-[var(--text-muted)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-hidden="true">
                <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
              </svg>
            </Button>
          }
          items={STATUS_OPTIONS.map((o) => ({
            label: o.label,
            disabled: o.value === ticket.status,
            onClick: () => handleUpdate("status", o.value),
          }))}
        />
      </div>

      {/* Priority */}
      <div className="flex items-center justify-between gap-3">
        <span className="text-xs font-medium text-[var(--text-muted)]">Priority</span>
        <Dropdown
          align="right"
          trigger={
            <Button
              variant="ghost"
              size="sm"
              loading={updating.field === "priority"}
              disabled={isUpdating}
              aria-label="Change priority"
              className="gap-1.5"
            >
              <TicketPriorityBadge priority={ticket.priority} />
              <svg className="h-3 w-3 text-[var(--text-muted)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-hidden="true">
                <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
              </svg>
            </Button>
          }
          items={PRIORITY_OPTIONS.map((o) => ({
            label: o.label,
            disabled: o.value === ticket.priority,
            onClick: () => handleUpdate("priority", o.value),
          }))}
        />
      </div>

      {/* Inline error */}
      {updating.error && (
        <p role="alert" className="text-xs text-[var(--danger)]">
          {updating.error}
        </p>
      )}
    </div>
  );
}
