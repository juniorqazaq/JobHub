import { useTranslation } from "react-i18next";
import { useAuth } from "../app/authContext";
import { SiteShell } from "../layouts/SiteShell";

export function EmployerPage() {
  const { t } = useTranslation();
  const { session } = useAuth();
  return (
    <SiteShell>
      <main className="shell-page page-container">
        <p className="auth-eyebrow">{t("auth.employerEyebrow")}</p>
        <h1>{t("auth.employerTitle", { name: session!.user.fullName })}</h1>
        <dl className="profile-list">
          <div><dt>{t("auth.companyName")}</dt><dd>{session!.company?.name || t("common.notProvided")}</dd></div>
          <div><dt>{t("auth.role")}</dt><dd>{t("auth.roles.employer")}</dd></div>
        </dl>
        <p>{t("auth.employerNext")}</p>
      </main>
    </SiteShell>
  );
}
