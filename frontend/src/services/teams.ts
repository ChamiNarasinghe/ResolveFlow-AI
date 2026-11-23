import { apiClient } from "../lib/api-client";
import type {
  AddTeamMemberRequest,
  CreateTeamRequest,
  ListResponse,
  Team,
  TeamMember,
  UpdateTeamRequest,
} from "../types";

export const teamsService = {
  list(): Promise<ListResponse<Team>> {
    return apiClient.get<ListResponse<Team>>("/api/v1/teams");
  },

  get(id: string): Promise<Team> {
    return apiClient.get<Team>(`/api/v1/teams/${id}`);
  },

  // Admin only.
  create(body: CreateTeamRequest): Promise<Team> {
    return apiClient.post<Team>("/api/v1/teams", body);
  },

  // Admin only.
  update(id: string, body: UpdateTeamRequest): Promise<Team> {
    return apiClient.patch<Team>(`/api/v1/teams/${id}`, body);
  },

  listMembers(teamId: string): Promise<ListResponse<TeamMember>> {
    return apiClient.get<ListResponse<TeamMember>>(
      `/api/v1/teams/${teamId}/members`,
    );
  },

  // Admin only.
  addMember(teamId: string, body: AddTeamMemberRequest): Promise<TeamMember> {
    return apiClient.post<TeamMember>(`/api/v1/teams/${teamId}/members`, body);
  },

  // Admin only.
  removeMember(teamId: string, userId: string): Promise<void> {
    return apiClient.delete<void>(
      `/api/v1/teams/${teamId}/members/${userId}`,
    );
  },
};
