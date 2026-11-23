import { apiClient } from "../lib/api-client";
import type {
  AuthResponse,
  LoginRequest,
  MeResponse,
  RegisterRequest,
} from "../types";

export const authService = {
  login(body: LoginRequest): Promise<AuthResponse> {
    return apiClient.post<AuthResponse>("/api/v1/auth/login", body);
  },

  // Only available when APP_ENV=development on the backend.
  register(body: RegisterRequest): Promise<AuthResponse> {
    return apiClient.post<AuthResponse>("/api/v1/auth/register", body);
  },

  me(): Promise<MeResponse> {
    return apiClient.get<MeResponse>("/api/v1/auth/me");
  },
};
