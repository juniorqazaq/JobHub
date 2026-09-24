import type { JobsRepository } from "../repositories/JobsRepository";
import type { JobSearchParams, JobSearchResponse, JobSummary } from "../models/job";
import { apiClient } from "../client";
import type { JobDto, JobSearchResponseDto } from "./dto/jobDto";
import { mapJobDto } from "./jobMappers";

export class HttpJobsRepository implements JobsRepository {
  async search(params: JobSearchParams): Promise<JobSearchResponse> {
    const response = await apiClient.get<JobSearchResponseDto>("/jobs", {
      params: {
        q: params.query,
        city: params.city,
        preferred_city: params.preferredCity,
        work_mode: params.workModes?.join(","),
        salary_min: params.salaryMin,
        currency: params.currency,
        experience: params.experience,
        employment: params.employment,
        date_posted: params.datePosted,
        sort: params.sort,
        page: params.page,
        page_size: params.pageSize,
      },
    });

    return {
      items: response.data.items.map(mapJobDto),
      page: response.data.page,
      pageSize: response.data.page_size,
      total: response.data.total,
      totalPages: response.data.total_pages,
    };
  }

  async getById(id: string): Promise<JobSummary> {
    const response = await apiClient.get<JobDto>(`/jobs/${id}`);
    return mapJobDto(response.data);
  }

  async save(id: string): Promise<void> {
    await apiClient.put(`/jobs/${id}/saved`);
  }

  async unsave(id: string): Promise<void> {
    await apiClient.delete(`/jobs/${id}/saved`);
  }
}
