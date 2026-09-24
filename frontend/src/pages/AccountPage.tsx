import { ArrowRight, Building2, LogOut, UserRoundPen } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, useNavigate } from "react-router-dom";
import { useAuth } from "../app/authContext";
import { SiteShell } from "../layouts/SiteShell";
import { CandidateWorkspaceLayout } from "../components/candidate/CandidateWorkspaceLayout";
import { Button } from "../components/ui/Button";
import type { AuthUser } from "../api/models/auth";

export function AccountPage() {
  const { t } = useTranslation();
  const auth = useAuth();
  const { session } = auth;
  const navigate = useNavigate();
  const user = session!.user;
  const [logoutPending, setLogoutPending] = useState(false);
  const [logoutError, setLogoutError] = useState("");
  const logout = async () => {
    if (logoutPending) return;
    setLogoutError("");
    setLogoutPending(true);
    try {
      await auth.logout();
      navigate("/");
    } catch {
      setLogoutError(t("auth.logoutError"));
      setLogoutPending(false);
    }
  };

  if (user.role === "job_seeker") {
    return (
      <CandidateWorkspaceLayout>
        <section
          className="account-page account-page--candidate"
          aria-labelledby="account-title"
        >
          <header className="account-page__header">
            <h1 id="account-title">{t("auth.accountEyebrow")}</h1>
          </header>
          <section
            className="account-summary"
            aria-labelledby="account-summary-title"
          >
            <h2 id="account-summary-title">{t("auth.accountSummary")}</h2>
            <AccountDetails user={user} />
            <div className="account-actions">
              <Button
                variant="secondary"
                leadingIcon={<LogOut size={17} />}
                isLoading={logoutPending}
                onClick={() => void logout()}
              >
                {t("auth.logout")}
              </Button>
              {logoutError ? (
                <p className="auth-error account-logout-error" role="alert">
                  {logoutError}
                </p>
              ) : null}
            </div>
          </section>
        </section>
      </CandidateWorkspaceLayout>
    );
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

function AccountDetails({ user }: { user: AuthUser }) {
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
