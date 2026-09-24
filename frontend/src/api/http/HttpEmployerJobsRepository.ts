import type { NativeJobInput, PublicationStatus } from "../models/job";
import type { EmployerJobsRepository } from "../repositories/EmployerJobsRepository";
import { apiClient } from "../client";
import { mapJobDto } from "./jobMappers";
import type { EmployerJobsResponseDto, JobDto } from "./dto/jobDto";

export class HttpEmployerJobsRepository implements EmployerJobsRepository {
  async list() {
    const { data } = await apiClient.get<EmployerJobsResponseDto>("/employer/jobs");
    return data.items.map(mapJobDto);
  }

  async getById(id: string) {
    const { data } = await apiClient.get<JobDto>(`/employer/jobs/${id}`);
    return mapJobDto(data);
  }

  async create(input: NativeJobInput, csrfToken: string) {
    const { data } = await apiClient.post<JobDto>("/jobs", toRequest(input), { headers: csrf(csrfToken) });
    return mapJobDto(data);
  }

  async update(id: string, input: NativeJobInput, csrfToken: string) {
    const { data } = await apiClient.patch<JobDto>(`/jobs/${id}`, toRequest(input), { headers: csrf(csrfToken) });
    return mapJobDto(data);
  }

  async transition(id: string, status: PublicationStatus, csrfToken: string) {
    const { data } = await apiClient.patch<JobDto>(`/jobs/${id}`, { publication_status: status }, { headers: csrf(csrfToken) });
    return mapJobDto(data);
  }

  async delete(id: string, csrfToken: string) {
    await apiClient.delete(`/jobs/${id}`, { headers: csrf(csrfToken) });
  }
}

function csrf(token: string) { return { "X-CSRF-Token": token }; }
function toRequest(input: NativeJobInput) {
  return {
    title: input.title, category: input.category, description: input.description,
    responsibilities: input.responsibilities, requirements: input.requirements,
    nice_to_have: input.niceToHave || undefined, skills: input.skills, city_id: input.cityId,
    work_mode: input.workMode, employment_type: input.employmentType,
    experience_level: input.experienceLevel, salary_min: input.salaryMin,
    salary_max: input.salaryMax, salary_currency: input.salaryCurrency || undefined,
    salary_period: input.salaryPeriod || undefined, salary_visible: input.salaryVisible,
    benefits: input.benefits, expires_at: input.expiresAt || undefined,
  };
}
