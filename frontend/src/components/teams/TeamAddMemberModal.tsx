"use client";

import { useEffect, useRef, useState } from "react";
import { teamsService } from "../../services/teams";
import { ApiError } from "../../lib/api-client";
import { Modal } from "../ui/Modal";
import { Input } from "../ui/Input";
import { Button } from "../ui/Button";
import type { TeamMemberRole } from "../../types";

// UUID v4 pattern — permissive enough for all UUID versions.
const UUID_RE =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

const ROLE_OPTIONS: { value: TeamMemberRole; label: string }[] = [
  { value: "member", label: "Member" },
  { value: "lead", label: "Lead" },
];

const selectClass =
  "w-full h-9 rounded-[var(--radius)] border border-[var(--border)] bg-white px-3 pr-8 text-sm text-[var(--text-primary)] shadow-[var(--shadow-sm)] transition-colors hover:border-[var(--border-strong)] focus:outline-none focus:ring-2 focus:ring-[var(--accent-ring)] focus:border-[var(--accent)] disabled:opacity-50 disabled:cursor-not-allowed appearance-none cursor-pointer";

interface TeamAddMemberModalProps {
  open: boolean;
  teamId: string;
  onClose: () => void;
  onSuccess: () => void;
}

export function TeamAddMemberModal({
  open,
  teamId,
  onClose,
  onSuccess,
}: TeamAddMemberModalProps) {
  const [userId, setUserId] = useState("");
  const [role, setRole] = useState<TeamMemberRole>("member");
  const [userIdError, setUserIdError] = useState<string | undefined>();
  const [apiError, setApiError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (open) {
      const id = setTimeout(() => inputRef.current?.focus(), 0);
      return () => clearTimeout(id);
    }
  }, [open]);

  function handleClose() {
    if (submitting) return;
    setUserId("");
    setRole("member");
    setUserIdError(undefined);
    setApiError(null);
    onClose();
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (submitting) return;

    const trimmed = userId.trim();
    if (!trimmed) {
      setUserIdError("User ID is required.");
      return;
    }
    if (!UUID_RE.test(trimmed)) {
      setUserIdError("Enter a valid UUID (e.g. 550e8400-e29b-41d4-a716-446655440000).");
      return;
    }

    setUserIdError(undefined);
    setApiError(null);
    setSubmitting(true);

    try {
      await teamsService.addMember(teamId, { user_id: trimmed, team_role: role });
      setUserId("");
      setRole("member");
      setApiError(null);
      onSuccess();
    } catch (err) {
      let message = "An unexpected error occurred. Please try again.";
      if (err instanceof ApiError) {
        if (err.status === 403) message = "Admin access required.";
        else if (err.status === 404) message = err.message.includes("user") ? "User not found. Check the UUID and try again." : "Team not found.";
        else if (err.status === 409) message = "This user is already a member of the team.";
        else if (err.status === 400) message = err.message;
        else message = err.message || message;
      }
      setApiError(message);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Modal open={open} onClose={handleClose} title="Add Member" size="sm">
      <form onSubmit={handleSubmit} noValidate>
        <div className="flex flex-col gap-4">
          <Input
            ref={inputRef}
            id="add-member-user-id"
            label="User ID"
            type="text"
            value={userId}
            onChange={(e) => {
              setUserId(e.target.value);
              if (userIdError) setUserIdError(undefined);
              if (apiError) setApiError(null);
            }}
            placeholder="550e8400-e29b-41d4-a716-446655440000"
            hint="Paste the UUID of the user to add."
            disabled={submitting}
            error={userIdError}
            aria-required="true"
          />

          {/* Role */}
          <div className="flex flex-col gap-1.5">
            <label
              htmlFor="add-member-role"
              className="text-sm font-medium text-[var(--text-primary)]"
            >
              Role
            </label>
            <div className="relative">
              <select
                id="add-member-role"
                value={role}
                onChange={(e) => setRole(e.target.value as TeamMemberRole)}
                disabled={submitting}
                className={selectClass}
              >
                {ROLE_OPTIONS.map((o) => (
                  <option key={o.value} value={o.value}>
                    {o.label}
                  </option>
                ))}
              </select>
              <span className="pointer-events-none absolute inset-y-0 right-2.5 flex items-center text-[var(--text-muted)]">
                <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-hidden="true">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
                </svg>
              </span>
            </div>
          </div>

          {apiError && (
            <p role="alert" className="rounded-[var(--radius)] bg-[var(--danger-bg)] px-3 py-2 text-sm text-[var(--danger)]">
              {apiError}
            </p>
          )}

          <div className="flex justify-end gap-3 pt-1">
            <Button type="button" variant="secondary" size="md" onClick={handleClose} disabled={submitting}>
              Cancel
            </Button>
            <Button type="submit" variant="primary" size="md" loading={submitting} disabled={submitting}>
              Add Member
            </Button>
          </div>
        </div>
      </form>
    </Modal>
  );
}
