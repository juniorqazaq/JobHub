import type { AuthSession, LoginRequest, RegisterRequest } from "../models/auth";

export interface AuthRepository {
  register(input: RegisterRequest): Promise<AuthSession>;
  login(input: LoginRequest): Promise<AuthSession>;
  me(): Promise<AuthSession>;
  logout(csrfToken: string): Promise<void>;
}
