import { useQuery } from "@tanstack/react-query";
import { ArrowRight } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link, useNavigate } from "react-router-dom";
import { repositories } from "../api/repositories";
import { JobSearchForm, type JobSearchValues } from "../components/jobs/JobSearchForm";
import { MarketplaceJobRow } from "../components/jobs/MarketplaceJobRow";
import { EmptyState, ErrorState, JobListSkeleton } from "../components/ui/Feedback";
import { SiteShell } from "../layouts/SiteShell";
import { serializeJobSearchParams } from "../lib/jobSearchParams";
import { usePreferredSearchCity } from "../lib/searchCityPreference";

export function HomePage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const preferredCity = usePreferredSearchCity();
  const latestJobs = useQuery({
    queryKey: ["jobs", "latest", 3, preferredCity],
    queryFn: () => repositories.jobs.search({ page: 1, pageSize: 3, sort: "newest", preferredCity }),
  });

  const search = (values: JobSearchValues) => {
    const params = serializeJobSearchParams({ query: values.query || undefined, city: values.city || undefined, sort: "newest", page: 1 });
    void navigate({ pathname: "/jobs", search: params.toString() });
  };

  return (
    <SiteShell>
      <section className="home-hero">
        <div className="page-container home-hero__inner">
          <h1>{t("home.title")}</h1>
          <p>{t("home.description")}</p>
          <JobSearchForm variant="hero" initialValues={{ city: preferredCity }} onSubmit={search} />
        </div>
      </section>
      <section className="page-container latest-jobs" aria-labelledby="latest-jobs-title">
        <div className="section-heading">
          <div><h2 id="latest-jobs-title">{t("home.latestTitle")}</h2><p>{t("home.latestDescription")}</p></div>
          <Link to="/jobs">{t("home.viewAll")}<ArrowRight size={17} aria-hidden="true" /></Link>
        </div>
        {latestJobs.isPending ? <JobListSkeleton label={t("jobs.loading")} /> : null}
        {latestJobs.isError ? <ErrorState title={t("jobs.errorTitle")} description={t("jobs.errorDescription")} actionLabel={t("common.retry")} onAction={() => void latestJobs.refetch()} /> : null}
        {latestJobs.data?.items.length === 0 ? <EmptyState title={t("jobs.emptyTitle")} description={t("jobs.emptyDescription")} /> : null}
        {latestJobs.data?.items.length ? <div className="job-list">{latestJobs.data.items.map((job) => <MarketplaceJobRow key={job.id} job={job} />)}</div> : null}
      </section>
    </SiteShell>
  );
}
