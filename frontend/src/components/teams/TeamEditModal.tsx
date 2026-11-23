"use client";

import { useEffect, useRef, useState } from "react";
import { teamsService } from "../../services/teams";
import { ApiError } from "../../lib/api-client";
import { Modal } from "../ui/Modal";
import { Input } from "../ui/Input";
import { Textarea } from "../ui/Textarea";
import { Button } from "../ui/Button";
import type { Team } from "../../types";

interface TeamEditModalProps {
  open: boolean;
  team: Team;
  onClose: () => void;
  onSuccess: (team: Team) => void;
}

// NOTE: The parent must pass key={team.id} so this component remounts
// whenever a different team is selected, ensuring state seeds correctly.
export function TeamEditModal({ open, team, onClose, onSuccess }: TeamEditModalProps) {
  // State seeds from props at mount — correct because parent remounts via key.
  const [name, setName] = useState(team.name);
  const [description, setDescription] = useState(team.description ?? "");
  const [nameError, setNameError] = useState<string | undefined>();
  const [apiError, setApiError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const nameRef = useRef<HTMLInputElement>(null);

  // Focus the name field when the modal opens — no setState here.
  useEffect(() => {
    if (!open) return;
    const id = setTimeout(() => nameRef.current?.focus(), 0);
    return () => clearTimeout(id);
  }, [open]);

  function handleClose() {
    if (submitting) return;
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

    const trimmedDesc = description.trim();
    const descriptionChanged = trimmedDesc !== (team.description ?? "").trim();
    const nameChanged = trimmedName !== team.name;

    // Nothing changed — close without a request.
    if (!nameChanged && !descriptionChanged) {
      setSubmitting(false);
      onClose();
      return;
    }

    try {
      const body: {
        name?: string;
        description?: string;
        clear_description?: boolean;
      } = {};

      if (nameChanged) body.name = trimmedName;

      if (descriptionChanged) {
        if (trimmedDesc === "") {
          body.clear_description = true;
        } else {
          body.description = trimmedDesc;
        }
      }

      const updated = await teamsService.update(team.id, body);
      setApiError(null);
      onSuccess(updated);
    } catch (err) {
      let message = "An unexpected error occurred. Please try again.";
      if (err instanceof ApiError) {
        if (err.status === 403) message = "Admin access required.";
        else if (err.status === 404) message = "Team not found.";
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
    <Modal open={open} onClose={handleClose} title="Edit Team" size="sm">
      <form onSubmit={handleSubmit} noValidate>
        <div className="flex flex-col gap-4">
          <Input
            ref={nameRef}
            id="edit-team-name"
            label="Team name"
            type="text"
            value={name}
            onChange={(e) => {
              setName(e.target.value);
              if (nameError) setNameError(undefined);
              if (apiError) setApiError(null);
            }}
            maxLength={100}
            disabled={submitting}
            error={nameError}
            aria-required="true"
          />

          <Textarea
            id="edit-team-description"
            label="Description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Optional — briefly describe this team's focus."
            rows={3}
            disabled={submitting}
            hint="Clear this field to remove the description."
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
              Save Changes
            </Button>
          </div>
        </div>
      </form>
    </Modal>
  );
}
