import { useQuery } from "@tanstack/react-query";
import { ArrowRight, BriefcaseBusiness, MapPin, Search, Star } from "lucide-react";
import { useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { Link, useSearchParams } from "react-router-dom";
import { repositories } from "../api/repositories";
import { CompanyIdentity } from "../components/companies/CompanyIdentity";
import { EmptyState, ErrorState, JobListSkeleton } from "../components/ui/Feedback";
import { Pagination } from "../components/ui/Navigation";
import { SiteShell } from "../layouts/SiteShell";

export function CompaniesPage() {
  const { t } = useTranslation();
  const [params, setParams] = useSearchParams();
  const query = params.get("q") ?? "";
  const page = Math.max(1, Number(params.get("page")) || 1);
  const [draft, setDraft] = useState(query);
  const companies = useQuery({ queryKey: ["companies", query, page], queryFn: () => repositories.companies.search(query, page) });
  const submit = (event: FormEvent) => {
    event.preventDefault();
    const next = new URLSearchParams();
    if (draft.trim()) next.set("q", draft.trim());
    setParams(next);
  };
  return (
    <SiteShell>
      <div className="page-container companies-page">
        <header className="companies-page__header"><p className="auth-eyebrow">{t("companies.eyebrow")}</p><h1>{t("companies.title")}</h1><p>{t("companies.description")}</p></header>
        <form className="company-search" onSubmit={submit} role="search">
          <label htmlFor="company-query">{t("companies.searchLabel")}</label>
          <div><Search size={19} aria-hidden="true" /><input id="company-query" value={draft} onChange={(event) => setDraft(event.target.value)} maxLength={120} placeholder={t("companies.searchPlaceholder")} /><button type="submit">{t("companies.search")}</button></div>
        </form>
        {companies.isPending ? <JobListSkeleton label={t("companies.loading")} /> : null}
        {companies.isError ? <ErrorState title={t("companies.errorTitle")} description={t("companies.errorDescription")} actionLabel={t("common.retry")} onAction={() => void companies.refetch()} /> : null}
        {companies.data?.items.length === 0 ? <EmptyState title={t("companies.emptyTitle")} description={t("companies.emptyDescription")} /> : null}
        {companies.data?.items.length ? <div className="company-grid">{companies.data.items.map((company) => (
          <article className="company-card" key={company.id}>
            <CompanyIdentity company={company} verifiedLabel={t("companies.verified")} />
            {company.description ? <p>{company.description}</p> : <p>{t("companies.noDescription")}</p>}
            <div className="company-card__facts">
              {company.industry ? <span><BriefcaseBusiness size={16} aria-hidden="true" />{company.industry}</span> : null}
              {company.city ? <span><MapPin size={16} aria-hidden="true" />{company.city}</span> : null}
            </div>
            <div className="company-card__rating"><Star size={16} aria-hidden="true" /><span>{t("companies.ratingUnavailable")}</span></div>
            <div className="company-card__footer">
              <span className="company-card__vacancies"><BriefcaseBusiness size={16} aria-hidden="true" />{t("companies.jobCount", { count: company.openJobsCount })}</span>
              <Link className="ui-button ui-button--secondary ui-button--md" to={`/companies/${company.id}`}>{t("companies.openProfile")}<ArrowRight size={17} aria-hidden="true" /></Link>
            </div>
          </article>
        ))}</div> : null}
        {companies.data && companies.data.totalPages > 1 ? <div className="results-pagination"><Pagination page={companies.data.page} totalPages={companies.data.totalPages} onChange={(nextPage) => { const next = new URLSearchParams(params); next.set("page", String(nextPage)); setParams(next); }} label={t("companies.pagination")} previousLabel={t("common.previous")} nextLabel={t("common.next")} pageLabel={(value) => t("common.page", { page: value })} /></div> : null}
      </div>
    </SiteShell>
  );
}
