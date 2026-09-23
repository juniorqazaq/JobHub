import { BriefcaseBusiness, ChevronRight, ExternalLink, MapPin } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link, useLocation } from "react-router-dom";
import type { JobSummary } from "../../api/models/job";

export function MarketplaceJobRow({ job }: { job: JobSummary }) {
  const { t, i18n } = useTranslation();
  const location = useLocation();
  const external = job.source.type === "external";
  const posted = formatDate(job.postedAt, i18n.language);

  return (
    <article className="job-row">
      <Link className="job-row__link" to={`/jobs/${job.id}`} state={{ from: `${location.pathname}${location.search}` }}>
        <div className="job-row__body">
          <h3>{job.title}</h3>
          <div className="job-row__facts">
            <span><BriefcaseBusiness size={16} aria-hidden="true" />{job.company.name || t("common.notProvided")}</span>
            <span><MapPin size={16} aria-hidden="true" />{job.location || t("common.notProvided")}</span>
            {job.employmentType ? <span>{job.employmentType}</span> : null}
          </div>
          {job.summary ? <p>{job.summary}</p> : null}
          <div className="job-row__footer">
            <span className={external ? "source-cue source-cue--external" : "source-cue"}>
              {external ? <ExternalLink size={14} aria-hidden="true" /> : null}
              {job.source.name}
            </span>
            <span>{t("jobs.updated", { date: posted })}</span>
            {job.salaryRaw ? <strong>{job.salaryRaw}</strong> : null}
          </div>
        </div>
        <ChevronRight className="job-row__arrow" size={20} aria-hidden="true" />
      </Link>
    </article>
  );
}

function formatDate(value: string, language: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "" : new Intl.DateTimeFormat(language, { dateStyle: "medium" }).format(date);
}
