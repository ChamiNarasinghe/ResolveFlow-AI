# ResolveFlow AI — Development Log

Records phase completion status, audit outcomes, and known issues.

---

## Phase 4 — AI Prediction Integration

### Phase 4.3 — Frontend AI Prediction Display

**Status: APPROVED WITH MINOR ISSUES**
**Audit completed:** 2025

#### Scope

End-to-end verification that the AI prediction result produced by the Python classifier, persisted by the Go backend, and served via `GET /api/v1/tickets/{id}/ai-prediction` is correctly fetched and rendered by the Next.js frontend.

#### Verification results

| Area | Result | Method |
|---|---|---|
| Go backend AI integration | Verified | Source code review |
| Python AI service classifier | Verified | Source code review |
| Prediction persistence and retrieval | Verified | Source code review + live DB query |
| JWT-protected endpoint | Verified | Source code review + curl |
| Frontend API integration (`tickets.ts`) | Verified | Source code review |
| Frontend TypeScript types (`AIPrediction`) | Verified | Source code review — 10 fields match backend JSON exactly |
| Prediction success state (200) | Verified | Source code review + backend access logs |
| Prediction absent state (404) | Verified | Source code review + backend access logs |
| Loading and error states | Source-code verified | Source code review |
| Visual browser verification | Not performed | No valid test credentials available |
| Mobile visual verification | Not performed | No valid test credentials available |

#### Verification notes

- Backend access logs confirmed the frontend issues the correct concurrent requests on ticket detail load: `GET /tickets/{id}`, `/history`, `/comments`, `/ai-prediction`.
- 404 on `/ai-prediction` (tickets with no prediction) does not affect other sections — confirmed in logs.
- React StrictMode causes two identical requests per mount in development. The `cancelled` flag in the `useEffect` cleanup handles this correctly.
- Direct curl calls confirmed the endpoint returns 401 for missing or invalid JWT tokens.
- Database query confirmed 6 prediction records across 4 tickets at time of audit.

#### Known minor issues

| ID | Issue | Severity | Location |
|---|---|---|---|
| M-1 | Mid-session JWT expiration displays an AI panel error rather than globally redirecting to login. `AuthGuard` only handles expiry at hydration time. | Minor UX | `frontend/src/components/tickets/TicketAIPrediction.tsx` |
| M-2 | `ticket_classifier.py.bak` (archived keyword-rules classifier) remains committed to the repository. Not imported anywhere, but creates confusion. | Minor / housekeeping | `ai-service/app/services/ticket_classifier.py.bak` |
| M-3 | Model version string `"tfidf-logreg-v1"` is hardcoded in the Go backend. Future model updates require a code change. | Minor / maintainability | `backend/internal/handlers/ticket.go` |
| M-4 | Four Go repository integration tests are skipped when `DATABASE_URL` is not set in the test environment. Tests themselves are correct. | Minor / CI gap | `backend/tests/` |

#### Historical data note

The database contains prediction records with `model_version = "rules-v1"` created before the TF-IDF + Logistic Regression classifier replaced the earlier keyword-rules implementation. These records are accurate for the model that produced them and must not be modified.

#### Audit constraints respected

- No application source code was modified during the audit.
- No database records were created, modified, or deleted.
- No authentication configuration was changed.
- No model artifacts were modified.
- No dependencies were added or removed.
- No files were deleted.
- No commits were made.

---

## Stale documentation corrections

The following entries in `requirements.md` and `architecture.md` were updated in this log pass to reflect the current TF-IDF + Logistic Regression implementation. The classifier was upgraded from keyword-rules to a trained ML pipeline prior to Phase 4.3.

| Document | Stale text | Corrected in |
|---|---|---|
| `requirements.md` CON-2 | "keyword-based classification (no ML model)" | `requirements.md` |
| `requirements.md` Out of Scope | "ML-based classification (current implementation is keyword-based)" | `requirements.md` |
| `architecture.md` AI service responsibilities | "Keyword-based ticket classification" | `architecture.md` |
