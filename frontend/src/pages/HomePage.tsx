import { useQuery } from "@tanstack/react-query";
import {
  ArrowRight, Banknote, BarChart3, BriefcaseBusiness,
  Code2, MapPin, Plane, Wrench,
} from "lucide-react";
import type { ComponentType } from "react";
import { useTranslation } from "react-i18next";
import { Link, useNavigate } from "react-router-dom";
import type { JobSearchParams, JobSummary } from "../api/models/job";
import { repositories } from "../api/repositories";
import { CompanyLogo } from "../components/jobs/CompanyLogo";
import { JobSaveButton } from "../components/jobs/JobSaveButton";
import { JobSearchForm, type JobSearchValues } from "../components/jobs/JobSearchForm";
import { ErrorState, JobListSkeleton } from "../components/ui/Feedback";
import { SiteShell } from "../layouts/SiteShell";
import { companyPresentationFor } from "../lib/companyPresentation";
import { displayJobLocation } from "../lib/jobLocation";
import { serializeJobSearchParams } from "../lib/jobSearchParams";
import { usePreferredSearchCity } from "../lib/searchCityPreference";

interface LandingLink {
  key: string;
  count: number;
  href: string;
  icon: ComponentType<{ size?: number; "aria-hidden"?: "true" }>;
}

export function HomePage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const preferredCity = usePreferredSearchCity();
  const jobsQuery = useQuery({
    queryKey: ["jobs", "landing", "all"],
    queryFn: async () => {
      const firstPage = await repositories.jobs.search({ page: 1, pageSize: 100, sort: "newest" });
      if (firstPage.totalPages <= 1) return firstPage.items;
      const remainingPages = await Promise.all(
        Array.from({ length: firstPage.totalPages - 1 }, (_, index) =>
          repositories.jobs.search({ page: index + 2, pageSize: 100, sort: "newest" }),
        ),
      );
      return [firstPage, ...remainingPages].flatMap((page) => page.items);
    },
  });
  const jobs = jobsQuery.data ?? [];
  const trendingJobs = jobs.slice(0, 4);
  const companies = topCompanies(jobs);
  const preferences = preferenceLinks(jobs);

  const search = (values: JobSearchValues) => {
    const params = serializeJobSearchParams({
      query: values.query || undefined,
      city: values.city || undefined,
      sort: "newest",
      page: 1,
    });
    void navigate({ pathname: "/jobs", search: params.toString() });
  };

  return (
    <SiteShell variant="landing">
      <section className="home-hero">
        <span className="home-shape home-shape--blue" aria-hidden="true" />
        <span className="home-shape home-shape--yellow" aria-hidden="true" />
        <div className="page-container home-hero__inner">
          <div className="home-hero__copy">
            <h1>{t("home.title")}</h1>
            <p>{t("home.description")}</p>
          </div>
          <div className="home-hero__visual">
            <img className="home-hero__illustration" src="/illustrations/jobhub-astana-skyline.png" width="1774" height="887" alt="" />
            <p className="home-hero__note">{t("home.heroNote")}</p>
          </div>
          <JobSearchForm variant="hero" initialValues={{ city: preferredCity }} onSubmit={search} />
        </div>
      </section>

      <section className="landing-section page-container" aria-labelledby="trending-jobs-title">
        <SectionHeading id="trending-jobs-title" title={t("home.trending.title")} description={t("home.trending.description")} href="/jobs" action={t("home.viewAll")} />
        {jobsQuery.isPending ? <JobListSkeleton label={t("jobs.loading")} /> : null}
        {jobsQuery.isError ? <ErrorState title={t("jobs.errorTitle")} description={t("jobs.errorDescription")} actionLabel={t("common.retry")} onAction={() => void jobsQuery.refetch()} /> : null}
        {trendingJobs.length ? (
          <div className="home-trending-layout">
            <div className="home-job-grid">{trendingJobs.map((job) => <HomeJobCard key={job.id} job={job} />)}</div>
            <Link className="home-opportunity-card" to="/jobs">
              <img src="/brand/jobhub-mark.png" width="512" height="512" alt="" />
              <span><strong>{t("home.opportunity.title")}</strong><small>{t("home.opportunity.description")}</small></span>
              <ArrowRight size={22} aria-hidden="true" />
            </Link>
          </div>
        ) : null}
      </section>

      {companies.length ? (
        <section className="landing-section landing-section--companies page-container" aria-labelledby="top-companies-title">
          <SectionHeading id="top-companies-title" title={t("home.topCompanies.title")} description={t("home.topCompanies.description")} href="/companies" action={t("home.topCompanies.action")} />
          <div className="home-company-rail">
            {companies.map(({ name, count, representative }) => (
              <Link className="home-company-card" key={name} to={jobsHref({ query: name })}>
                <CompanyLogo job={representative} />
                <strong>{name}</strong>
                <span>{t("home.jobsCount", { count })}</span>
                <ArrowRight size={18} aria-hidden="true" />
              </Link>
            ))}
          </div>
        </section>
      ) : null}

      {preferences.length ? (
        <section className="home-preferences" aria-labelledby="preferences-title">
          <div className="page-container">
            <div className="home-preferences__intro"><h2 id="preferences-title">{t("home.preferences.title")}</h2><p>{t("home.preferences.description")}</p></div>
            <div className="home-preference-grid">
              {preferences.map(({ key, count, href }) => (
                <Link key={key} to={href} className="home-preference-card">
                  <img className="home-preference-card__photo" src={`/images/professions/${key}.jpg`} width="640" height="320" alt="" loading="lazy" decoding="async" />
                  <strong>{t(`home.preferences.items.${key}`)}</strong>
                  <small>{t("home.jobsCount", { count })}</small>
                  <ArrowRight size={18} aria-hidden="true" />
                </Link>
              ))}
            </div>
          </div>
        </section>
      ) : null}

      <section className="home-faq" aria-labelledby="faq-title">
        <span className="home-faq__shape" aria-hidden="true" />
        <div className="page-container home-faq__inner">
          <div className="home-faq__intro"><h2 id="faq-title">{t("home.faq.title")}</h2><p>{t("home.faq.description")}</p></div>
          <div className="home-faq__list">
            {(["registration", "apply", "sources", "save", "contact"] as const).map((key, index) => (
              <details key={key} open={index === 0}><summary>{t(`home.faq.items.${key}.question`)}</summary><p>{t(`home.faq.items.${key}.answer`)}</p></details>
            ))}
          </div>
        </div>
      </section>

      <section className="home-cta" aria-labelledby="home-cta-title">
        <span className="home-cta__shape home-cta__shape--left" aria-hidden="true" />
        <span className="home-cta__shape home-cta__shape--right" aria-hidden="true" />
        <div className="page-container">
          <h2 id="home-cta-title">{t("home.cta.title")}</h2>
          <p>{t("home.cta.description")}</p>
          <div className="home-cta__action">
            <svg className="home-cta__sketch" viewBox="0 0 130 90" fill="none" aria-hidden="true" focusable="false">
              <path d="M8 12C14 63 60 77 82 51C101 27 58 17 65 48C71 70 100 71 119 65M106 55L121 65L108 78" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
            <Link className="ui-button ui-button--primary ui-button--lg" to="/jobs"><span>{t("home.cta.action")}</span><ArrowRight size={18} aria-hidden="true" /></Link>
          </div>
        </div>
      </section>
    </SiteShell>
  );
}

function SectionHeading({ id, title, description, href, action }: { id: string; title: string; description: string; href?: string; action?: string }) {
  return <div className="home-section-heading"><div><h2 id={id}>{title}</h2><p>{description}</p></div>{href && action ? <Link to={href}>{action}<ArrowRight size={17} aria-hidden="true" /></Link> : null}</div>;
}

function HomeJobCard({ job }: { job: JobSummary }) {
  const { t, i18n } = useTranslation();
  const company = companyPresentationFor(job);
  const location = displayJobLocation(job, i18n.language);
  return (
    <article className="home-job-card">
      <JobSaveButton job={job} compact />
      <Link to={`/jobs/${job.id}`}>
        <div className="home-job-card__company"><CompanyLogo job={job} /><span>{company.name}</span></div>
        <h3>{job.title}</h3>
        <div className="home-job-card__meta">{location ? <span><MapPin size={16} aria-hidden="true" />{location}</span> : null}{job.employmentType ? <span><BriefcaseBusiness size={16} aria-hidden="true" />{job.employmentType}</span> : null}</div>
        <span className="home-job-card__source">{job.source.name}</span>
        <span className="home-job-card__open">{t("home.trending.open")}<ArrowRight size={17} aria-hidden="true" /></span>
      </Link>
    </article>
  );
}

function topCompanies(jobs: JobSummary[]) {
  const companies = new Map<string, { name: string; count: number; representative: JobSummary }>();
  jobs.forEach((job) => {
    const name = companyPresentationFor(job).name || job.company.name;
    const current = companies.get(name);
    if (current) current.count += 1;
    else companies.set(name, { name, count: 1, representative: job });
  });
  return [...companies.values()].sort((a, b) => b.count - a.count || a.name.localeCompare(b.name)).slice(0, 5);
}

function preferenceLinks(jobs: JobSummary[]): LandingLink[] {
  const definitions = [
    { key: "sales", query: "продаж", icon: BarChart3, matches: /(продаж|sales)/i },
    { key: "engineering", query: "инженер", icon: Wrench, matches: /(инженер|engineer|engineering)/i },
    { key: "finance", query: "financial", icon: Banknote, matches: /(финанс|financ)/i },
    { key: "it", query: "разработ", icon: Code2, matches: /(разработ|developer|backend|\bqa\b|information technology)/i },
    { key: "aviation", query: "Air Astana", icon: Plane, matches: /(air astana|pilot|cabin crew|aviation)/i },
  ];
  return definitions.map((item) => ({ key: item.key, icon: item.icon, href: jobsHref({ query: item.query }), count: jobs.filter((job) => item.matches.test(`${job.title} ${job.category ?? ""} ${job.company.name}`)).length })).filter((item) => item.count > 0);
}

function jobsHref(params: JobSearchParams) {
  const search = serializeJobSearchParams({ ...params, sort: "newest", page: 1 });
  return `/jobs?${search.toString()}`;
}
