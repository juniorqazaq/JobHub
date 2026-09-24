import { useTranslation } from "react-i18next";
import { useAuth } from "../app/authContext";
import { SiteShell } from "../layouts/SiteShell";

export function AccountPage() {
  const { t } = useTranslation();
  const { session } = useAuth();
  const user = session!.user;
  return (
    <SiteShell>
      <main className="shell-page page-container">
        <p className="auth-eyebrow">{t("auth.accountEyebrow")}</p>
        <h1>{t("auth.accountTitle", { name: user.fullName })}</h1>
        <dl className="profile-list">
          <div><dt>{t("auth.fullName")}</dt><dd>{user.fullName}</dd></div>
          <div><dt>{t("auth.email")}</dt><dd>{user.email}</dd></div>
          <div><dt>{t("auth.role")}</dt><dd>{t(`auth.roles.${user.role}`)}</dd></div>
        </dl>
        <p>{t("auth.profileNext")}</p>
      </main>
    </SiteShell>
  );
}
