import type { JobSearchParams, JobSearchResponse, JobSummary } from "../models/job";
import type { JobsRepository } from "../repositories/JobsRepository";
import { mockJobs } from "../../mocks/data/jobs";

const NETWORK_DELAY_MS = 250;

function wait() {
  return new Promise<void>((resolve) => window.setTimeout(resolve, NETWORK_DELAY_MS));
}

export class MockJobsRepository implements JobsRepository {
  private jobs = mockJobs.map((job) => ({ ...job }));

  async search(params: JobSearchParams): Promise<JobSearchResponse> {
    await wait();
    const query = params.query?.trim().toLocaleLowerCase();
    const location = params.location?.trim().toLocaleLowerCase();
    const page = Math.max(params.page ?? 1, 1);
    const pageSize = Math.max(params.pageSize ?? 10, 1);

    const filtered = this.jobs.filter((job) => {
      const matchesQuery =
        !query ||
        [job.title, job.company.name, ...job.tags].some((value) =>
          value.toLocaleLowerCase().includes(query),
        );
      const matchesLocation = !location || job.location.toLocaleLowerCase().includes(location);
      return matchesQuery && matchesLocation;
    });

    filtered.sort((a, b) => {
      const difference = new Date(b.postedAt).getTime() - new Date(a.postedAt).getTime();
      return params.sort === "oldest" ? -difference : difference;
    });

    const start = (page - 1) * pageSize;
    return {
      items: filtered.slice(start, start + pageSize),
      page,
      pageSize,
      total: filtered.length,
      totalPages: Math.max(Math.ceil(filtered.length / pageSize), 1),
    };
  }

  async getById(id: string): Promise<JobSummary> {
    await wait();
    const job = this.jobs.find((candidate) => candidate.id === id);
    if (!job) throw new Error("JOB_NOT_FOUND");
    return { ...job };
  }

  async save(id: string): Promise<void> {
    await wait();
    this.setSaved(id, true);
  }

  async unsave(id: string): Promise<void> {
    await wait();
    this.setSaved(id, false);
  }

  private setSaved(id: string, isSaved: boolean) {
    const job = this.jobs.find((candidate) => candidate.id === id);
    if (!job) throw new Error("JOB_NOT_FOUND");
    job.isSaved = isSaved;
  }
}
