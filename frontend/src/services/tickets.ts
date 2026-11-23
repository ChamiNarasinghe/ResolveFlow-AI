import { apiClient } from "../lib/api-client";
import type {
  AIPrediction,
  AssignTicketRequest,
  CreateTicketRequest,
  PaginatedResponse,
  Ticket,
  TicketHistory,
  TicketListParams,
  UpdateTicketRequest,
} from "../types";

export const ticketsService = {
  list(params: TicketListParams = {}): Promise<PaginatedResponse<Ticket>> {
    const query = new URLSearchParams();
    if (params.page !== undefined) query.set("page", String(params.page));
    if (params.page_size !== undefined)
      query.set("page_size", String(params.page_size));
    if (params.status) query.set("status", params.status);
    if (params.priority) query.set("priority", params.priority);

    const qs = query.toString();
    return apiClient.get<PaginatedResponse<Ticket>>(
      `/api/v1/tickets${qs ? `?${qs}` : ""}`,
    );
  },

  get(id: string): Promise<Ticket> {
    return apiClient.get<Ticket>(`/api/v1/tickets/${id}`);
  },

  create(body: CreateTicketRequest): Promise<Ticket> {
    return apiClient.post<Ticket>("/api/v1/tickets", body);
  },

  update(id: string, body: UpdateTicketRequest): Promise<Ticket> {
    return apiClient.patch<Ticket>(`/api/v1/tickets/${id}`, body);
  },

  // Admin only.
  delete(id: string): Promise<void> {
    return apiClient.delete<void>(`/api/v1/tickets/${id}`);
  },

  // Admin only.
  assign(id: string, body: AssignTicketRequest): Promise<Ticket> {
    return apiClient.patch<Ticket>(`/api/v1/tickets/${id}/assignment`, body);
  },

  getHistory(
    id: string,
    params: { page?: number; page_size?: number } = {},
  ): Promise<PaginatedResponse<TicketHistory>> {
    const query = new URLSearchParams();
    if (params.page !== undefined) query.set("page", String(params.page));
    if (params.page_size !== undefined)
      query.set("page_size", String(params.page_size));

    const qs = query.toString();
    return apiClient.get<PaginatedResponse<TicketHistory>>(
      `/api/v1/tickets/${id}/history${qs ? `?${qs}` : ""}`,
    );
  },

  getAIPrediction(id: string): Promise<AIPrediction> {
    return apiClient.get<AIPrediction>(`/api/v1/tickets/${id}/ai-prediction`);
  },
};
