import { BadgeCheck, Building2 } from "lucide-react";
import { useState } from "react";
import type { Company } from "../../api/models/company";
import { companyMarkFor } from "../../lib/companyPresentation";

export function CompanyIdentity({ company, size = "card", verifiedLabel }: { company: Company; size?: "card" | "hero"; verifiedLabel: string }) {
  const [failedUrl, setFailedUrl] = useState<string>();
  const mark = companyMarkFor(company.name, company.logoUrl);
  return (
    <div className={`company-identity company-identity--${size}`}>
      <span className="company-identity__logo" aria-hidden="true">
        {mark.logoUrl && failedUrl !== mark.logoUrl ? <img src={mark.logoUrl} alt="" onError={() => setFailedUrl(mark.logoUrl)} /> : <Building2 size={24} />}
      </span>
      <div>
        {size === "hero" ? <h1 className="company-identity__name">{company.name}</h1> : <span className="company-identity__name">{company.name}</span>}
        {company.verified ? <span className="verified-badge"><BadgeCheck size={16} aria-hidden="true" />{verifiedLabel}</span> : null}
      </div>
    </div>
  );
}
