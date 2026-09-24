import { createContext, useContext } from "react";
import type { AuthSession, LoginRequest, RegisterRequest, UserRole } from "../api/models/auth";

interface AuthContextValue {
  session?: AuthSession;
  isLoading: boolean;
  isAuthenticated: boolean;
  register(input: RegisterRequest): Promise<AuthSession>;
  login(input: LoginRequest): Promise<AuthSession>;
  logout(): Promise<void>;
  hasRole(role: UserRole): boolean;
}

export const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) throw new Error("useAuth must be used inside AuthProvider");
  return context;
}
