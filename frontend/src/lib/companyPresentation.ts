import type { JobSummary } from "../api/models/job";

export interface CompanyPresentation {
  name: string;
  logoUrl?: string;
  websiteUrl?: string;
}

export interface CompanyMark {
  brand: string;
  initials: string;
  logoUrl?: string;
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

const wordmarkCompanies: Record<string, { brand: string; initials: string }> = {
  "kaspi.kz": { brand: "kaspi", initials: "K" },
  kaspi: { brand: "kaspi", initials: "K" },
  "halyk bank": { brand: "halyk", initials: "H" },
  kcell: { brand: "kcell", initials: "K" },
  fortebank: { brand: "forte", initials: "F" },
  "freedom bank": { brand: "freedom", initials: "F" },
  "kolesa group": { brand: "kolesa", initials: "KG" },
  technodom: { brand: "technodom", initials: "T" },
  "bi group": { brand: "bi", initials: "BI" },
  magnum: { brand: "magnum", initials: "M" },
};

/** Prefer bundled official assets so cards do not depend on remote image hosts. */
export function companyMarkFor(name: string, logoUrl?: string): CompanyMark {
  const normalized = name.trim().toLocaleLowerCase();
  const assets: Record<string, string> = {
    "air astana": "air-astana.svg", "kaspi.kz": "kaspi.png", kaspi: "kaspi.png",
    "halyk bank": "halyk-bank.svg", kcell: "kcell.svg", fortebank: "forte.ico",
    "freedom bank": "freedom.svg", "kolesa group": "kolesa-group.svg",
    technodom: "technodom.svg", magnum: "magnum.ico", beeline: "beeline.svg", "beeline kazakhstan": "beeline.svg", kazpost: "kazpost.ico", "қазпошта": "kazpost.ico",
  };
  return { brand: wordmarkCompanies[normalized]?.brand ?? "default", initials: "", logoUrl: assets[normalized] ? `/company-logos/${assets[normalized]}` : logoUrl };
}

export function companyInitials(name: string): string {
  const words = name.trim().split(/\s+/).filter(Boolean);
  return words.slice(0, 2).map((word) => word[0]).join("").toLocaleUpperCase() || "?";
}
