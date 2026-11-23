"use client";

import { useCallback, useEffect, useReducer } from "react";
import { commentsService } from "../../services/comments";
import { Card, CardHeader, CardBody } from "../ui/Card";
import { Skeleton } from "../ui/Skeleton";
import { EmptyState } from "../ui/EmptyState";
import { Avatar } from "../ui/Avatar";
import { Badge } from "../ui/Badge";
import type { Comment } from "../../types";

// ─── State ────────────────────────────────────────────────────────────────────

interface ThreadState {
  comments: Comment[];
  loading: boolean;
  error: string | null;
}

type ThreadAction =
  | { type: "LOADING" }
  | { type: "SUCCESS"; comments: Comment[] }
  | { type: "ERROR"; message: string };

function threadReducer(state: ThreadState, action: ThreadAction): ThreadState {
  switch (action.type) {
    case "LOADING":
      return { ...state, loading: true, error: null };
    case "SUCCESS":
      return { comments: action.comments, loading: false, error: null };
    case "ERROR":
      return { ...state, loading: false, error: action.message };
  }
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

function authorTypeLabel(type: string | null): string {
  switch (type) {
    case "agent":
      return "Agent";
    case "customer":
      return "Customer";
    case "system":
      return "System";
    default:
      return "Unknown";
  }
}

// Returns a display name for the avatar and header.
// Uses the current user's full_name only when the IDs match.
function resolveAuthorName(
  comment: Comment,
  currentUserId: string | null | undefined,
  currentUserName: string | null | undefined,
): string {
  if (
    comment.author_id &&
    currentUserId &&
    comment.author_id === currentUserId &&
    currentUserName
  ) {
    return currentUserName;
  }
  const label = authorTypeLabel(comment.author_type);
  if (comment.author_id) {
    return `${label} ${comment.author_id.slice(0, 8)}…`;
  }
  return label;
}

// ─── Comment bubble ───────────────────────────────────────────────────────────

interface CommentBubbleProps {
  comment: Comment;
  currentUserId: string | null | undefined;
  currentUserName: string | null | undefined;
}

function CommentBubble({
  comment,
  currentUserId,
  currentUserName,
}: CommentBubbleProps) {
  const authorName = resolveAuthorName(comment, currentUserId, currentUserName);
  const isInternal = comment.is_internal;

  return (
    <li
      className={[
        "rounded-[var(--radius)] border p-4",
        isInternal
          ? "border-amber-200 bg-amber-50/60"
          : "border-[var(--border)] bg-[var(--bg-surface)]",
      ].join(" ")}
      aria-label={isInternal ? "Internal note" : undefined}
    >
      {/* Header row */}
      <div className="mb-2.5 flex flex-wrap items-center gap-2">
        <Avatar name={authorName} size="xs" />
        <span className="text-xs font-semibold text-[var(--text-primary)]">
          {authorName}
        </span>
        <span className="text-[10px] text-[var(--text-muted)]">
          · {formatDateTime(comment.created_at)}
        </span>
        {isInternal && (
          <Badge variant="warning" className="ml-auto shrink-0">
            Internal note
          </Badge>
        )}
      </div>

      {/* Body */}
      <p className="whitespace-pre-wrap text-sm leading-relaxed text-[var(--text-secondary)]">
        {comment.body}
      </p>
    </li>
  );
}

// ─── Loading skeleton ─────────────────────────────────────────────────────────

function CommentSkeleton() {
  return (
    <li className="rounded-[var(--radius)] border border-[var(--border)] bg-[var(--bg-surface)] p-4">
      <div className="mb-3 flex items-center gap-2">
        <Skeleton className="h-6 w-6 rounded-full" />
        <Skeleton className="h-3 w-32" />
        <Skeleton className="h-3 w-24" />
      </div>
      <Skeleton className="mb-1.5 h-3 w-full" />
      <Skeleton className="h-3 w-3/4" />
    </li>
  );
}

// ─── Component ────────────────────────────────────────────────────────────────

interface TicketCommentThreadProps {
  ticketId: string;
  currentUserId: string | null | undefined;
  currentUserName: string | null | undefined;
  // Increment this to trigger a re-fetch from the parent.
  refreshKey?: number;
}

export function TicketCommentThread({
  ticketId,
  currentUserId,
  currentUserName,
  refreshKey = 0,
}: TicketCommentThreadProps) {
  const [state, dispatch] = useReducer(threadReducer, {
    comments: [],
    loading: true,
    error: null,
  });

  const load = useCallback(() => {
    let cancelled = false;

    dispatch({ type: "LOADING" });

    commentsService
      .list(ticketId)
      .then((comments) => {
        if (!cancelled) dispatch({ type: "SUCCESS", comments });
      })
      .catch((err: unknown) => {
        if (!cancelled)
          dispatch({
            type: "ERROR",
            message:
              err instanceof Error ? err.message : "Failed to load comments.",
          });
      });

    return () => {
      cancelled = true;
    };
  }, [ticketId]);

  // Re-fetch whenever ticketId or refreshKey changes.
  useEffect(() => {
    const cancel = load();
    return cancel;
  }, [load, refreshKey]);

  const { comments, loading, error } = state;

  return (
    <Card padding={false}>
      <CardHeader>
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold text-[var(--text-primary)]">
            Comments
          </h2>
          {!loading && !error && comments.length > 0 && (
            <span className="text-xs text-[var(--text-muted)]">
              {comments.length} comment{comments.length !== 1 ? "s" : ""}
            </span>
          )}
        </div>
      </CardHeader>

      <CardBody>
        {loading ? (
          <ul className="flex flex-col gap-3" aria-label="Loading comments">
            {Array.from({ length: 2 }).map((_, i) => (
              <CommentSkeleton key={i} />
            ))}
          </ul>
        ) : error ? (
          <div className="flex flex-col items-center gap-2 py-4 text-center">
            <p className="text-xs text-[var(--danger)]">{error}</p>
            <button
              onClick={load}
              className="text-xs font-medium text-[var(--accent)] hover:underline focus-visible:outline-none"
            >
              Try again
            </button>
          </div>
        ) : comments.length === 0 ? (
          <EmptyState
            title="No comments yet"
            description="Add the first comment below."
            className="border-0 bg-transparent py-6 shadow-none"
          />
        ) : (
          <ul className="flex flex-col gap-3">
            {comments.map((comment) => (
              <CommentBubble
                key={comment.id}
                comment={comment}
                currentUserId={currentUserId}
                currentUserName={currentUserName}
              />
            ))}
          </ul>
        )}
      </CardBody>
    </Card>
  );
}
