import { useState } from "react";
import { Building2 } from "lucide-react";
import type { JobSummary } from "../../api/models/job";
import { companyMarkFor, companyPresentationFor } from "../../lib/companyPresentation";

interface CompanyLogoProps {
  job: JobSummary;
  size?: "list" | "detail";
}

export function CompanyLogo({ job, size = "list" }: CompanyLogoProps) {
  const presentation = companyPresentationFor(job);
  const mark = companyMarkFor(presentation.name || job.company.name, presentation.logoUrl);
  const [failedUrl, setFailedUrl] = useState<string>();
  const showLogo = Boolean(mark.logoUrl && failedUrl !== mark.logoUrl);

  return (
    <span className={`company-logo company-logo--${size}`} aria-hidden="true">
      {showLogo ? (
        <img src={mark.logoUrl} alt="" onError={() => setFailedUrl(mark.logoUrl)} />
      ) : (
        <Building2 size={22} />
      )}
    </span>
  );
}
