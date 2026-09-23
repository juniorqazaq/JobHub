import { env } from "../lib/env";
import { HttpJobsRepository } from "./http/HttpJobsRepository";
import { MockJobsRepository } from "./mock/MockJobsRepository";
import type { JobsRepository } from "./repositories/JobsRepository";

export interface Repositories {
  jobs: JobsRepository;
}

export const repositories: Repositories = {
  jobs: env.useMocks ? new MockJobsRepository() : new HttpJobsRepository(),
};
