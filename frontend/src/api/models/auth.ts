export type UserRole = "job_seeker" | "employer" | "admin";
export type UserStatus = "active" | "suspended";

export interface AuthUser {
  id: string;
  fullName: string;
  email: string;
  role: UserRole;
  status: UserStatus;
}

export interface AuthCompany {
  id: string;
  name: string;
}

export interface AuthSession {
  user: AuthUser;
  company?: AuthCompany;
  csrfToken: string;
  expiresAt: string;
}

export interface RegisterRequest {
  fullName: string;
  email: string;
  password: string;
  role: "job_seeker" | "employer";
  companyName?: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}
