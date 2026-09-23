import type {
  JobSearchParams,
  JobSearchResponse,
  JobSummary,
} from "../models/job";

export interface JobsRepository {
  search(params: JobSearchParams): Promise<JobSearchResponse>;
  getById(id: string): Promise<JobSummary>;
  save(id: string): Promise<void>;
  unsave(id: string): Promise<void>;
}
