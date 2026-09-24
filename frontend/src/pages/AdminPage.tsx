import { useTranslation } from "react-i18next";
import { useAuth } from "../app/authContext";
import { SiteShell } from "../layouts/SiteShell";

export function AdminPage() {
  const { t } = useTranslation();
  const { session } = useAuth();
  return (
    <SiteShell>
      <main className="shell-page page-container">
        <p className="auth-eyebrow">{t("auth.adminEyebrow")}</p>
        <h1>{t("auth.adminTitle", { name: session!.user.fullName })}</h1>
        <p>{t("auth.adminNext")}</p>
      </main>
    </SiteShell>
  );
}
