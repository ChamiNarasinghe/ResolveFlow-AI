"use client";

import { useEffect, useState } from "react";
import { ticketsService } from "../../services/tickets";
import { teamsService } from "../../services/teams";
import { ApiError } from "../../lib/api-client";
import { Dropdown } from "../ui/Dropdown";
import { Button } from "../ui/Button";
import { Spinner } from "../ui/Spinner";
import type { Team, TeamMember, Ticket } from "../../types";

// ─── Props ────────────────────────────────────────────────────────────────────

interface TicketAssignmentControlsProps {
  ticket: Ticket;
  teams: Team[];
  isAdmin: boolean;
  onSuccess: (updated: Ticket) => void;
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function errorMessage(err: unknown, fallback: string): string {
  if (err instanceof ApiError) {
    if (err.status === 404) return "Ticket or team not found.";
    if (err.status === 403) return "Admin access required.";
    if (err.status === 400) return err.message;
  }
  return fallback;
}

// ─── Read-only row ────────────────────────────────────────────────────────────

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-start justify-between gap-4 py-2.5 border-b border-[var(--border)] last:border-0">
      <span className="shrink-0 text-xs text-[var(--text-muted)]">{label}</span>
      <span className="text-right text-xs font-medium text-[var(--text-primary)]">{children}</span>
    </div>
  );
}

// ─── Control row (label + dropdown) ──────────────────────────────────────────

function ControlRow({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-3 py-1.5">
      <span className="text-xs font-medium text-[var(--text-muted)]">{label}</span>
      {children}
    </div>
  );
}

// ─── Chevron icon ─────────────────────────────────────────────────────────────

function ChevronDown() {
  return (
    <svg
      className="h-3 w-3 text-[var(--text-muted)]"
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
      strokeWidth={2}
      aria-hidden="true"
    >
      <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
    </svg>
  );
}

// ─── Component ────────────────────────────────────────────────────────────────

export function TicketAssignmentControls({
  ticket,
  teams,
  isAdmin,
  onSuccess,
}: TicketAssignmentControlsProps) {
  // Members for the currently assigned team.
  // { status: "idle" } = no team assigned
  // { status: "loading" } = fetch in flight
  // { status: "ready", data: TeamMember[] } = fetch complete
  type MembersState =
    | { status: "idle" }
    | { status: "loading" }
    | { status: "ready"; data: TeamMember[] };

  const [membersState, setMembersState] = useState<MembersState>({ status: "idle" });

  // In-flight guard — true while any assignment request is running.
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Fetch members whenever the ticket's assigned team changes.
  // All setState calls happen inside async callbacks or the cleanup, never
  // synchronously in the effect body, satisfying react-hooks/set-state-in-effect.
  useEffect(() => {
    const teamId = ticket.assigned_team_id;
    if (!teamId) return;

    let cancelled = false;

    teamsService
      .listMembers(teamId)
      .then((res) => {
        if (!cancelled) setMembersState({ status: "ready", data: res.data });
      })
      .catch(() => {
        if (!cancelled) setMembersState({ status: "ready", data: [] });
      });

    // Signal loading only after the microtask queue — avoids synchronous setState.
    Promise.resolve().then(() => {
      if (!cancelled) setMembersState({ status: "loading" });
    });

    return () => {
      cancelled = true;
      setMembersState({ status: "idle" });
    };
  }, [ticket.assigned_team_id]);

  const members =
    membersState.status === "ready" ? membersState.data : null;
  const membersLoading = membersState.status === "loading";

  // ── Resolve display names ─────────────────────────────────────────────────

  const teamName = ticket.assigned_team_id
    ? (teams.find((t) => t.id === ticket.assigned_team_id)?.name ?? "Unknown team")
    : "Unassigned";

  const agentName = (() => {
    if (!ticket.assigned_to) return "Unassigned";
    const member = members?.find((m) => m.user_id === ticket.assigned_to);
    if (member?.user) return member.user.full_name;
    // Team not assigned or members not yet loaded — show truncated UUID.
    return `Agent (${ticket.assigned_to.slice(0, 8)}…)`;
  })();

  // ── Non-admin: read-only ──────────────────────────────────────────────────

  if (!isAdmin) {
    return (
      <div>
        <Row label="Team">{teamName}</Row>
        <Row label="Agent">{agentName}</Row>
      </div>
    );
  }

  // ── Admin: interactive controls ───────────────────────────────────────────

  async function handleTeamChange(newTeamId: string | null) {
    if (saving) return;
    setError(null);
    setSaving(true);

    try {
      // Step 1: assign (or unassign) the team via the dedicated endpoint.
      const afterTeam = await ticketsService.assign(ticket.id, {
        team_id: newTeamId,
      });

      // Step 2: if an agent was set and the team changed, clear the agent.
      // The backend does not do this automatically.
      if (ticket.assigned_to !== null) {
        const afterAgent = await ticketsService.update(ticket.id, {
          assigned_to: null,
        });
        onSuccess(afterAgent);
      } else {
        onSuccess(afterTeam);
      }
    } catch (err) {
      setError(errorMessage(err, "Failed to update team assignment."));
    } finally {
      setSaving(false);
    }
  }

  async function handleAgentChange(newUserId: string | null) {
    if (saving) return;
    setError(null);
    setSaving(true);

    try {
      const updated = await ticketsService.update(ticket.id, {
        assigned_to: newUserId,
      });
      onSuccess(updated);
    } catch (err) {
      setError(errorMessage(err, "Failed to update agent assignment."));
    } finally {
      setSaving(false);
    }
  }

  // ── Team dropdown items ───────────────────────────────────────────────────

  const teamItems = [
    {
      label: "Unassigned",
      disabled: ticket.assigned_team_id === null,
      onClick: () => handleTeamChange(null),
    },
    ...teams.map((t) => ({
      label: t.name,
      disabled: t.id === ticket.assigned_team_id,
      onClick: () => handleTeamChange(t.id),
    })),
  ];

  // ── Agent dropdown items ──────────────────────────────────────────────────

  const agentItems = ticket.assigned_team_id
    ? [
        {
          label: "Unassigned",
          disabled: ticket.assigned_to === null,
          onClick: () => handleAgentChange(null),
        },
        ...(members ?? [])
          .filter((m) => m.user?.is_active)
          .map((m) => ({
            label: m.user?.full_name ?? m.user_id,
            disabled: m.user_id === ticket.assigned_to,
            onClick: () => handleAgentChange(m.user_id),
          })),
      ]
    : [];

  const agentLabel = (() => {
    if (!ticket.assigned_team_id) return "Select a team first";
    if (membersLoading) return "Loading…";
    return agentName;
  })();

  return (
    <div className="flex flex-col gap-1">
      {/* Team */}
      <ControlRow label="Team">
        <Dropdown
          align="right"
          trigger={
            <Button
              variant="ghost"
              size="sm"
              loading={saving}
              disabled={saving}
              aria-label="Change team assignment"
              className="gap-1.5 max-w-[9rem] truncate"
            >
              <span className="truncate text-xs">{teamName}</span>
              <ChevronDown />
            </Button>
          }
          items={teamItems}
        />
      </ControlRow>

      {/* Agent */}
      <ControlRow label="Agent">
        {membersLoading ? (
          <Spinner size="sm" className="text-[var(--text-muted)]" />
        ) : (
          <Dropdown
            align="right"
            trigger={
              <Button
                variant="ghost"
                size="sm"
                loading={saving}
                disabled={saving || !ticket.assigned_team_id || membersLoading}
                aria-label="Change agent assignment"
                className="gap-1.5 max-w-[9rem] truncate"
              >
                <span className="truncate text-xs">{agentLabel}</span>
                {ticket.assigned_team_id && !membersLoading && <ChevronDown />}
              </Button>
            }
            items={agentItems}
          />
        )}
      </ControlRow>

      {/* Inline error */}
      {error && (
        <p role="alert" className="mt-1 text-xs text-[var(--danger)]">
          {error}
        </p>
      )}
    </div>
  );
}
