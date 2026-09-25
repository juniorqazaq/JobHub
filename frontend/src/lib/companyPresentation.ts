import type { JobSummary } from "../api/models/job";

export interface CompanyPresentation {
  name: string;
  logoUrl?: string;
  websiteUrl?: string;
}

const sourcePresentations: Record<string, CompanyPresentation> = {
  "kcell:careers": {
    name: "Kcell",
    logoUrl: "/company-logos/kcell.svg",
    websiteUrl: "https://kcell.kz",
  },
  "airastana:careers": {
    name: "Air Astana",
    logoUrl: "/company-logos/air-astana.svg",
    websiteUrl: "https://airastana.com",
  },
  "halyk:careers": {
    name: "Halyk Bank",
    logoUrl: "/company-logos/halyk-bank.svg",
    websiteUrl: "https://halykbank.kz",
  },
  "technodom:careers": {
    name: "Technodom",
    logoUrl: "/company-logos/technodom.svg",
    websiteUrl: "https://www.technodom.kz",
  },
  "kolesa:careers": {
    name: "Kolesa Group",
    logoUrl: "/company-logos/kolesa-group.svg",
    websiteUrl: "https://kolesa.group",
  },
};

export function companyPresentationFor(job: JobSummary): CompanyPresentation {
  return sourcePresentations[job.source.id] ?? {
    name: job.company.name,
    logoUrl: job.company.logoUrl,
  };
}

export function companyInitials(name: string): string {
  const words = name.trim().split(/\s+/).filter(Boolean);
  return words.slice(0, 2).map((word) => word[0]).join("").toLocaleUpperCase() || "?";
}
