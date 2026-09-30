import type { PropsWithChildren } from "react";
import { useTranslation } from "react-i18next";
import { NavLink } from "react-router-dom";
import { CandidateWorkspaceLayout } from "./CandidateWorkspaceLayout";

interface CandidateJobsLayoutProps extends PropsWithChildren {
  tabTitle: string;
  tabDescription: string;
}

export function CandidateJobsLayout({ children, tabTitle, tabDescription }: CandidateJobsLayoutProps) {
  const { t } = useTranslation();
  return (
    <CandidateWorkspaceLayout>
      <section className="candidate-list-page" aria-labelledby="candidate-jobs-title">
        <header>
          <p className="auth-eyebrow">{t("workspace.title")}</p>
          <h1 id="candidate-jobs-title">{t("workspace.nav.myJobs")}</h1>
          <p>{t("workspace.myJobsDescription")}</p>
        </header>
        <nav className="candidate-tabs" aria-label={t("workspace.myJobsTabsLabel")}>
          <NavLink to="/saved" end>{t("workspace.tabs.saved")}</NavLink>
	          <NavLink to="/saved-companies" end>{t("workspace.tabs.companies")}</NavLink>
          <NavLink to="/applications" end>{t("workspace.tabs.applications")}</NavLink>
        </nav>
        <header className="candidate-tab-header">
          <h2>{tabTitle}</h2>
          <p>{tabDescription}</p>
        </header>
        {children}
      </section>
    </CandidateWorkspaceLayout>
  );
}
