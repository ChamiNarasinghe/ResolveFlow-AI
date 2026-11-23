"use client";

import { useEffect, useRef, useState } from "react";
import { teamsService } from "../../services/teams";
import { ApiError } from "../../lib/api-client";
import { Modal } from "../ui/Modal";
import { Input } from "../ui/Input";
import { Textarea } from "../ui/Textarea";
import { Button } from "../ui/Button";
import type { Team } from "../../types";

interface TeamCreateModalProps {
  open: boolean;
  onClose: () => void;
  onSuccess: (team: Team) => void;
}

export function TeamCreateModal({ open, onClose, onSuccess }: TeamCreateModalProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [nameError, setNameError] = useState<string | undefined>();
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
    setDescription("");
    setNameError(undefined);
    setApiError(null);
    onClose();
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (submitting) return;

    const trimmedName = name.trim();
    if (!trimmedName) {
      setNameError("Team name is required.");
      return;
    }
    if (trimmedName.length > 100) {
      setNameError("Team name must be 100 characters or fewer.");
      return;
    }

    setNameError(undefined);
    setApiError(null);
    setSubmitting(true);

    try {
      const trimmedDesc = description.trim();
      const team = await teamsService.create({
        name: trimmedName,
        ...(trimmedDesc ? { description: trimmedDesc } : {}),
      });
      setName("");
      setDescription("");
      setApiError(null);
      onSuccess(team);
    } catch (err) {
      let message = "An unexpected error occurred. Please try again.";
      if (err instanceof ApiError) {
        if (err.status === 403) message = "Admin access required.";
        else if (err.status === 409) message = "A team with this name already exists.";
        else if (err.status === 400) message = err.message;
        else message = err.message || message;
      }
      setApiError(message);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Modal open={open} onClose={handleClose} title="Create Team" size="sm">
      <form onSubmit={handleSubmit} noValidate>
        <div className="flex flex-col gap-4">
          <Input
            ref={nameRef}
            id="create-team-name"
            label="Team name"
            type="text"
            value={name}
            onChange={(e) => {
              setName(e.target.value);
              if (nameError) setNameError(undefined);
              if (apiError) setApiError(null);
            }}
            maxLength={100}
            placeholder="e.g. Billing Support"
            disabled={submitting}
            error={nameError}
            aria-required="true"
          />

          <Textarea
            id="create-team-description"
            label="Description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Optional — briefly describe this team's focus."
            rows={3}
            disabled={submitting}
          />

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
              Create Team
            </Button>
          </div>
        </div>
      </form>
    </Modal>
  );
}
