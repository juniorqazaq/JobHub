import type { UserRole, UserStatus } from "../../models/auth";

export interface AuthSessionDto {
  user: {
    id: string;
    full_name: string;
    email: string;
    role: UserRole;
    status: UserStatus;
  };
  company?: {
    id: string;
    name: string;
  };
  csrf_token: string;
  expires_at: string;
}
