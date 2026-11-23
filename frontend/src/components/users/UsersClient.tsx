"use client";

import { useState } from "react";
import { useAuthContext } from "../../context/AuthContext";
import { useRouter } from "next/navigation";
import { Badge } from "../ui/Badge";
import { Button } from "../ui/Button";
import { UserProvisionModal } from "./UserProvisionModal";
import type { User } from "../../types";

interface SuccessConfirmationProps {
  user: User;
  onCreateAnother: () => void;
}

function SuccessConfirmation({ user, onCreateAnother }: SuccessConfirmationProps) {
  const roleBadgeVariant =
    user.role === "admin" ? ("danger" as const) : ("neutral" as const);

  return (
    <div className="flex flex-col gap-4 rounded-[var(--radius-lg)] border border-green-200 bg-[var(--success-bg)] p-5">
      <div className="flex items-start gap-3">
        <svg
          className="mt-0.5 h-5 w-5 shrink-0 text-[var(--success)]"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          strokeWidth={2}
          aria-hidden="true"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
        <div className="flex flex-col gap-1">
          <p className="text-sm font-semibold text-[var(--success)]">
            Account created
          </p>
          <p className="text-sm text-[var(--text-secondary)]">
            {user.full_name} &middot; {user.email}
          </p>
          <Badge variant={roleBadgeVariant} className="mt-1 w-fit capitalize">
            {user.role}
          </Badge>
        </div>
      </div>

      <p className="rounded-[var(--radius)] border border-amber-200 bg-[var(--warning-bg)] px-3 py-2.5 text-sm text-[var(--warning)]">
        <strong>Share the temporary password securely.</strong> There is no
        email onboarding or password-reset mechanism — the administrator must
        communicate the initial password to the user out of band.
      </p>

      <div className="flex justify-end">
        <Button variant="primary" size="md" onClick={onCreateAnother}>
          Create another user
        </Button>
      </div>
    </div>
  );
}

export function UsersClient() {
  const { isAdmin, isLoading } = useAuthContext();
  const router = useRouter();
  const [modalOpen, setModalOpen] = useState(false);
  const [lastProvisioned, setLastProvisioned] = useState<User | null>(null);

  // Wait for auth hydration before making any access decision.
  if (isLoading) return null;

  // Non-admins cannot access this page — redirect silently.
  // Sidebar already hides the link; this is a defence-in-depth guard.
  if (!isAdmin) {
    router.replace("/dashboard");
    return null;
  }

  function handleSuccess(user: User) {
    setModalOpen(false);
    setLastProvisioned(user);
  }

  function handleCreateAnother() {
    setLastProvisioned(null);
    setModalOpen(true);
  }

  return (
    <div className="flex flex-col gap-6">
      {/* Page header */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-xl font-bold text-[var(--text-primary)]">
            Users
          </h1>
          <p className="mt-0.5 text-sm text-[var(--text-muted)]">
            Provision new agent and admin accounts.
          </p>
        </div>
        {!lastProvisioned && (
          <Button
            variant="primary"
            size="md"
            onClick={() => setModalOpen(true)}
            className="self-start sm:self-auto"
          >
            <svg
              className="h-4 w-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
              aria-hidden="true"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M12 4v16m8-8H4"
              />
            </svg>
            Create User
          </Button>
        )}
      </div>

      {lastProvisioned ? (
        <SuccessConfirmation
          user={lastProvisioned}
          onCreateAnother={handleCreateAnother}
        />
      ) : (
        <p className="rounded-[var(--radius)] border border-[var(--border)] bg-[var(--bg-subtle)] px-4 py-3 text-sm text-[var(--text-muted)]">
          User accounts are managed via team membership. Use the button above
          to provision a new account.
        </p>
      )}

      <UserProvisionModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        onSuccess={handleSuccess}
      />
    </div>
  );
}
