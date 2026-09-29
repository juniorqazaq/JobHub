import { BadgeCheck, Building2 } from "lucide-react";
import { useState } from "react";
import type { Company } from "../../api/models/company";
import { companyInitials } from "../../lib/companyPresentation";

export function CompanyIdentity({ company, size = "card", verifiedLabel }: { company: Company; size?: "card" | "hero"; verifiedLabel: string }) {
  const [failed, setFailed] = useState(false);
  return (
    <div className={`company-identity company-identity--${size}`}>
      <span className="company-identity__logo" aria-hidden="true">
        {company.logoUrl && !failed ? <img src={company.logoUrl} alt="" onError={() => setFailed(true)} /> : companyInitials(company.name) || <Building2 />}
      </span>
      <div>
        <span className="company-identity__name">{company.name}</span>
        {company.verified ? <span className="verified-badge"><BadgeCheck size={16} aria-hidden="true" />{verifiedLabel}</span> : null}
      </div>
    </div>
  );
}
