import { useState } from "react";
import type { JobSummary } from "../../api/models/job";
import { companyInitials, companyPresentationFor } from "../../lib/companyPresentation";

interface CompanyLogoProps {
  job: JobSummary;
  size?: "list" | "detail";
}

export function CompanyLogo({ job, size = "list" }: CompanyLogoProps) {
  const presentation = companyPresentationFor(job);
  const [failedUrl, setFailedUrl] = useState<string>();
  const showLogo = Boolean(presentation.logoUrl && failedUrl !== presentation.logoUrl);

  return (
    <span className={`company-logo company-logo--${size}`} aria-hidden="true">
      {showLogo ? (
        <img src={presentation.logoUrl} alt="" onError={() => setFailedUrl(presentation.logoUrl)} />
      ) : (
        <span>{companyInitials(presentation.name || job.company.name)}</span>
      )}
    </span>
  );
}
