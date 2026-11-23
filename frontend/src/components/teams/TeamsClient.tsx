"use client";

import { useCallback, useEffect, useReducer, useState } from "react";
import { teamsService } from "../../services/teams";
import { useAuthContext } from "../../context/AuthContext";
import { Card, CardHeader, CardBody } from "../ui/Card";
import { Button } from "../ui/Button";
import { Skeleton } from "../ui/Skeleton";
import { EmptyState } from "../ui/EmptyState";
import { ErrorState } from "../ui/ErrorState";
import { Dropdown } from "../ui/Dropdown";
import { TeamMemberList } from "./TeamMemberList";
import { TeamCreateModal } from "./TeamCreateModal";
import { TeamEditModal } from "./TeamEditModal";
import { TeamAddMemberModal } from "./TeamAddMemberModal";
import type { Team } from "../../types";

// ─── Fetch state ──────────────────────────────────────────────────────────────

interface TeamsState {
  teams: Team[];
  loading: boolean;
  error: string | null;
}

type TeamsAction =
  | { type: "LOADING" }
  | { type: "SUCCESS"; teams: Team[] }
  | { type: "ERROR"; message: string };

function teamsReducer(state: TeamsState, action: TeamsAction): TeamsState {
  switch (action.type) {
    case "LOADING": return { ...state, loading: true, error: null };
    case "SUCCESS": return { teams: action.teams, loading: false, error: null };
    case "ERROR":   return { ...state, loading: false, error: action.message };
  }
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

// ─── Team card ────────────────────────────────────────────────────────────────

interface TeamCardProps {
  team: Team;
  isAdmin: boolean;
  onEdit: (team: Team) => void;
  onAddMember: (team: Team) => void;
}

function TeamCard({ team, isAdmin, onEdit, onAddMember }: TeamCardProps) {
  const [expanded, setExpanded] = useState(false);
  const [memberRefreshKey, setMemberRefreshKey] = useState(0);

  const adminItems = [
    {
      label: "Edit team",
      onClick: () => onEdit(team),
      icon: (
        <svg fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-hidden="true">
          <path strokeLinecap="round" strokeLinejoin="round" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
        </svg>
      ),
    },
    {
      label: "Add member",
      onClick: () => onAddMember(team),
      icon: (
        <svg fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-hidden="true">
          <path strokeLinecap="round" strokeLinejoin="round" d="M18 9v3m0 0v3m0-3h3m-3 0h-3m-2-5a4 4 0 11-8 0 4 4 0 018 0zM3 20a6 6 0 0112 0v1H3v-1z" />
        </svg>
      ),
    },
  ];

  return (
    <Card padding={false}>
      <CardHeader>
        <div className="flex items-start justify-between gap-3">
          {/* Team info */}
          <div className="min-w-0 flex-1">
            <h2 className="truncate text-sm font-semibold text-[var(--text-primary)]">
              {team.name}
            </h2>
            {team.description && (
              <p className="mt-0.5 line-clamp-2 text-xs text-[var(--text-muted)]">
                {team.description}
              </p>
            )}
            <p className="mt-1.5 text-xs text-[var(--text-muted)]">
              Created {formatDate(team.created_at)}
            </p>
          </div>

          {/* Actions */}
          <div className="flex shrink-0 items-center gap-2">
            {isAdmin && (
              <Dropdown
                align="right"
                trigger={
                  <Button variant="ghost" size="sm" aria-label={`Actions for ${team.name}`}>
                    <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-hidden="true">
                      <path strokeLinecap="round" strokeLinejoin="round" d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
                    </svg>
                  </Button>
                }
                items={adminItems}
              />
            )}
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setExpanded((v) => !v)}
              aria-expanded={expanded}
              aria-label={expanded ? "Collapse members" : "Expand members"}
            >
              <svg
                className={`h-4 w-4 transition-transform duration-200 ${expanded ? "rotate-180" : ""}`}
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2}
                aria-hidden="true"
              >
                <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
              </svg>
            </Button>
          </div>
        </div>
      </CardHeader>

      {expanded && (
        <CardBody>
          <div className="mb-3 flex items-center justify-between">
            <h3 className="text-xs font-semibold uppercase tracking-wide text-[var(--text-muted)]">
              Members
            </h3>
            {isAdmin && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onAddMember(team)}
                className="text-xs"
              >
                <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-hidden="true">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
                </svg>
                Add member
              </Button>
            )}
          </div>
          <TeamMemberList
            teamId={team.id}
            isAdmin={isAdmin}
            refreshKey={memberRefreshKey}
            onMemberRemoved={() => setMemberRefreshKey((k) => k + 1)}
          />
        </CardBody>
      )}
    </Card>
  );
}

// ─── Skeleton card ────────────────────────────────────────────────────────────

function TeamCardSkeleton() {
  return (
    <div className="rounded-[var(--radius-lg)] border border-[var(--border)] bg-[var(--bg-surface)] shadow-[var(--shadow-sm)]">
      <div className="border-b border-[var(--border)] px-5 py-4">
        <div className="flex items-start justify-between gap-3">
          <div className="flex flex-col gap-2 flex-1">
            <Skeleton className="h-4 w-40" />
            <Skeleton className="h-3 w-64" />
            <Skeleton className="h-3 w-24" />
          </div>
          <Skeleton className="h-8 w-8 rounded-[var(--radius)]" />
        </div>
      </div>
    </div>
  );
}

// ─── Main component ───────────────────────────────────────────────────────────

export function TeamsClient() {
  const { isAdmin } = useAuthContext();
  const [state, dispatch] = useReducer(teamsReducer, {
    teams: [],
    loading: true,
    error: null,
  });
  const [fetchKey, setFetchKey] = useState(0);

  // Modal state
  const [createOpen, setCreateOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<Team | null>(null);
  const [addMemberTarget, setAddMemberTarget] = useState<Team | null>(null);

  useEffect(() => {
    let cancelled = false;
    dispatch({ type: "LOADING" });

    teamsService
      .list()
      .then((res) => {
        if (!cancelled) dispatch({ type: "SUCCESS", teams: res.data });
      })
      .catch((err: unknown) => {
        if (!cancelled)
          dispatch({
            type: "ERROR",
            message: err instanceof Error ? err.message : "Failed to load teams.",
          });
      });

    return () => { cancelled = true; };
  }, [fetchKey]);

  const refresh = useCallback(() => setFetchKey((k) => k + 1), []);

  function handleCreateSuccess(team: Team) {
    setCreateOpen(false);
    // Optimistically prepend then refresh to get server-ordered list.
    dispatch({ type: "SUCCESS", teams: [team, ...state.teams] });
    refresh();
  }

  function handleEditSuccess(updated: Team) {
    setEditTarget(null);
    dispatch({
      type: "SUCCESS",
      teams: state.teams.map((t) => (t.id === updated.id ? updated : t)),
    });
  }

  function handleAddMemberSuccess() {
    setAddMemberTarget(null);
    // Member list refresh is handled inside TeamCard via its own refreshKey.
    // No full team list refresh needed.
  }

  const { teams, loading, error } = state;

  return (
    <div className="flex flex-col gap-6">
      {/* Page header */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-xl font-bold text-[var(--text-primary)]">Teams</h1>
          <p className="mt-0.5 text-sm text-[var(--text-muted)]">
            Browse support teams and their members.
          </p>
        </div>

        <div className="flex items-center gap-2 self-start sm:self-auto">
          <button
            onClick={refresh}
            disabled={loading}
            aria-label="Refresh teams"
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

          {isAdmin && (
            <Button
              variant="primary"
              size="md"
              onClick={() => setCreateOpen(true)}
            >
              <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-hidden="true">
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
              </svg>
              Create Team
            </Button>
          )}
        </div>
      </div>

      {/* Content */}
      {error ? (
        <ErrorState message={error} onRetry={refresh} />
      ) : loading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <TeamCardSkeleton key={i} />
          ))}
        </div>
      ) : teams.length === 0 ? (
        <EmptyState
          title="No teams yet"
          description={
            isAdmin
              ? "Create your first team to start organising agents."
              : "No teams have been created yet."
          }
          icon={
            <svg fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5} aria-hidden="true">
              <path strokeLinecap="round" strokeLinejoin="round" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
          }
          action={
            isAdmin ? (
              <Button variant="primary" size="md" onClick={() => setCreateOpen(true)}>
                Create Team
              </Button>
            ) : undefined
          }
        />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {teams.map((team) => (
            <TeamCard
              key={team.id}
              team={team}
              isAdmin={isAdmin}
              onEdit={setEditTarget}
              onAddMember={setAddMemberTarget}
            />
          ))}
        </div>
      )}

      {/* Modals */}
      <TeamCreateModal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onSuccess={handleCreateSuccess}
      />

      {editTarget && (
        <TeamEditModal
          key={editTarget.id}
          open={editTarget !== null}
          team={editTarget}
          onClose={() => setEditTarget(null)}
          onSuccess={handleEditSuccess}
        />
      )}

      {addMemberTarget && (
        <TeamAddMemberModal
          open={addMemberTarget !== null}
          teamId={addMemberTarget.id}
          onClose={() => setAddMemberTarget(null)}
          onSuccess={handleAddMemberSuccess}
        />
      )}
    </div>
  );
}
