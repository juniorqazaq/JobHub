import { env } from "../lib/env";
import { HttpJobsRepository } from "./http/HttpJobsRepository";
import { HttpAuthRepository } from "./http/HttpAuthRepository";
import { HttpEmployerJobsRepository } from "./http/HttpEmployerJobsRepository";
import { MockJobsRepository } from "./mock/MockJobsRepository";
import type { AuthRepository } from "./repositories/AuthRepository";
import type { JobsRepository } from "./repositories/JobsRepository";
import type { EmployerJobsRepository } from "./repositories/EmployerJobsRepository";
import type { CandidateRepository } from "./repositories/CandidateRepository";
import { HttpCandidateRepository } from "./http/HttpCandidateRepository";

export interface Repositories {
  auth: AuthRepository;
  jobs: JobsRepository;
  employerJobs: EmployerJobsRepository;
  candidate: CandidateRepository;
}

export const repositories: Repositories = {
  auth: new HttpAuthRepository(),
  jobs: env.useMocks ? new MockJobsRepository() : new HttpJobsRepository(),
  employerJobs: new HttpEmployerJobsRepository(),
  candidate: new HttpCandidateRepository(),
};
