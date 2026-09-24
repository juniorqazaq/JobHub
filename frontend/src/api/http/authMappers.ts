import type { AuthSession } from "../models/auth";
import type { AuthSessionDto } from "./dto/authDto";

export function mapAuthSession(dto: AuthSessionDto): AuthSession {
  return {
    user: {
      id: dto.user.id,
      fullName: dto.user.full_name,
      email: dto.user.email,
      role: dto.user.role,
      status: dto.user.status,
    },
    company: dto.company,
    csrfToken: dto.csrf_token,
    expiresAt: dto.expires_at,
  };
}
