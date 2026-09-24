import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, BriefcaseBusiness, ExternalLink, MapPin } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link, useLocation, useParams } from "react-router-dom";
import { repositories } from "../api/repositories";
import { ErrorState, Skeleton } from "../components/ui/Feedback";
import { SiteShell } from "../layouts/SiteShell";
import { JobSaveButton } from "../components/jobs/JobSaveButton";
import { ApplyAction } from "../components/jobs/ApplyAction";
import { formatSalary } from "../lib/formatSalary";

interface DetailLocationState { from?: string }

export function JobDetailPage() {
  const { t, i18n } = useTranslation();
  const { jobId = "" } = useParams();
  const location = useLocation();
  const backTo = (location.state as DetailLocationState | null)?.from || "/jobs";
  const job = useQuery({ queryKey: ["jobs", "detail", jobId], queryFn: () => repositories.jobs.getById(jobId), enabled: Boolean(jobId) });

  return (
    <SiteShell>
      <div className="page-container detail-page">
        <Link className="back-link" to={backTo}><ArrowLeft size={17} aria-hidden="true" />{t("jobDetail.back")}</Link>
        {job.isPending ? <DetailSkeleton label={t("jobDetail.loading")} /> : null}
        {job.isError ? <ErrorState title={t("jobDetail.errorTitle")} description={t("jobDetail.errorDescription")} actionLabel={t("common.retry")} onAction={() => void job.refetch()} secondary={<Link className="text-link" to="/jobs">{t("jobDetail.back")}</Link>} /> : null}
        {job.data ? <div className="detail-layout">
          <article className="detail-content">
            <header>
              <h1>{job.data.title}</h1>
              <p className="detail-company">{job.data.company.name || t("common.notProvided")}</p>
              <div className="detail-facts">
                <span><MapPin size={19} aria-hidden="true" />{job.data.location || t("common.notProvided")}</span>
                {job.data.salaryRaw || job.data.salary ? <strong>{job.data.salaryRaw || formatSalary(job.data.salary!, i18n.language, t(`employer.salaryPeriods.${job.data.salary!.period}`))}</strong> : null}
                {job.data.employmentType ? <span><BriefcaseBusiness size={19} aria-hidden="true" />{job.data.source.type === "native" ? t(`employer.employmentTypes.${job.data.employmentType}`) : job.data.employmentType}</span> : null}
              </div>
            </header>
            <div className="detail-source-cue"><span aria-hidden="true" />{t(job.data.source.type === "external" ? "jobDetail.importedFrom" : "jobDetail.publishedBy", { source: job.data.source.name })}</div>
            <section aria-labelledby="about-job-title">
              <h2 id="about-job-title">{t("jobDetail.aboutTitle")}</h2>
              <p className="detail-description">{job.data.summary || t("jobDetail.noDescription")}</p>
              {job.data.descriptionKind === "snippet" ? <p className="detail-note">{t("jobDetail.snippetNote")}</p> : null}
            </section>
            {job.data.responsibilities ? <DetailListSection title={t("jobDetail.responsibilitiesTitle")} value={job.data.responsibilities} /> : null}
            {job.data.requirements ? <DetailListSection title={t("jobDetail.requirementsTitle")} value={job.data.requirements} /> : null}
            {job.data.niceToHave ? <DetailListSection title={t("jobDetail.niceToHaveTitle")} value={job.data.niceToHave} /> : null}
            {job.data.tags.length ? <section><h2>{t("jobDetail.skillsTitle")}</h2><ul className="detail-tag-list">{job.data.tags.map((skill) => <li key={skill}>{skill}</li>)}</ul></section> : null}
            {job.data.benefits?.length ? <section><h2>{t("jobDetail.benefitsTitle")}</h2><ul className="detail-list">{job.data.benefits.map((benefit) => <li key={benefit}>{benefit}</li>)}</ul></section> : null}
          </article>
          <aside className="detail-rail">
            {job.data.application.method === "external" && job.data.application.ctaUrl ? <>
              <JobSaveButton job={job.data} />
              <a className="external-cta" href={job.data.application.ctaUrl} target="_blank" rel="noreferrer">{t("jobDetail.externalCta")}<ExternalLink size={18} aria-hidden="true" /></a>
              <p className="detail-rail__hint">{t("jobDetail.externalHint", { source: job.data.source.name })}</p>
            </> : <div className="job-detail-actions"><JobSaveButton job={job.data} /><ApplyAction job={job.data} /></div>}
            <section>
              <h2>{t("jobDetail.sourceTitle")}</h2>
              <p className="detail-source-name">{job.data.source.name}</p>
              <dl>
                {job.data.firstSeenAt ? <div><dt>{t("jobDetail.firstSeen")}</dt><dd>{formatDate(job.data.firstSeenAt, i18n.language)}</dd></div> : null}
                {job.data.lastSyncedAt ? <div><dt>{t("jobDetail.lastUpdated")}</dt><dd>{formatDate(job.data.lastSyncedAt, i18n.language)}</dd></div> : null}
              </dl>
            </section>
            {job.data.source.type === "external" ? <section><h2>{t("jobDetail.whyTitle")}</h2><p>{t("jobDetail.whyExternal", { source: job.data.source.name })}</p></section> : null}
          </aside>
        </div> : null}
      </div>
    </SiteShell>
  );
}

function DetailListSection({ title, value }: { title: string; value: string }) { const items = value.split(/\r?\n/).map((item) => item.trim()).filter(Boolean); return <section><h2>{title}</h2>{items.length > 1 ? <ul className="detail-list">{items.map((item, index) => <li key={`${index}-${item.slice(0, 24)}`}>{item}</li>)}</ul> : <p className="detail-description">{value}</p>}</section>; }

function DetailSkeleton({ label }: { label: string }) {
  return <div className="detail-skeleton" role="status"><span className="visually-hidden">{label}</span><Skeleton className="detail-skeleton__title" /><Skeleton /><Skeleton /><Skeleton className="detail-skeleton__body" /></div>;
}

function formatDate(value: string, language: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "" : new Intl.DateTimeFormat(language, { dateStyle: "medium" }).format(date);
}
