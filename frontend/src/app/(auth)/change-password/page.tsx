"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { ApiError, useAuthContext } from "../../../context/AuthContext";
import { authService } from "../../../services/auth";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";

const MIN_PASSWORD_LENGTH = 8;

export default function ChangePasswordPage() {
  const { updateSession } = useAuthContext();
  const router = useRouter();

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    if (newPassword.length < MIN_PASSWORD_LENGTH) {
      setError(`New password must be at least ${MIN_PASSWORD_LENGTH} characters.`);
      return;
    }
    if (newPassword !== confirmPassword) {
      setError("New password and confirmation do not match.");
      return;
    }

    setLoading(true);
    try {
      const { user, access_token } = await authService.changePassword({
        current_password: currentPassword,
        new_password: newPassword,
        confirm_password: confirmPassword,
      });
      updateSession(user, access_token);
      // Clear fields before navigating away.
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      router.replace("/dashboard");
    } catch (err) {
      if (err instanceof ApiError) {
        // 422 = incorrect current password
        setError(
          err.status === 422
            ? "Current password is incorrect."
            : err.message,
        );
      } else {
        setError("An unexpected error occurred. Please try again.");
      }
      // Clear only the password fields on error, not the current-password field,
      // so the user can correct just the new password if that was the issue.
      setNewPassword("");
      setConfirmPassword("");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="rounded-[var(--radius-xl)] border border-[var(--border)] bg-[var(--bg-surface)] p-8 shadow-[var(--shadow-md)]">
      {/* Header */}
      <div className="mb-8 text-center">
        <div className="mx-auto mb-4 flex h-10 w-10 items-center justify-center rounded-xl bg-[var(--accent)]">
          <svg
            className="h-5 w-5 text-white"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={2}
            aria-hidden="true"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"
            />
          </svg>
        </div>
        <h1 className="text-xl font-semibold text-[var(--text-primary)]">
          Set a new password
        </h1>
        <p className="mt-1 text-sm text-[var(--text-muted)]">
          Your account requires a password change before you can continue.
        </p>
      </div>

      <form onSubmit={handleSubmit} noValidate className="flex flex-col gap-4">
        <Input
          label="Current password"
          type="password"
          autoComplete="current-password"
          value={currentPassword}
          onChange={(e) => setCurrentPassword(e.target.value)}
          required
          disabled={loading}
        />
        <Input
          label="New password"
          type="password"
          autoComplete="new-password"
          value={newPassword}
          onChange={(e) => setNewPassword(e.target.value)}
          required
          disabled={loading}
        />
        <Input
          label="Confirm new password"
          type="password"
          autoComplete="new-password"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          required
          disabled={loading}
        />

        {error && (
          <p
            role="alert"
            className="rounded-[var(--radius)] bg-[var(--danger-bg)] px-3 py-2 text-sm text-[var(--danger)]"
          >
            {error}
          </p>
        )}

        <Button type="submit" fullWidth loading={loading} className="mt-1">
          Change password
        </Button>
      </form>
    </div>
  );
}
