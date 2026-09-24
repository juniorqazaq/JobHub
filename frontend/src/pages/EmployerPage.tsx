import { ArrowRight, FilePlus2, LayoutList } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { useAuth } from "../app/authContext";
import { EmployerWorkspaceNav } from "../components/employer/EmployerWorkspaceNav";
import { SiteShell } from "../layouts/SiteShell";

export function EmployerPage() {
  const { t } = useTranslation();
  const { session } = useAuth();
  return (
    <SiteShell>
      <main className="employer-page page-container">
        <EmployerWorkspaceNav />
        <header className="employer-page__header">
          <p className="auth-eyebrow">{t("auth.employerEyebrow")}</p>
          <h1>{t("employer.dashboardTitle", { company: session!.company?.name || t("common.notProvided") })}</h1>
          <p>{t("employer.dashboardDescription")}</p>
        </header>
        <div className="employer-actions">
          <Link to="/employer/vacancies" className="employer-action"><LayoutList size={22} aria-hidden="true" /><span><strong>{t("employer.manageVacancies")}</strong><small>{t("employer.manageVacanciesDescription")}</small></span><ArrowRight size={18} aria-hidden="true" /></Link>
          <Link to="/employer/vacancies/new" className="employer-action"><FilePlus2 size={22} aria-hidden="true" /><span><strong>{t("employer.createVacancy")}</strong><small>{t("employer.createVacancyDescription")}</small></span><ArrowRight size={18} aria-hidden="true" /></Link>
        </div>
      </main>
    </SiteShell>
  );
}
