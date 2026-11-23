"use client";

import { useCallback, useEffect, useReducer, useState } from "react";
import { teamsService } from "../../services/teams";
import { ApiError } from "../../lib/api-client";
import { Avatar } from "../ui/Avatar";
import { Button } from "../ui/Button";
import { Skeleton } from "../ui/Skeleton";
import { EmptyState } from "../ui/EmptyState";
import { TeamRoleBadge } from "./TeamRoleBadge";
import type { TeamMember } from "../../types";

// ─── State ────────────────────────────────────────────────────────────────────

interface MemberState {
  members: TeamMember[];
  loading: boolean;
  error: string | null;
}

type MemberAction =
  | { type: "LOADING" }
  | { type: "SUCCESS"; members: TeamMember[] }
  | { type: "ERROR"; message: string };

function memberReducer(state: MemberState, action: MemberAction): MemberState {
  switch (action.type) {
    case "LOADING": return { ...state, loading: true, error: null };
    case "SUCCESS": return { members: action.members, loading: false, error: null };
    case "ERROR":   return { ...state, loading: false, error: action.message };
  }
}

// ─── Props ────────────────────────────────────────────────────────────────────

interface TeamMemberListProps {
  teamId: string;
  isAdmin: boolean;
  refreshKey: number;
  onMemberRemoved: () => void;
}

// ─── Component ────────────────────────────────────────────────────────────────

export function TeamMemberList({
  teamId,
  isAdmin,
  refreshKey,
  onMemberRemoved,
}: TeamMemberListProps) {
  const [state, dispatch] = useReducer(memberReducer, {
    members: [],
    loading: true,
    error: null,
  });
  const [removingId, setRemovingId] = useState<string | null>(null);
  const [removeError, setRemoveError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    dispatch({ type: "LOADING" });

    teamsService
      .listMembers(teamId)
      .then((res) => {
        if (!cancelled) dispatch({ type: "SUCCESS", members: res.data });
      })
      .catch((err: unknown) => {
        if (!cancelled)
          dispatch({
            type: "ERROR",
            message: err instanceof Error ? err.message : "Failed to load members.",
          });
      });

    return () => { cancelled = true; };
  }, [teamId, refreshKey]);

  const handleRemove = useCallback(
    async (userId: string, userName: string) => {
      if (!window.confirm(`Remove ${userName} from this team?`)) return;
      setRemovingId(userId);
      setRemoveError(null);
      try {
        await teamsService.removeMember(teamId, userId);
        onMemberRemoved();
      } catch (err) {
        let message = "Failed to remove member. Please try again.";
        if (err instanceof ApiError) {
          if (err.status === 403) message = "Admin access required.";
          else if (err.status === 404) message = "Membership not found.";
          else message = err.message || message;
        }
        setRemoveError(message);
      } finally {
        setRemovingId(null);
      }
    },
    [teamId, onMemberRemoved],
  );

  const { members, loading, error } = state;

  if (loading) {
    return (
      <div className="flex flex-col gap-3 py-1">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i} className="flex items-center gap-3">
            <Skeleton className="h-8 w-8 rounded-full shrink-0" />
            <div className="flex flex-col gap-1.5 flex-1">
              <Skeleton className="h-3 w-32" />
              <Skeleton className="h-3 w-48" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <p className="py-2 text-xs text-[var(--danger)]" role="alert">
        {error}
      </p>
    );
  }

  if (members.length === 0) {
    return (
      <EmptyState
        title="No members yet"
        description="Add members to this team using the button above."
        className="border-0 bg-transparent py-6 shadow-none"
      />
    );
  }

  return (
    <div className="flex flex-col gap-0.5">
      {removeError && (
        <p className="mb-2 text-xs text-[var(--danger)]" role="alert">
          {removeError}
        </p>
      )}
      {members.map((m) => {
        const name = m.user?.full_name ?? m.user_id.slice(0, 8) + "…";
        const email = m.user?.email ?? "—";
        const isRemoving = removingId === m.user_id;

        return (
          <div
            key={m.user_id}
            className="flex items-center gap-3 rounded-[var(--radius)] px-2 py-2.5 transition-colors hover:bg-[var(--bg-subtle)]"
          >
            <Avatar name={name} size="sm" />
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-medium text-[var(--text-primary)]">
                {name}
              </p>
              <p className="truncate text-xs text-[var(--text-muted)]">{email}</p>
            </div>
            <TeamRoleBadge role={m.team_role as "lead" | "member"} />
            {isAdmin && (
              <Button
                variant="ghost"
                size="sm"
                loading={isRemoving}
                disabled={removingId !== null}
                onClick={() => handleRemove(m.user_id, name)}
                aria-label={`Remove ${name} from team`}
                className="shrink-0 text-[var(--danger)] hover:bg-[var(--danger-bg)] hover:text-[var(--danger)]"
              >
                Remove
              </Button>
            )}
          </div>
        );
      })}
    </div>
  );
}
