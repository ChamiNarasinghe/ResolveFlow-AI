"use client";

import { useEffect, useRef, useState } from "react";
import { adminService } from "../../services/admin";
import { ApiError } from "../../lib/api-client";
import { Modal } from "../ui/Modal";
import { Input } from "../ui/Input";
import { Button } from "../ui/Button";
import type { ProvisionableRole, User } from "../../types";

const ROLE_OPTIONS: { value: ProvisionableRole; label: string }[] = [
  { value: "agent", label: "Agent" },
  { value: "admin", label: "Admin" },
];

const selectClass =
  "w-full h-9 rounded-[var(--radius)] border border-[var(--border)] bg-white px-3 pr-8 text-sm text-[var(--text-primary)] shadow-[var(--shadow-sm)] transition-colors hover:border-[var(--border-strong)] focus:outline-none focus:ring-2 focus:ring-[var(--accent-ring)] focus:border-[var(--accent)] disabled:opacity-50 disabled:cursor-not-allowed appearance-none cursor-pointer";

interface UserProvisionModalProps {
  open: boolean;
  onClose: () => void;
  onSuccess: (user: User) => void;
}

interface FieldErrors {
  name?: string;
  email?: string;
  password?: string;
}

export function UserProvisionModal({
  open,
  onClose,
  onSuccess,
}: UserProvisionModalProps) {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState<ProvisionableRole>("agent");
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [apiError, setApiError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const nameRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (open) {
      const id = setTimeout(() => nameRef.current?.focus(), 0);
      return () => clearTimeout(id);
    }
  }, [open]);

  function handleClose() {
    if (submitting) return;
    setName("");
    setEmail("");
    setPassword("");
    setRole("agent");
    setFieldErrors({});
    setApiError(null);
    onClose();
  }

  function validate(): FieldErrors {
    const errors: FieldErrors = {};
    if (!name.trim()) errors.name = "Name is required.";
    if (!email.trim()) errors.email = "Email is required.";
    else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim()))
      errors.email = "Enter a valid email address.";
    if (!password) errors.password = "Password is required.";
    else if (password.length < 8)
      errors.password = "Password must be at least 8 characters.";
    return errors;
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (submitting) return;

    const errors = validate();
    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      return;
    }

    setFieldErrors({});
    setApiError(null);
    setSubmitting(true);

    try {
      const { user } = await adminService.provisionUser({
        name: name.trim(),
        email: email.trim(),
        password,
        role,
      });
      setName("");
      setEmail("");
      setPassword("");
      setRole("agent");
      onSuccess(user);
    } catch (err) {
      let message = "An unexpected error occurred. Please try again.";
      if (err instanceof ApiError) {
        if (err.status === 403) message = "Admin access required.";
        else if (err.status === 409)
          message = "An account with this email already exists.";
        else if (err.status === 400) message = err.message;
        else message = err.message || message;
      }
      setApiError(message);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Modal open={open} onClose={handleClose} title="Create User" size="sm">
      <form onSubmit={handleSubmit} noValidate>
        <div className="flex flex-col gap-4">
          <Input
            ref={nameRef}
            id="provision-name"
            label="Full name"
            type="text"
            value={name}
            onChange={(e) => {
              setName(e.target.value);
              if (fieldErrors.name) setFieldErrors((p) => ({ ...p, name: undefined }));
              if (apiError) setApiError(null);
            }}
            placeholder="Jane Smith"
            disabled={submitting}
            error={fieldErrors.name}
            aria-required="true"
          />

          <Input
            id="provision-email"
            label="Email"
            type="email"
            value={email}
            onChange={(e) => {
              setEmail(e.target.value);
              if (fieldErrors.email) setFieldErrors((p) => ({ ...p, email: undefined }));
              if (apiError) setApiError(null);
            }}
            placeholder="jane@example.com"
            disabled={submitting}
            error={fieldErrors.email}
            aria-required="true"
          />

          <Input
            id="provision-password"
            label="Temporary password"
            type="password"
            value={password}
            onChange={(e) => {
              setPassword(e.target.value);
              if (fieldErrors.password)
                setFieldErrors((p) => ({ ...p, password: undefined }));
              if (apiError) setApiError(null);
            }}
            hint="Min 8 characters, must include a letter and a number."
            disabled={submitting}
            error={fieldErrors.password}
            aria-required="true"
          />

          {/* Role */}
          <div className="flex flex-col gap-1.5">
            <label
              htmlFor="provision-role"
              className="text-sm font-medium text-[var(--text-primary)]"
            >
              Role
            </label>
            <div className="relative">
              <select
                id="provision-role"
                value={role}
                onChange={(e) => setRole(e.target.value as ProvisionableRole)}
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
                    d="M19 9l-7 7-7-7"
                  />
                </svg>
              </span>
            </div>
          </div>

          {apiError && (
            <p
              role="alert"
              className="rounded-[var(--radius)] bg-[var(--danger-bg)] px-3 py-2 text-sm text-[var(--danger)]"
            >
              {apiError}
            </p>
          )}

          <div className="flex justify-end gap-3 pt-1">
            <Button
              type="button"
              variant="secondary"
              size="md"
              onClick={handleClose}
              disabled={submitting}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              variant="primary"
              size="md"
              loading={submitting}
              disabled={submitting}
            >
              Create User
            </Button>
          </div>
        </div>
      </form>
    </Modal>
  );
}
