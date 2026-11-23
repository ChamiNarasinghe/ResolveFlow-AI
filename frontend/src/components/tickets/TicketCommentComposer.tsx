"use client";

import { useState } from "react";
import { commentsService } from "../../services/comments";
import { ApiError } from "../../lib/api-client";
import { Textarea } from "../ui/Textarea";
import { Button } from "../ui/Button";

const MAX_BODY = 5000;

interface TicketCommentComposerProps {
  ticketId: string;
  onSuccess: () => void;
}

export function TicketCommentComposer({
  ticketId,
  onSuccess,
}: TicketCommentComposerProps) {
  const [body, setBody] = useState("");
  const [isInternal, setIsInternal] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const trimmed = body.trim();
  const bodyError =
    trimmed === ""
      ? undefined
      : trimmed.length > MAX_BODY
        ? `Comment cannot exceed ${MAX_BODY} characters`
        : undefined;

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    if (trimmed === "") {
      setError("Comment body is required.");
      return;
    }
    if (trimmed.length > MAX_BODY) {
      setError(`Comment cannot exceed ${MAX_BODY} characters.`);
      return;
    }

    setError(null);
    setSubmitting(true);

    try {
      await commentsService.create(ticketId, {
        body: trimmed,
        is_internal: isInternal,
      });
      setBody("");
      setIsInternal(false);
      onSuccess();
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message || "Failed to post comment.");
      } else {
        setError("Failed to post comment. Please try again.");
      }
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="rounded-[var(--radius-lg)] border border-[var(--border)] bg-[var(--bg-surface)] shadow-[var(--shadow-sm)]">
      {/* Header */}
      <div className="border-b border-[var(--border)] px-5 py-4">
        <h2 className="text-sm font-semibold text-[var(--text-primary)]">
          Add comment
        </h2>
      </div>

      <form onSubmit={handleSubmit} noValidate>
        <div className="flex flex-col gap-4 p-5">
          <Textarea
            label="Comment"
            id="comment-body"
            value={body}
            onChange={(e) => {
              setBody(e.target.value);
              if (error) setError(null);
            }}
            placeholder="Write a comment…"
            rows={4}
            maxLength={MAX_BODY}
            showCount
            disabled={submitting}
            error={bodyError}
            aria-required="true"
          />

          {/* Internal note toggle */}
          <label className="flex cursor-pointer items-start gap-3">
            <input
              type="checkbox"
              checked={isInternal}
              onChange={(e) => setIsInternal(e.target.checked)}
              disabled={submitting}
              className="mt-0.5 h-4 w-4 shrink-0 cursor-pointer rounded border-[var(--border)] accent-[var(--accent)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-ring)]"
            />
            <span className="flex flex-col gap-0.5">
              <span className="text-sm font-medium text-[var(--text-primary)]">
                Internal note
              </span>
              <span className="text-xs text-[var(--text-muted)]">
                Visible to staff only
              </span>
            </span>
          </label>

          {/* Submission error */}
          {error && (
            <p className="text-xs text-[var(--danger)]" role="alert">
              {error}
            </p>
          )}

          <div className="flex justify-end">
            <Button
              type="submit"
              variant="primary"
              size="md"
              loading={submitting}
              disabled={submitting || trimmed === ""}
            >
              {isInternal ? "Post internal note" : "Post comment"}
            </Button>
          </div>
        </div>
      </form>
    </div>
  );
}
