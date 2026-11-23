"use client";

import { useEffect, useRef, useState } from "react";
import { ticketsService } from "../../services/tickets";
import { ApiError } from "../../lib/api-client";
import { Modal } from "../ui/Modal";
import { Input } from "../ui/Input";
import { Textarea } from "../ui/Textarea";
import { Button } from "../ui/Button";
import type { Ticket, TicketChannel } from "../../types";

// ─── Channel options ──────────────────────────────────────────────────────────

const CHANNEL_OPTIONS: { value: TicketChannel; label: string }[] = [
  { value: "web", label: "Web" },
  { value: "email", label: "Email" },
  { value: "chat", label: "Chat" },
  { value: "phone", label: "Phone" },
];

// Matches the select styling used in TicketFilters.tsx.
const selectClass =
  "w-full h-9 rounded-[var(--radius)] border border-[var(--border)] bg-white px-3 pr-8 text-sm text-[var(--text-primary)] shadow-[var(--shadow-sm)] transition-colors hover:border-[var(--border-strong)] focus:outline-none focus:ring-2 focus:ring-[var(--accent-ring)] focus:border-[var(--accent)] disabled:opacity-50 disabled:cursor-not-allowed appearance-none cursor-pointer";

// ─── Form state ───────────────────────────────────────────────────────────────

interface FormValues {
  subject: string;
  description: string;
  customer_name: string;
  customer_email: string;
  channel: TicketChannel;
}

interface FormErrors {
  subject?: string;
  description?: string;
  customer_name?: string;
  customer_email?: string;
}

const EMPTY_VALUES: FormValues = {
  subject: "",
  description: "",
  customer_name: "",
  customer_email: "",
  channel: "web",
};

// ─── Validation ───────────────────────────────────────────────────────────────

// Basic email check — intentionally permissive. The backend (net/mail) is the
// authoritative validator; this only catches obvious typos before the request.
function isValidEmail(value: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);
}

function validate(values: FormValues): FormErrors {
  const errors: FormErrors = {};

  if (values.subject.trim() === "") {
    errors.subject = "Subject is required";
  } else if (values.subject.trim().length > 255) {
    errors.subject = "Subject must be 255 characters or fewer";
  }

  if (values.description.trim() === "") {
    errors.description = "Description is required";
  }

  if (values.customer_name.trim() === "") {
    errors.customer_name = "Customer name is required";
  }

  if (values.customer_email.trim() === "") {
    errors.customer_email = "Customer email is required";
  } else if (!isValidEmail(values.customer_email.trim())) {
    errors.customer_email = "Enter a valid email address";
  }

  return errors;
}

function hasErrors(errors: FormErrors): boolean {
  return Object.keys(errors).length > 0;
}

// ─── Props ────────────────────────────────────────────────────────────────────

interface TicketCreateModalProps {
  open: boolean;
  onClose: () => void;
  onSuccess: (ticket: Ticket) => void;
}

// ─── Component ────────────────────────────────────────────────────────────────

export function TicketCreateModal({
  open,
  onClose,
  onSuccess,
}: TicketCreateModalProps) {
  const [values, setValues] = useState<FormValues>(EMPTY_VALUES);
  const [errors, setErrors] = useState<FormErrors>({});
  const [apiError, setApiError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // Focus the Subject field when the modal opens.
  const subjectRef = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (open) {
      // Defer one tick so the dialog's showModal() has completed.
      const id = setTimeout(() => subjectRef.current?.focus(), 0);
      return () => clearTimeout(id);
    }
  }, [open]);

  // Reset all state when the modal closes so reopening starts clean.
  function handleClose() {
    if (submitting) return;
    setValues(EMPTY_VALUES);
    setErrors({});
    setApiError(null);
    onClose();
  }

  function handleChange(
    field: keyof FormValues,
    value: string,
  ) {
    setValues((prev) => ({ ...prev, [field]: value }));
    // Clear the field error as the user corrects it.
    if (errors[field as keyof FormErrors]) {
      setErrors((prev) => {
        const next = { ...prev };
        delete next[field as keyof FormErrors];
        return next;
      });
    }
    if (apiError) setApiError(null);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    // Submission guard — prevents double-submit.
    if (submitting) return;

    const fieldErrors = validate(values);
    if (hasErrors(fieldErrors)) {
      setErrors(fieldErrors);
      return;
    }

    setErrors({});
    setApiError(null);
    setSubmitting(true);

    try {
      const ticket = await ticketsService.create({
        subject: values.subject.trim(),
        description: values.description.trim(),
        customer_name: values.customer_name.trim(),
        customer_email: values.customer_email.trim(),
        channel: values.channel,
      });

      // Reset before calling onSuccess so the modal is clean if it is
      // reopened without unmounting.
      setValues(EMPTY_VALUES);
      setErrors({});
      setApiError(null);
      onSuccess(ticket);
    } catch (err) {
      let message = "An unexpected error occurred. Please try again.";

      if (err instanceof ApiError) {
        if (err.status === 401) {
          message = "Session expired. Please sign in again.";
        } else if (err.status === 400) {
          // Backend returns the raw validation message as plain text.
          message = err.message;
        } else if (err.status >= 500) {
          message = "Failed to create ticket. Please try again.";
        } else {
          message = err.message || message;
        }
      }

      setApiError(message);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Modal open={open} onClose={handleClose} title="New Ticket" size="md">
      <form onSubmit={handleSubmit} noValidate>
        <div className="flex flex-col gap-4">
          {/* Subject */}
          <Input
            ref={subjectRef}
            id="create-ticket-subject"
            label="Subject"
            type="text"
            value={values.subject}
            onChange={(e) => handleChange("subject", e.target.value)}
            maxLength={255}
            placeholder="Brief summary of the issue"
            disabled={submitting}
            error={errors.subject}
            aria-required="true"
          />

          {/* Description */}
          <Textarea
            id="create-ticket-description"
            label="Description"
            value={values.description}
            onChange={(e) => handleChange("description", e.target.value)}
            placeholder="Describe the issue in detail…"
            rows={4}
            disabled={submitting}
            error={errors.description}
            aria-required="true"
          />

          {/* Customer Name */}
          <Input
            id="create-ticket-customer-name"
            label="Customer Name"
            type="text"
            value={values.customer_name}
            onChange={(e) => handleChange("customer_name", e.target.value)}
            placeholder="Full name"
            disabled={submitting}
            error={errors.customer_name}
            aria-required="true"
          />

          {/* Customer Email */}
          <Input
            id="create-ticket-customer-email"
            label="Customer Email"
            type="email"
            autoComplete="off"
            value={values.customer_email}
            onChange={(e) => handleChange("customer_email", e.target.value)}
            placeholder="customer@example.com"
            disabled={submitting}
            error={errors.customer_email}
            aria-required="true"
          />

          {/* Channel */}
          <div className="flex flex-col gap-1.5">
            <label
              htmlFor="create-ticket-channel"
              className="text-sm font-medium text-[var(--text-primary)]"
            >
              Channel
            </label>
            <div className="relative">
              <select
                id="create-ticket-channel"
                value={values.channel}
                onChange={(e) =>
                  handleChange("channel", e.target.value)
                }
                disabled={submitting}
                className={selectClass}
              >
                {CHANNEL_OPTIONS.map((o) => (
                  <option key={o.value} value={o.value}>
                    {o.label}
                  </option>
                ))}
              </select>
              {/* Chevron overlay — matches TicketFilters pattern */}
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

          {/* API error */}
          {apiError && (
            <p
              role="alert"
              className="rounded-[var(--radius)] bg-[var(--danger-bg)] px-3 py-2 text-sm text-[var(--danger)]"
            >
              {apiError}
            </p>
          )}

          {/* Footer */}
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
              Create Ticket
            </Button>
          </div>
        </div>
      </form>
    </Modal>
  );
}
