"use client";

import { useEffect, useReducer, useState } from "react";
import Link from "next/link";
import { ticketsService } from "../../services/tickets";
import { teamsService } from "../../services/teams";
import { ApiError } from "../../lib/api-client";
import { useAuthContext } from "../../context/AuthContext";
import { Card, CardHeader, CardBody } from "../ui/Card";
import { ErrorState } from "../ui/ErrorState";
import { TicketStatusBadge } from "./TicketStatusBadge";
import { TicketPriorityBadge } from "./TicketPriorityBadge";
import { TicketUpdateControls } from "./TicketUpdateControls";
import { TicketAssignmentControls } from "./TicketAssignmentControls";
import { TicketAIPrediction } from "./TicketAIPrediction";
import { TicketHistoryTimeline } from "./TicketHistoryTimeline";
import { TicketCommentThread } from "./TicketCommentThread";
import { TicketCommentComposer } from "./TicketCommentComposer";
import { TicketDetailSkeleton } from "./TicketDetailSkeleton";
import type { Team, Ticket } from "../../types";

// ─── Ticket fetch state ───────────────────────────────────────────────────────

interface TicketState {
  ticket: Ticket | null;
  loading: boolean;
  error: string | null;
  notFound: boolean;
}

type TicketAction =
  | { type: "LOADING" }
  | { type: "SUCCESS"; ticket: Ticket }
  | { type: "NOT_FOUND" }
  | { type: "ERROR"; message: string };

function ticketReducer(state: TicketState, action: TicketAction): TicketState {
  switch (action.type) {
    case "LOADING":   return { ticket: null,          loading: true,  error: null,          notFound: false };
    case "SUCCESS":   return { ticket: action.ticket, loading: false, error: null,          notFound: false };
    case "NOT_FOUND": return { ticket: null,          loading: false, error: null,          notFound: true  };
    case "ERROR":     return { ticket: null,          loading: false, error: action.message, notFound: false };
  }
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("en-US", {
    month: "long", day: "numeric", year: "numeric",
  });
}

function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString("en-US", {
    month: "short", day: "numeric", year: "numeric",
    hour: "numeric", minute: "2-digit",
  });
}

function formatCategory(cat: string | null | undefined): string {
  if (!cat) return "—";
  return cat.split("_").map((w) => w[0].toUpperCase() + w.slice(1)).join(" ");
}

function formatChannel(ch: string): string {
  return ch.charAt(0).toUpperCase() + ch.slice(1);
}

function MetaRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-start justify-between gap-4 py-2.5 border-b border-[var(--border)] last:border-0">
      <span className="shrink-0 text-xs text-[var(--text-muted)]">{label}</span>
      <span className="text-right text-xs font-medium text-[var(--text-primary)]">{children}</span>
    </div>
  );
}

// ─── Component ────────────────────────────────────────────────────────────────

export function TicketDetailClient({ ticketId }: { ticketId: string }) {
  const { user, isAdmin } = useAuthContext();
  const [state, dispatch] = useReducer(ticketReducer, {
    ticket: null, loading: true, error: null, notFound: false,
  });
  const [teams, setTeams] = useState<Team[]>([]);
  const [fetchKey, setFetchKey] = useState(0);
  const [commentRefreshKey, setCommentRefreshKey] = useState(0);

  // Fetch teams once — non-fatal.
  useEffect(() => {
    teamsService
      .list()
      .then((res) => {
        setTeams(res.data);
      })
      .catch(() => undefined);
  }, []);

  // Fetch ticket.
  useEffect(() => {
    let cancelled = false;

    dispatch({ type: "LOADING" });

    ticketsService
      .get(ticketId)
      .then((t) => {
        if (!cancelled) dispatch({ type: "SUCCESS", ticket: t });
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        if (err instanceof ApiError && (err.status === 404 || err.status === 400)) {
          dispatch({ type: "NOT_FOUND" });
        } else {
          dispatch({
            type: "ERROR",
            message: err instanceof Error ? err.message : "Failed to load ticket.",
          });
        }
      });

    return () => { cancelled = true; };
  }, [ticketId, fetchKey]);

  // ── Loading ──────────────────────────────────────────────────────────────────
  if (state.loading) return <TicketDetailSkeleton />;

  // ── Not found ────────────────────────────────────────────────────────────────
  if (state.notFound) {
    return (
      <div className="flex flex-col gap-4">
        <BackLink />
        <ErrorState
          title="Ticket not found"
          message="This ticket does not exist or has been deleted."
        />
      </div>
    );
  }

  // ── Error ────────────────────────────────────────────────────────────────────
  if (state.error) {
    return (
      <div className="flex flex-col gap-4">
        <BackLink />
        <ErrorState
          message={state.error}
          onRetry={() => setFetchKey((k) => k + 1)}
        />
      </div>
    );
  }

  const ticket = state.ticket!;

  return (
    <div className="flex flex-col gap-6">
      {/* ── Page header ──────────────────────────────────────────────────────── */}
      <div className="flex flex-col gap-3">
        <BackLink />

        <div className="flex flex-wrap items-start justify-between gap-4">
          {/* Title block */}
          <div className="flex flex-wrap items-center gap-3 min-w-0">
            <span className="font-mono text-sm font-semibold text-[var(--text-muted)] shrink-0">
              #{ticket.ticket_number}
            </span>
            <h1 className="text-lg font-bold text-[var(--text-primary)] leading-snug">
              {ticket.subject}
            </h1>
            <TicketStatusBadge status={ticket.status} />
            <TicketPriorityBadge priority={ticket.priority} />
          </div>

          {/* Refresh */}
          <button
            onClick={() => setFetchKey((k) => k + 1)}
            aria-label="Refresh ticket"
            className="inline-flex h-9 shrink-0 items-center gap-2 rounded-[var(--radius)] border border-[var(--border)] bg-[var(--bg-surface)] px-3 text-xs font-medium text-[var(--text-secondary)] shadow-[var(--shadow-sm)] transition-colors hover:bg-[var(--bg-subtle)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-ring)]"
          >
            <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-hidden="true">
              <path strokeLinecap="round" strokeLinejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            Refresh
          </button>
        </div>
      </div>

      {/* ── Two-column body ───────────────────────────────────────────────────── */}
      <div className="grid gap-5 lg:grid-cols-3">

        {/* ── Main column ──────────────────────────────────────────────────────── */}
        <div className="flex flex-col gap-5 lg:col-span-2">

          {/* Description */}
          <Card padding={false}>
            <CardHeader>
              <h2 className="text-sm font-semibold text-[var(--text-primary)]">Description</h2>
            </CardHeader>
            <CardBody>
              <p className="whitespace-pre-wrap text-sm leading-relaxed text-[var(--text-secondary)]">
                {ticket.description}
              </p>
            </CardBody>
          </Card>

          {/* Customer + ticket metadata */}
          <Card padding={false}>
            <CardHeader>
              <h2 className="text-sm font-semibold text-[var(--text-primary)]">Details</h2>
            </CardHeader>
            <CardBody>
              <MetaRow label="Customer">{ticket.customer_name}</MetaRow>
              <MetaRow label="Email">
                <a
                  href={`mailto:${ticket.customer_email}`}
                  className="text-[var(--accent)] hover:underline focus-visible:outline-none"
                >
                  {ticket.customer_email}
                </a>
              </MetaRow>
              <MetaRow label="Channel">{formatChannel(ticket.channel)}</MetaRow>
              <MetaRow label="Category">{formatCategory(ticket.category)}</MetaRow>
              <MetaRow label="Created">{formatDate(ticket.created_at)}</MetaRow>
              <MetaRow label="Last updated">{formatDateTime(ticket.updated_at)}</MetaRow>
              {ticket.resolved_at && (
                <MetaRow label="Resolved">{formatDateTime(ticket.resolved_at)}</MetaRow>
              )}
            </CardBody>
          </Card>

          {/* History */}
          <TicketHistoryTimeline ticketId={ticket.id} />

          {/* Comments */}
          <TicketCommentThread
            ticketId={ticket.id}
            currentUserId={user?.id}
            currentUserName={user?.full_name}
            refreshKey={commentRefreshKey}
          />
          <TicketCommentComposer
            ticketId={ticket.id}
            onSuccess={() => setCommentRefreshKey((k) => k + 1)}
          />
        </div>

        {/* ── Side column ──────────────────────────────────────────────────────── */}
        <div className="flex flex-col gap-5">

          {/* Status + priority controls */}
          <Card padding={false}>
            <CardHeader>
              <h2 className="text-sm font-semibold text-[var(--text-primary)]">Update ticket</h2>
            </CardHeader>
            <CardBody>
              <TicketUpdateControls
                ticket={ticket}
                onSuccess={(updated) => dispatch({ type: "SUCCESS", ticket: updated })}
              />
            </CardBody>
          </Card>

          {/* Assignment */}
          <Card padding={false}>
            <CardHeader>
              <h2 className="text-sm font-semibold text-[var(--text-primary)]">Assignment</h2>
            </CardHeader>
            <CardBody>
              <TicketAssignmentControls
                ticket={ticket}
                teams={teams}
                isAdmin={isAdmin}
                onSuccess={(updated) => dispatch({ type: "SUCCESS", ticket: updated })}
              />
            </CardBody>
          </Card>

          {/* AI prediction */}
          <TicketAIPrediction ticketId={ticket.id} />
        </div>
      </div>
    </div>
  );
}

function BackLink() {
  return (
    <Link
      href="/tickets"
      className="inline-flex items-center gap-1.5 text-xs font-medium text-[var(--text-muted)] transition-colors hover:text-[var(--text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-ring)] rounded"
    >
      <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-hidden="true">
        <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7" />
      </svg>
      Back to Tickets
    </Link>
  );
}
