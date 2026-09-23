import { ExternalLink } from "lucide-react";
import type { JobSummary } from "../../api/models/job";
import { Badge } from "../ui/DataDisplay";

interface SourceAwareJobRowProps {
  job: JobSummary;
  externalLabel: string;
  sourceLabel: string;
  snippetLabel: string;
}

export function SourceAwareJobRow({ job, externalLabel, sourceLabel, snippetLabel }: SourceAwareJobRowProps) {
  const external = job.source.type === "external";
  const ctaUrl = external ? job.application.ctaUrl : undefined;

  return (
    <article className="source-job-row">
      <div className="source-job-row__main">
        <div className="source-job-row__meta">
          <Badge tone={external ? "warning" : "success"}>{sourceLabel}: {job.source.name}</Badge>
          {job.descriptionKind === "snippet" ? <span>{snippetLabel}</span> : null}
        </div>
        <h3>{job.title}</h3>
        <p className="source-job-row__company">{job.company.name || "—"} · {job.location || "—"}</p>
        <p className="source-job-row__summary">{job.summary}</p>
        {job.salaryRaw ? <p className="source-job-row__salary">{job.salaryRaw}</p> : null}
      </div>
      {ctaUrl ? (
        <a className="source-job-row__cta" href={ctaUrl} target="_blank" rel="noreferrer">
          {externalLabel}<ExternalLink size={16} aria-hidden="true" />
        </a>
      ) : null}
    </article>
  );
}
