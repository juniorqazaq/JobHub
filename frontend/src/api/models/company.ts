export interface Company {
  id: string;
  name: string;
  description?: string;
  websiteUrl?: string;
  logoUrl?: string;
  industry?: string;
  city?: string;
  verified: boolean;
  openJobsCount: number;
  followerCount: number;
  createdAt: string;
  updatedAt: string;
}

export interface CompanyVacancy {
  id: string;
  title: string;
  location?: string;
  cityId?: string;
  workMode?: string;
  employmentType?: string;
  salaryMin?: number;
  salaryMax?: number;
  salaryCurrency?: string;
  salaryPeriod?: string;
  salaryVisible: boolean;
  publishedAt?: string;
  source: "native" | "external";
  sourceName: string;
}

export interface CompanySearchResponse {
  items: Company[];
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
}

export interface CompanyVacancySearchResponse {
  items: CompanyVacancy[];
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
}

export interface CompanyProfileInput {
  name: string;
  description: string;
  websiteUrl: string;
  logoUrl: string;
  industry: string;
  city: string;
}
