export type WorkMode = "remote" | "hybrid" | "on_site";
export type EmploymentType = "full_time" | "part_time" | "contract" | "internship";
export type ExperienceLevel = "internship" | "junior" | "middle" | "senior" | "lead";
export type JobSort = "newest" | "oldest";

export interface SalaryRange {
  min: number;
  max?: number;
  currency: "KZT" | "USD";
  period: "month" | "year";
}

export interface CompanySummary {
  id?: string;
  name: string;
  logoUrl?: string;
  verified: boolean;
}

export interface JobSource {
  id: string;
  name: string;
  type: "native" | "external";
  url?: string;
  upstreamName?: string;
}

export interface JobApplication {
  method: "internal" | "external";
  ctaUrl?: string;
}

export interface JobSummary {
  id: string;
  title: string;
  company: CompanySummary;
  location: string;
  workMode?: WorkMode;
  employmentType?: EmploymentType | string;
  experienceLevel?: ExperienceLevel;
  salary?: SalaryRange;
  salaryRaw?: string;
  postedAt: string;
  firstSeenAt?: string;
  lastSeenAt?: string;
  lastSyncedAt?: string;
  externalPublishedAt?: string;
  externalUpdatedAt?: string;
  externalExpiresAt?: string;
  tags: string[];
  summary: string;
  isSaved: boolean;
  descriptionKind?: "full" | "snippet";
  source: JobSource;
  application: JobApplication;
}

export interface JobSearchParams {
  query?: string;
  location?: string;
  sort?: JobSort;
  page?: number;
  pageSize?: number;
}

export interface JobSearchResponse {
  items: JobSummary[];
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
}
