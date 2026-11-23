"use client";

import { useEffect, useReducer } from "react";
import { ticketsService } from "../../services/tickets";
import { ApiError } from "../../lib/api-client";
import { Card, CardHeader, CardBody } from "../ui/Card";
import { Skeleton } from "../ui/Skeleton";
import { EmptyState } from "../ui/EmptyState";
import { Badge } from "../ui/Badge";
import type { AIPrediction } from "../../types";

// ─── State ────────────────────────────────────────────────────────────────────

interface AIState {
  prediction: AIPrediction | null;
  loading: boolean;
  error: string | null;
  absent: boolean; // true when the backend returned 404
}

type AIAction =
  | { type: "LOADING" }
  | { type: "SUCCESS"; prediction: AIPrediction }
  | { type: "ABSENT" }
  | { type: "ERROR"; message: string };

function aiReducer(state: AIState, action: AIAction): AIState {
  switch (action.type) {
    case "LOADING":  return { prediction: null, loading: true,  error: null,          absent: false };
    case "SUCCESS":  return { prediction: action.prediction, loading: false, error: null, absent: false };
    case "ABSENT":   return { prediction: null, loading: false, error: null,          absent: true  };
    case "ERROR":    return { prediction: null, loading: false, error: action.message, absent: false };
  }
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

const SENTIMENT_VARIANT = {
  positive: "success",
  neutral:  "neutral",
  negative: "danger",
} as const;

const CATEGORY_LABEL: Record<string, string> = {
  billing:           "Billing",
  technical_support: "Technical Support",
  account:           "Account",
  general_inquiry:   "General Inquiry",
};

function formatConfidence(value: number): string {
  return `${(value * 100).toFixed(1)}%`;
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString("en-US", {
    month: "short", day: "numeric", year: "numeric",
    hour: "numeric", minute: "2-digit",
  });
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-3 py-2 border-b border-[var(--border)] last:border-0">
      <span className="text-xs text-[var(--text-muted)] shrink-0">{label}</span>
      <span className="text-xs font-medium text-[var(--text-primary)] text-right">{children}</span>
    </div>
  );
}

// ─── Component ────────────────────────────────────────────────────────────────

export function TicketAIPrediction({ ticketId }: { ticketId: string }) {
  const [state, dispatch] = useReducer(aiReducer, {
    prediction: null, loading: true, error: null, absent: false,
  });

  useEffect(() => {
    let cancelled = false;

    dispatch({ type: "LOADING" });

    ticketsService
      .getAIPrediction(ticketId)
      .then((p) => {
        if (!cancelled) dispatch({ type: "SUCCESS", prediction: p });
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        if (err instanceof ApiError && err.status === 404) {
          dispatch({ type: "ABSENT" });
        } else {
          dispatch({
            type: "ERROR",
            message: err instanceof Error ? err.message : "Failed to load AI prediction.",
          });
        }
      });

    return () => { cancelled = true; };
  }, [ticketId]);

  return (
    <Card padding={false}>
      <CardHeader>
        <div className="flex items-center gap-2">
          <svg className="h-4 w-4 text-[var(--accent)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.75} aria-hidden="true">
            <path strokeLinecap="round" strokeLinejoin="round" d="M9.75 3.104v5.714a2.25 2.25 0 01-.659 1.591L5 14.5M9.75 3.104c-.251.023-.501.05-.75.082m.75-.082a24.301 24.301 0 014.5 0m0 0v5.714c0 .597.237 1.17.659 1.591L19.8 15.3M14.25 3.104c.251.023.501.05.75.082M19.8 15.3l-1.57.393A9.065 9.065 0 0112 15a9.065 9.065 0 00-6.23-.693L5 14.5m14.8.8l1.402 1.402c1.232 1.232.65 3.318-1.067 3.611A48.309 48.309 0 0112 21c-2.773 0-5.491-.235-8.135-.687-1.718-.293-2.3-2.379-1.067-3.61L5 14.5" />
          </svg>
          <h2 className="text-sm font-semibold text-[var(--text-primary)]">AI Prediction</h2>
        </div>
      </CardHeader>

      <CardBody>
        {state.loading ? (
          <div className="flex flex-col gap-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="flex justify-between">
                <Skeleton className="h-3 w-20" />
                <Skeleton className="h-3 w-24" />
              </div>
            ))}
          </div>
        ) : state.absent ? (
          <EmptyState
            title="No prediction yet"
            description="AI classification runs automatically when a ticket is created. It may be unavailable if the AI service was unreachable."
            className="border-0 bg-transparent py-6 shadow-none"
          />
        ) : state.error ? (
          <p className="text-xs text-[var(--danger)]">{state.error}</p>
        ) : state.prediction ? (
          <div>
            {state.prediction.category && (
              <Row label="Category">
                {CATEGORY_LABEL[state.prediction.category] ?? state.prediction.category}
              </Row>
            )}
            {state.prediction.predicted_priority && (
              <Row label="Predicted priority">
                <span className="capitalize">{state.prediction.predicted_priority}</span>
              </Row>
            )}
            {state.prediction.sentiment && (
              <Row label="Sentiment">
                <Badge variant={SENTIMENT_VARIANT[state.prediction.sentiment]}>
                  <span className="capitalize">{state.prediction.sentiment}</span>
                </Badge>
              </Row>
            )}
            {state.prediction.confidence != null && (
              <Row label="Model confidence">
                <span title="Raw model score — not a calibrated probability">
                  {formatConfidence(state.prediction.confidence)}
                </span>
              </Row>
            )}
            <Row label="Model">{state.prediction.model_version}</Row>
            <Row label="Classified">{formatDate(state.prediction.created_at)}</Row>
            <p className="mt-3 text-[10px] leading-relaxed text-[var(--text-muted)]">
              Model confidence is a raw classifier score, not a calibrated probability.
            </p>
          </div>
        ) : null}
      </CardBody>
    </Card>
  );
}
