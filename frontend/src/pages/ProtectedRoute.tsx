import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { Link, Navigate, useLocation } from "react-router-dom";
import type { UserRole } from "../api/models/auth";
import { useAuth } from "../app/authContext";
import { SiteShell } from "../layouts/SiteShell";

interface ProtectedRouteProps {
  role?: UserRole;
  children: ReactNode;
}

export function ProtectedRoute({ role, children }: ProtectedRouteProps) {
  const { t } = useTranslation();
  const auth = useAuth();
  const location = useLocation();
  if (auth.isLoading) return <div className="route-loading" aria-hidden="true" />;
  if (!auth.session) return <Navigate to={`/login?returnTo=${encodeURIComponent(location.pathname + location.search)}`} replace />;
  if (role && auth.session.user.role !== role) {
    return (
      <SiteShell>
        <main className="auth-page page-container">
          <section className="auth-panel">
            <p className="auth-eyebrow">{t("auth.accessDeniedEyebrow")}</p>
            <h1>{t("auth.accessDeniedTitle")}</h1>
            <p>{t("auth.accessDeniedDescription")}</p>
            <Link className="text-link" to="/jobs">{t("nav.jobs")}</Link>
          </section>
        </main>
      </SiteShell>
    );
  }
  return children;
}
