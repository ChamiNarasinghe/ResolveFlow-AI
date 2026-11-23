// ─── Domain models ────────────────────────────────────────────────────────────
// Derived directly from backend/internal/models/*.go and handler response shapes.

export type UserRole = "admin" | "manager" | "agent";

export interface User {
  id: string;
  full_name: string;
  email: string;
  role: UserRole;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export type TicketStatus =
  | "open"
  | "in_progress"
  | "waiting_for_customer"
  | "resolved"
  | "closed";

export type TicketPriority = "low" | "medium" | "high" | "urgent";

export type TicketChannel = "web" | "email" | "chat" | "phone";

export type TicketCategory =
  | "billing"
  | "technical_support"
  | "account"
  | "general_inquiry";

export interface Ticket {
  id: string;
  ticket_number: number;
  subject: string;
  description: string;
  customer_name: string;
  customer_email: string;
  status: TicketStatus;
  priority: TicketPriority;
  channel: TicketChannel;
  category: TicketCategory | null;
  assigned_team_id: string | null;
  assigned_to: string | null;
  created_by: string | null;
  created_at: string;
  updated_at: string;
  resolved_at: string | null;
}

export type AIPredictionCategory = TicketCategory;
export type AIPredictionPriority = TicketPriority;
export type AIPredictionSentiment = "positive" | "neutral" | "negative";

export interface AIPrediction {
  id: string;
  ticket_id: string;
  model_version: string;
  category: AIPredictionCategory | null;
  predicted_priority: AIPredictionPriority | null;
  sentiment: AIPredictionSentiment | null;
  confidence: number | null;
  suggested_team_id: string | null;
  response_suggestion: string | null;
  created_at: string;
}

export type CommentAuthorType = "agent" | "customer" | "system";

export interface Comment {
  id: string;
  ticket_id: string;
  author_id: string | null;
  author_type: CommentAuthorType | null;
  body: string;
  is_internal: boolean;
  created_at: string;
}

export type TeamMemberRole = "lead" | "member";

export interface Team {
  id: string;
  name: string;
  description: string | null;
  created_at: string;
}

export interface TeamMember {
  team_id: string;
  user_id: string;
  team_role: TeamMemberRole;
  joined_at: string;
  user?: User;
}

export interface TicketHistory {
  id: string;
  ticket_id: string;
  actor_id: string | null;
  action: string;
  old_value: Record<string, unknown> | null;
  new_value: Record<string, unknown> | null;
  created_at: string;
}

// ─── Auth shapes ──────────────────────────────────────────────────────────────
// Matches AuthResponse struct in backend/internal/auth/service.go

export interface AuthResponse {
  user: User;
  access_token: string;
  token_type: "Bearer";
}

export interface MeResponse {
  user: User;
}

// ─── Request bodies ───────────────────────────────────────────────────────────

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  full_name: string;
  email: string;
  password: string;
}

export interface CreateTicketRequest {
  subject: string;
  description: string;
  customer_name: string;
  customer_email: string;
  channel?: TicketChannel;
}

export interface UpdateTicketRequest {
  status?: TicketStatus;
  priority?: TicketPriority;
  category?: TicketCategory | null;
  assigned_team_id?: string | null;
  assigned_to?: string | null;
}

export interface AssignTicketRequest {
  team_id: string | null;
}

export interface CreateCommentRequest {
  body: string;
  is_internal?: boolean;
}

export interface CreateTeamRequest {
  name: string;
  description?: string;
}

export interface UpdateTeamRequest {
  name?: string;
  description?: string;
  clear_description?: boolean;
}

export interface AddTeamMemberRequest {
  user_id: string;
  team_role?: TeamMemberRole;
}

// ─── Paginated response wrapper ───────────────────────────────────────────────
// Matches the pagination shape returned by ListTickets and GetTicketHistory.

export interface Pagination {
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: Pagination;
}

// ─── List response wrappers ───────────────────────────────────────────────────
// Matches { "data": [...] } shape returned by ListTeams and ListTeamMembers.

export interface ListResponse<T> {
  data: T[];
}

// ─── Ticket list query params ─────────────────────────────────────────────────

export interface TicketListParams {
  page?: number;
  page_size?: number;
  status?: TicketStatus;
  priority?: TicketPriority;
}
