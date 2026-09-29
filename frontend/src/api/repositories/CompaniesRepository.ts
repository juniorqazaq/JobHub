import type { Company, CompanyDetail, CompanyProfileInput, CompanySearchResponse } from "../models/company";

export interface CompaniesRepository {
  search(query: string, page?: number): Promise<CompanySearchResponse>;
  get(id: string): Promise<CompanyDetail>;
  getEmployerCompany(): Promise<Company>;
  updateEmployerCompany(input: CompanyProfileInput, csrfToken: string): Promise<Company>;
  getFollowState(id: string): Promise<boolean>;
  follow(id: string, csrfToken: string): Promise<void>;
  unfollow(id: string, csrfToken: string): Promise<void>;
}
