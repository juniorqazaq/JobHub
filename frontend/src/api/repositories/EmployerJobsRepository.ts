import type { JobSummary, NativeJobInput, PublicationStatus } from "../models/job";

export interface EmployerJobsRepository {
  list(): Promise<JobSummary[]>;
  getById(id: string): Promise<JobSummary>;
  create(input: NativeJobInput, csrfToken: string): Promise<JobSummary>;
  update(id: string, input: NativeJobInput, csrfToken: string): Promise<JobSummary>;
  transition(id: string, status: PublicationStatus, csrfToken: string): Promise<JobSummary>;
  delete(id: string, csrfToken: string): Promise<void>;
}
