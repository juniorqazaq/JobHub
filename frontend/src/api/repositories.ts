import { env } from "../lib/env";
import { HttpJobsRepository } from "./http/HttpJobsRepository";
import { HttpAuthRepository } from "./http/HttpAuthRepository";
import { MockJobsRepository } from "./mock/MockJobsRepository";
import type { AuthRepository } from "./repositories/AuthRepository";
import type { JobsRepository } from "./repositories/JobsRepository";

export interface Repositories {
  auth: AuthRepository;
  jobs: JobsRepository;
}

export const repositories: Repositories = {
  auth: new HttpAuthRepository(),
  jobs: env.useMocks ? new MockJobsRepository() : new HttpJobsRepository(),
};
