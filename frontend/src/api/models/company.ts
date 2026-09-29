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
}

export interface CompanyDetail {
  company: Company;
  jobs: CompanyVacancy[];
}

export interface CompanySearchResponse {
  items: Company[];
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
