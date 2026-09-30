import type { Company, CompanyProfileInput, CompanySearchResponse, CompanyVacancySearchResponse } from "../models/company";

export interface CompaniesRepository {
  search(query: string, page?: number): Promise<CompanySearchResponse>;
  listFollowing(): Promise<Company[]>;
  get(id: string): Promise<Company>;
  getJobs(id: string, page?: number, pageSize?: number): Promise<CompanyVacancySearchResponse>;
  getEmployerCompany(): Promise<Company>;
  updateEmployerCompany(input: CompanyProfileInput, csrfToken: string): Promise<Company>;
  getFollowState(id: string): Promise<boolean>;
  follow(id: string, csrfToken: string): Promise<void>;
  unfollow(id: string, csrfToken: string): Promise<void>;
}
