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
    const page = Math.max(params.page ?? 1, 1);
    const pageSize = Math.max(params.pageSize ?? 10, 1);

    const filtered = this.jobs.filter((job) => {
      const matchesQuery =
        !query ||
        [job.title, job.company.name, ...job.tags].some((value) =>
          value.toLocaleLowerCase().includes(query),
        );
      const matchesCity = !params.city || job.cityId === params.city;
      const matchesMode = !params.workModes?.length || Boolean(job.workMode && params.workModes.includes(job.workMode));
      const matchesSalary = params.salaryMin == null || Boolean(job.salary && job.salary.period === "month" && job.salary.currency === params.currency && (job.salary.max ?? job.salary.min) >= params.salaryMin);
      const matchesExperience = !params.experience || job.experienceLevel === params.experience;
      const matchesEmployment = !params.employment || job.employmentType === params.employment;
      const cutoff = params.datePosted ? Date.now() - ({ "24h": 1, "3d": 3, "7d": 7, "30d": 30 }[params.datePosted] * 86400000) : 0;
      return matchesQuery && matchesCity && matchesMode && matchesSalary && matchesExperience && matchesEmployment && (!cutoff || new Date(job.postedAt).getTime() >= cutoff);
    });

    filtered.sort((a, b) => {
      if (params.preferredCity && a.cityId !== b.cityId) return a.cityId === params.preferredCity ? -1 : b.cityId === params.preferredCity ? 1 : 0;
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
