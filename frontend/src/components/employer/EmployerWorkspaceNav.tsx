import { Building2, FilePlus2, LayoutList, UserRound } from "lucide-react";
import { useTranslation } from "react-i18next";
import { NavLink } from "react-router-dom";

export function EmployerWorkspaceNav() {
  const { t } = useTranslation();
  return (
    <nav className="employer-nav" aria-label={t("employer.navLabel")}>
      <NavLink to="/employer/vacancies"><LayoutList size={17} aria-hidden="true" />{t("employer.vacancies")}</NavLink>
      <NavLink to="/employer/vacancies/new"><FilePlus2 size={17} aria-hidden="true" />{t("employer.createVacancy")}</NavLink>
      <span aria-disabled="true"><Building2 size={17} aria-hidden="true" />{t("employer.company")}</span>
      <NavLink to="/account"><UserRound size={17} aria-hidden="true" />{t("employer.account")}</NavLink>
    </nav>
  );
}
