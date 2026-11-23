import { apiClient } from "../lib/api-client";
import type { ProvisionUserRequest, ProvisionUserResponse } from "../types";

export const adminService = {
  provisionUser(body: ProvisionUserRequest): Promise<ProvisionUserResponse> {
    return apiClient.post<ProvisionUserResponse>("/api/v1/admin/users", body);
  },
};
