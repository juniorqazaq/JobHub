import { apiClient } from "../client";
import type { Company, CompanyDetail, CompanyProfileInput, CompanySearchResponse, CompanyVacancy } from "../models/company";
import type { CompaniesRepository } from "../repositories/CompaniesRepository";

type Snake = Record<string, unknown>;
const csrf = (token: string) => ({ headers: { "X-CSRF-Token": token } });

export class HttpCompaniesRepository implements CompaniesRepository {
  async search(query: string, page = 1) {
    const { data } = await apiClient.get<{ items: Snake[]; page: number; page_size: number; total: number; total_pages: number }>("/companies", {
      params: { q: query || undefined, page, page_size: 20 },
    });
    return { items: data.items.map(mapCompany), page: data.page, pageSize: data.page_size, total: data.total, totalPages: data.total_pages } satisfies CompanySearchResponse;
  }

  async get(id: string) {
    const { data } = await apiClient.get<{ company: Snake; jobs: Snake[] }>(`/companies/${id}`);
    return { company: mapCompany(data.company), jobs: data.jobs.map(mapVacancy) } satisfies CompanyDetail;
  }

  async getEmployerCompany() {
    const { data } = await apiClient.get<Snake>("/employer/company");
    return mapCompany(data);
  }

  async updateEmployerCompany(input: CompanyProfileInput, token: string) {
    const { data } = await apiClient.patch<Snake>("/employer/company", payload(input), csrf(token));
    return mapCompany(data);
  }

  async getFollowState(id: string) {
    const { data } = await apiClient.get<{ following: boolean }>(`/companies/${id}/follow`);
    return data.following;
  }

  async follow(id: string, token: string) {
    await apiClient.put(`/companies/${id}/follow`, undefined, csrf(token));
  }

  async unfollow(id: string, token: string) {
    await apiClient.delete(`/companies/${id}/follow`, csrf(token));
  }
}

function mapCompany(data: Snake): Company {
  return {
    id: String(data.id ?? ""),
    name: String(data.name ?? ""),
    description: optional(data.description),
    websiteUrl: optional(data.website_url),
    logoUrl: optional(data.logo_url),
    industry: optional(data.industry),
    city: optional(data.city),
    verified: Boolean(data.is_verified),
    openJobsCount: Number(data.open_jobs_count ?? 0),
    followerCount: Number(data.follower_count ?? 0),
    createdAt: String(data.created_at ?? ""),
    updatedAt: String(data.updated_at ?? ""),
  };
}

function mapVacancy(data: Snake): CompanyVacancy {
  return {
    id: String(data.id ?? ""),
    title: String(data.title ?? ""),
    location: optional(data.location),
    cityId: optional(data.city_id),
    workMode: optional(data.work_mode),
    employmentType: optional(data.employment_type),
    salaryMin: numberOptional(data.salary_min),
    salaryMax: numberOptional(data.salary_max),
    salaryCurrency: optional(data.salary_currency),
    salaryPeriod: optional(data.salary_period),
    salaryVisible: Boolean(data.salary_visible),
    publishedAt: optional(data.published_at),
  };
}

function payload(input: CompanyProfileInput) {
  return { name: input.name, description: input.description, website_url: input.websiteUrl, logo_url: input.logoUrl, industry: input.industry, city: input.city };
}

function optional(value: unknown) { return typeof value === "string" && value ? value : undefined; }
function numberOptional(value: unknown) { return value == null ? undefined : Number(value); }
