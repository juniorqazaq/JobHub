import { apiClient } from "../client";
import type { AuthSession, LoginRequest, RegisterRequest } from "../models/auth";
import type { AuthRepository } from "../repositories/AuthRepository";
import { mapAuthSession } from "./authMappers";
import type { AuthSessionDto } from "./dto/authDto";

export class HttpAuthRepository implements AuthRepository {
  async register(input: RegisterRequest): Promise<AuthSession> {
    const { data } = await apiClient.post<AuthSessionDto>("/auth/register", {
      full_name: input.fullName,
      email: input.email,
      password: input.password,
      role: input.role,
      company_name: input.companyName,
    });
    return mapAuthSession(data);
  }

  async login(input: LoginRequest): Promise<AuthSession> {
    const { data } = await apiClient.post<AuthSessionDto>("/auth/login", input);
    return mapAuthSession(data);
  }

  async me(): Promise<AuthSession> {
    const { data } = await apiClient.get<AuthSessionDto>("/auth/me");
    return mapAuthSession(data);
  }

  async logout(csrfToken: string): Promise<void> {
    await apiClient.post("/auth/logout", undefined, { headers: { "X-CSRF-Token": csrfToken } });
  }
}
