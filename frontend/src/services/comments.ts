import { apiClient } from "../lib/api-client";
import type { Comment, CreateCommentRequest } from "../types";

export const commentsService = {
  // Returns a plain array — the backend does not paginate comments.
  list(ticketId: string): Promise<Comment[]> {
    return apiClient.get<Comment[]>(`/api/v1/tickets/${ticketId}/comments`);
  },

  create(ticketId: string, body: CreateCommentRequest): Promise<Comment> {
    return apiClient.post<Comment>(
      `/api/v1/tickets/${ticketId}/comments`,
      body,
    );
  },
};
