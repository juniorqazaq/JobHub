import { useQuery } from "@tanstack/react-query";
import { Filter, X } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSearchParams } from "react-router-dom";
import type { JobSearchParams } from "../api/models/job";
import { repositories } from "../api/repositories";
import { JobFilters } from "../components/jobs/JobFilters";
import { JobSearchForm, type JobSearchValues } from "../components/jobs/JobSearchForm";
import { MarketplaceJobRow } from "../components/jobs/MarketplaceJobRow";
import { Button } from "../components/ui/Button";
import { EmptyState, ErrorState, JobListSkeleton } from "../components/ui/Feedback";
import { Pagination } from "../components/ui/Navigation";
import { Drawer } from "../components/ui/Overlays";
import { SiteShell } from "../layouts/SiteShell";
import { cityName } from "../lib/cities";
import { parseJobSearchParams, serializeJobSearchParams } from "../lib/jobSearchParams";

export function JobsPage() {
  const { t, i18n } = useTranslation();
  const [urlParams, setUrlParams] = useSearchParams();
  const [filtersOpen, setFiltersOpen] = useState(false);
  const search = useMemo(() => ({ ...parseJobSearchParams(urlParams), pageSize: 20 }), [urlParams]);
  const jobs = useQuery({ queryKey: ["jobs", search], queryFn: () => repositories.jobs.search(search), staleTime: 2 * 60_000 });
  const setSearch = (next: JobSearchParams) => setUrlParams(serializeJobSearchParams(next));
  const updateKeyword = (values: JobSearchValues) => setSearch({ ...search, query: values.query || undefined, page: 1 });
  const clear = () => setUrlParams(new URLSearchParams());
  const hasFilters = Boolean(search.query || search.city || search.workModes?.length || search.salaryMin != null || search.experience || search.employment || search.datePosted);
  const chips = makeChips(search, t, i18n.resolvedLanguage ?? i18n.language);
  return (
    <SiteShell>
      <div className="page-container jobs-page">
        <header className="jobs-page__header">
          <div><h1>{t("jobs.title")}</h1><p>{t("jobs.description")}</p></div>
          <Button className="mobile-filter-button" variant="secondary" leadingIcon={<Filter size={18} />} onClick={() => setFiltersOpen(true)}>{t("jobs.filters")}</Button>
        </header>
        <JobSearchForm variant="compact" showCity={false} initialValues={{ query: search.query }} onSubmit={updateKeyword} />
        {chips.length ? <div className="active-filters" aria-label={t("jobs.activeFilters")}>{chips.map((chip) => <button key={chip.key} type="button" onClick={() => setSearch({ ...search, ...chip.clear, page: 1 })}>{chip.label}<X size={14} aria-hidden="true" /></button>)}<button className="active-filters__reset" type="button" onClick={clear}>{t("jobs.clearAll")}</button></div> : null}
        <div className="jobs-workspace">
          <aside className="filter-rail" aria-labelledby="filter-title">
            <div className="filter-rail__heading"><h2 id="filter-title">{t("jobs.filters")}</h2>{hasFilters ? <button type="button" onClick={clear}>{t("jobs.clearAll")}</button> : null}</div>
            <JobFilters search={search} onChange={setSearch} onReset={clear} />
          </aside>
          <section className="results-panel" aria-labelledby="results-title">
            <div className="results-toolbar"><h2 id="results-title">{jobs.data ? t("jobs.resultCount", { count: jobs.data.total }) : t("jobs.results")}</h2><label><span>{t("jobs.sortLabel")}</span><select value={search.sort} onChange={(event) => setSearch({ ...search, sort: event.target.value === "oldest" ? "oldest" : "newest", page: 1 })}><option value="newest">{t("jobs.newest")}</option><option value="oldest">{t("jobs.oldest")}</option></select></label></div>
            {jobs.isPending ? <JobListSkeleton label={t("jobs.loading")} /> : null}
            {jobs.isError ? <ErrorState title={t("jobs.errorTitle")} description={t("jobs.errorDescription")} actionLabel={t("common.retry")} onAction={() => void jobs.refetch()} /> : null}
            {jobs.data?.items.length === 0 ? <EmptyState illustration="/illustrations/empty-jobs.png" title={t("jobs.emptyTitle")} description={t("jobs.emptyDescription")} actionLabel={t("jobs.reset")} onAction={clear} /> : null}
            {jobs.data?.items.length ? <div className="job-list">{jobs.data.items.map((job) => <MarketplaceJobRow key={job.id} job={job} />)}</div> : null}
            {jobs.data && jobs.data.totalPages > 1 ? <div className="results-pagination"><Pagination page={jobs.data.page} totalPages={jobs.data.totalPages} onChange={(page) => setSearch({ ...search, page })} label={t("jobs.pagination")} previousLabel={t("common.previous")} nextLabel={t("common.next")} pageLabel={(page) => t("common.page", { page })} /></div> : null}
          </section>
        </div>
      </div>
      <Drawer open={filtersOpen} onClose={() => setFiltersOpen(false)} closeLabel={t("common.close")} title={t("jobs.filters")} description={t("jobs.mobileFiltersDescription")}>
        <JobSearchForm variant="stacked" showCity={false} initialValues={{ query: search.query }} onSubmit={(values) => { updateKeyword(values); setFiltersOpen(false); }} />
        <JobFilters search={search} onChange={setSearch} onReset={clear} />
      </Drawer>
    </SiteShell>
  );
}

function makeChips(search: JobSearchParams, t: (key: string, options?: Record<string, unknown>) => string, language: string) {
  const chips: Array<{ key: string; label: string; clear: Partial<JobSearchParams> }> = [];
  if (search.query) chips.push({ key: "q", label: search.query, clear: { query: undefined } });
  if (search.city) chips.push({ key: "city", label: cityName(search.city, language), clear: { city: undefined } });
  for (const mode of search.workModes ?? []) chips.push({ key: `mode-${mode}`, label: t(`employer.workModes.${mode}`), clear: { workModes: search.workModes?.filter((item) => item !== mode) } });
  if (search.salaryMin != null && search.currency) chips.push({ key: "salary", label: `${new Intl.NumberFormat(language).format(search.salaryMin)}+ ${search.currency}`, clear: { salaryMin: undefined, currency: undefined } });
  if (search.experience) chips.push({ key: "experience", label: t(`employer.experienceLevels.${search.experience}`), clear: { experience: undefined } });
  if (search.employment) chips.push({ key: "employment", label: t(`employer.employmentTypes.${search.employment}`), clear: { employment: undefined } });
  if (search.datePosted) chips.push({ key: "date", label: t(`jobs.dateOptions.${search.datePosted}`), clear: { datePosted: undefined } });
  return chips;
}
