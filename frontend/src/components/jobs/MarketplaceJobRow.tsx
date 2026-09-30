import { BriefcaseBusiness, ChevronRight, ExternalLink, MapPin } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link, useLocation } from "react-router-dom";
import type { JobSummary } from "../../api/models/job";
import { JobSaveButton } from "./JobSaveButton";
import { formatSalary } from "../../lib/formatSalary";
import { displayJobLocation } from "../../lib/jobLocation";
import { companyPresentationFor } from "../../lib/companyPresentation";
import { CompanyLogo } from "./CompanyLogo";

export function MarketplaceJobRow({ job }: { job: JobSummary }) {
  const { t, i18n } = useTranslation();
  const location = useLocation();
  const external = job.source.type === "external";
  const posted = formatDate(job.postedAt, i18n.language);
  const company = companyPresentationFor(job);
  const summary = cleanCardSummary(job.summary) || (external ? t("jobs.externalSummary") : "");
  const salary = job.salaryRaw || (job.salary ? formatSalary(job.salary, i18n.language, t(`employer.salaryPeriods.${job.salary.period}`)) : "");
  const compactTags = Array.from(new Set([job.category, ...job.tags].filter((value): value is string => Boolean(value)))).slice(0, 3);
  const detailTarget = { pathname: `/jobs/${job.id}` };
  const detailState = { from: `${location.pathname}${location.search}` };

  return (
    <article className="job-row">
      <JobSaveButton job={job} compact />
      <Link className="job-row__title-link" to={detailTarget} state={detailState}><h3>{job.title}</h3></Link>
      {job.company.id ? <Link className="job-row__company" to={`/companies/${job.company.id}`} aria-label={`${t("companies.openProfile")}: ${company.name || t("common.notProvided")}`}><CompanyLogo job={job} /><span>{company.name || t("common.notProvided")}</span></Link> : <div className="job-row__company"><CompanyLogo job={job} /><span>{company.name || t("common.notProvided")}</span></div>}
      <div className="job-row__facts">
        <span><MapPin size={16} aria-hidden="true" />{displayJobLocation(job, i18n.language) || t("common.notProvided")}</span>
        {job.workMode ? <span>{job.source.type === "native" ? t(`employer.workModes.${job.workMode}`) : job.workMode}</span> : null}
        {job.employmentType ? <span><BriefcaseBusiness size={16} aria-hidden="true" />{job.source.type === "native" ? t(`employer.employmentTypes.${job.employmentType}`) : job.employmentType}</span> : null}
        {salary ? <strong>{salary}</strong> : null}
      </div>
      {summary ? <p className="job-row__summary">{summary}</p> : <div className="job-row__summary job-row__summary--empty" aria-hidden="true" />}
      <div className="job-row__tags" aria-hidden={compactTags.length === 0}>{compactTags.map((tag) => <span key={tag}>{tag}</span>)}</div>
      <div className="job-row__footer">
        <span className={external ? "source-cue source-cue--external" : "source-cue"}>{external ? <ExternalLink size={14} aria-hidden="true" /> : null}{external ? t("jobs.externalSource", { source: job.source.name }) : job.source.name}</span>
        <span>{t("jobs.updated", { date: posted })}</span>
        <Link className="job-row__action ui-button ui-button--secondary ui-button--md" to={detailTarget} state={detailState}>{external ? t("jobs.viewExternalDetails") : t("jobs.viewDetails")}<ChevronRight size={17} aria-hidden="true" /></Link>
      </div>
    </article>
  );
}

function cleanCardSummary(value?: string) {
  if (!value) return "";
  const compact = value.replace(/\s+/g, " ").trim();
  if (/<[^>]+>|\b(?:function|window|document)\s*[.(]|\bym\s*\(/i.test(compact)) return "";
  return compact.length > 240 ? `${compact.slice(0, 237).trimEnd()}…` : compact;
}

function formatDate(value: string, language: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "" : new Intl.DateTimeFormat(language, { dateStyle: "medium" }).format(date);
}
