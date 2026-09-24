import { ArrowRight, Building2, UserRoundPen } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link, Navigate } from "react-router-dom";
import { useAuth } from "../app/authContext";
import { SiteShell } from "../layouts/SiteShell";
import type { AuthUser } from "../api/models/auth";

export function AccountPage() {
  const { t } = useTranslation();
  const { session } = useAuth();
  const user = session!.user;

  if (user.role === "job_seeker") {
    return <Navigate to="/profile#account" replace />;
  }

  const initials = user.fullName
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toLocaleUpperCase())
    .join("");
  const content = (
    <section className="account-page page-container">
      <section className="account-overview" aria-labelledby="account-title">
        <header className="account-header">
          <div className="account-avatar" aria-hidden="true">
            {initials}
          </div>
          <div>
            <p className="auth-eyebrow">{t("auth.accountEyebrow")}</p>
            <h1 id="account-title">{user.fullName}</h1>
            <p className="account-header__meta">
              {t(`auth.roles.${user.role}`)}
              <span aria-hidden="true">·</span>
              <span className={`account-status account-status--${user.status}`}>
                {t(`auth.statuses.${user.status}`)}
              </span>
            </p>
          </div>
        </header>
        <div className="account-content">
          <section aria-labelledby="account-summary-title">
            <h2 id="account-summary-title">{t("auth.accountSummary")}</h2>
            <AccountDetails user={user} />
          </section>
          <aside
            className="account-next-step"
            aria-labelledby="account-next-title"
          >
            {user.role === "employer" ? (
              <Building2 size={24} aria-hidden="true" />
            ) : (
              <UserRoundPen size={24} aria-hidden="true" />
            )}
            <h2 id="account-next-title">
              {user.role === "employer"
                ? t("auth.employerAccountActionTitle")
                : t("auth.profileActionTitle")}
            </h2>
            <p id="account-next-description">
              {user.role === "employer"
                ? t("auth.employerAccountActionDescription")
                : t("profile.accountDescription")}
            </p>
            {user.role === "employer" ? (
              <Link
                className="ui-button ui-button--primary ui-button--md"
                to="/employer"
              >
                <span>{t("auth.goToEmployer")}</span>
                <ArrowRight size={17} aria-hidden="true" />
              </Link>
            ) : (
              <Link
                className="ui-button ui-button--secondary ui-button--md"
                to="/admin"
              >
                <span>{t("nav.admin")}</span>
                <ArrowRight size={17} aria-hidden="true" />
              </Link>
            )}
          </aside>
        </div>
      </section>
    </section>
  );
  return <SiteShell>{content}</SiteShell>;
}

export function AccountDetails({ user }: { user: AuthUser }) {
  const { t } = useTranslation();
  return (
    <dl className="profile-list">
      <div>
        <dt>{t("auth.fullName")}</dt>
        <dd>{user.fullName}</dd>
      </div>
      <div>
        <dt>{t("auth.email")}</dt>
        <dd>{user.email}</dd>
      </div>
      <div>
        <dt>{t("auth.role")}</dt>
        <dd>{t(`auth.roles.${user.role}`)}</dd>
      </div>
      <div>
        <dt>{t("auth.accountStatus")}</dt>
        <dd>
          <span className={`account-status account-status--${user.status}`}>
            {t(`auth.statuses.${user.status}`)}
          </span>
        </dd>
      </div>
    </dl>
  );
}
