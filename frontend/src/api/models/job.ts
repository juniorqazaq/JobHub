export type WorkMode = "remote" | "hybrid" | "on_site";
export type EmploymentType = "full_time" | "part_time" | "contract" | "temporary" | "internship";
export type ExperienceLevel = "no_experience" | "junior" | "middle" | "senior" | "lead";
export type PublicationStatus = "draft" | "published" | "paused" | "closed";
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
  category?: string;
  responsibilities?: string;
  requirements?: string;
  niceToHave?: string;
  salary?: SalaryRange;
  salaryRaw?: string;
  postedAt: string;
  firstSeenAt?: string;
  lastSeenAt?: string;
  lastSyncedAt?: string;
  externalPublishedAt?: string;
  externalUpdatedAt?: string;
  externalExpiresAt?: string;
  expiresAt?: string;
  publishedAt?: string;
  createdAt?: string;
  updatedAt?: string;
  publicationStatus?: PublicationStatus;
  moderationStatus?: "approved" | "pending" | "rejected";
  benefits?: string[];
  tags: string[];
  summary: string;
  isSaved: boolean;
  descriptionKind?: "full" | "snippet";
  source: JobSource;
  application: JobApplication;
}

export interface NativeJobInput {
  title: string;
  category: string;
  description: string;
  responsibilities: string;
  requirements: string;
  niceToHave?: string;
  skills: string[];
  location: string;
  workMode: WorkMode;
  employmentType: EmploymentType;
  experienceLevel: ExperienceLevel;
  salaryMin?: number;
  salaryMax?: number;
  salaryCurrency?: string;
  salaryPeriod?: string;
  salaryVisible: boolean;
  benefits: string[];
  expiresAt?: string;
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
