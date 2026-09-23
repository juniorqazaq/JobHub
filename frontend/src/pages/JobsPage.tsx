import { useQuery } from "@tanstack/react-query";
import { Filter, RotateCcw } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSearchParams } from "react-router-dom";
import { repositories } from "../api/repositories";
import { JobSearchForm, type JobSearchValues } from "../components/jobs/JobSearchForm";
import { MarketplaceJobRow } from "../components/jobs/MarketplaceJobRow";
import { Button } from "../components/ui/Button";
import { EmptyState, ErrorState, JobListSkeleton } from "../components/ui/Feedback";
import { Pagination } from "../components/ui/Navigation";
import { Drawer } from "../components/ui/Overlays";
import { SiteShell } from "../layouts/SiteShell";
import { parseJobSearchParams, serializeJobSearchParams } from "../lib/jobSearchParams";

export function JobsPage() {
  const { t } = useTranslation();
  const [urlParams, setUrlParams] = useSearchParams();
  const [filtersOpen, setFiltersOpen] = useState(false);
  const search = useMemo(() => ({ ...parseJobSearchParams(urlParams), pageSize: 20 }), [urlParams]);
  const jobs = useQuery({
    queryKey: ["jobs", search],
    queryFn: () => repositories.jobs.search(search),
  });

  const updateSearch = (values: JobSearchValues) => {
    setUrlParams(serializeJobSearchParams({ query: values.query || undefined, location: values.location || undefined, sort: search.sort, page: 1 }));
    setFiltersOpen(false);
  };
  const clear = () => setUrlParams(new URLSearchParams());
  const changePage = (page: number) => setUrlParams(serializeJobSearchParams({ ...search, page }));
  const changeSort = (sort: "newest" | "oldest") => setUrlParams(serializeJobSearchParams({ ...search, sort, page: 1 }));
  const hasFilters = Boolean(search.query || search.location);

  return (
    <SiteShell>
      <div className="page-container jobs-page">
        <header className="jobs-page__header">
          <div><h1>{t("jobs.title")}</h1><p>{t("jobs.description")}</p></div>
          <Button className="mobile-filter-button" variant="secondary" leadingIcon={<Filter size={18} />} onClick={() => setFiltersOpen(true)}>{t("jobs.filters")}</Button>
        </header>
        <JobSearchForm variant="compact" initialValues={{ query: search.query, location: search.location }} onSubmit={updateSearch} />
        <div className="jobs-workspace">
          <aside className="filter-rail" aria-labelledby="filter-title">
            <div className="filter-rail__heading"><h2 id="filter-title">{t("jobs.filters")}</h2>{hasFilters ? <button type="button" onClick={clear}>{t("jobs.clearAll")}</button> : null}</div>
            <dl>
              <div><dt>{t("search.queryLabel")}</dt><dd>{search.query || t("jobs.anyKeyword")}</dd></div>
              <div><dt>{t("search.locationLabel")}</dt><dd>{search.location || t("jobs.anyLocation")}</dd></div>
            </dl>
            {hasFilters ? <Button variant="secondary" leadingIcon={<RotateCcw size={16} />} onClick={clear}>{t("jobs.reset")}</Button> : null}
          </aside>
          <section className="results-panel" aria-labelledby="results-title">
            <div className="results-toolbar">
              <h2 id="results-title">{jobs.data ? t("jobs.resultCount", { count: jobs.data.total }) : t("jobs.results")}</h2>
              <label><span>{t("jobs.sortLabel")}</span><select value={search.sort} onChange={(event) => changeSort(event.target.value === "oldest" ? "oldest" : "newest")}><option value="newest">{t("jobs.newest")}</option><option value="oldest">{t("jobs.oldest")}</option></select></label>
            </div>
            {jobs.isPending ? <JobListSkeleton label={t("jobs.loading")} /> : null}
            {jobs.isError ? <ErrorState title={t("jobs.errorTitle")} description={t("jobs.errorDescription")} actionLabel={t("common.retry")} onAction={() => void jobs.refetch()} /> : null}
            {jobs.data?.items.length === 0 ? <EmptyState title={t("jobs.emptyTitle")} description={t("jobs.emptyDescription")} actionLabel={hasFilters ? t("jobs.clearAll") : undefined} onAction={hasFilters ? clear : undefined} /> : null}
            {jobs.data?.items.length ? <div className="job-list">{jobs.data.items.map((job) => <MarketplaceJobRow key={job.id} job={job} />)}</div> : null}
            {jobs.data && jobs.data.totalPages > 1 ? <div className="results-pagination"><Pagination page={jobs.data.page} totalPages={jobs.data.totalPages} onChange={changePage} label={t("jobs.pagination")} previousLabel={t("common.previous")} nextLabel={t("common.next")} pageLabel={(page) => t("common.page", { page })} /></div> : null}
          </section>
        </div>
      </div>
      <Drawer open={filtersOpen} onClose={() => setFiltersOpen(false)} closeLabel={t("common.close")} title={t("jobs.filters")} description={t("jobs.mobileFiltersDescription")}>
        <JobSearchForm variant="stacked" initialValues={{ query: search.query, location: search.location }} onSubmit={updateSearch} />
      </Drawer>
    </SiteShell>
  );
}
