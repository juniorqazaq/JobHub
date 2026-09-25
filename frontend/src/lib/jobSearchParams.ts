import type { EmploymentType, ExperienceLevel, JobSearchParams, JobSort, WorkMode } from "../api/models/job";
import { isCityId } from "./cities";

const workModes: WorkMode[] = ["on_site", "hybrid", "remote"];
const experiences: ExperienceLevel[] = ["no_experience", "junior", "middle", "senior", "lead"];
const employments: EmploymentType[] = ["full_time", "part_time", "contract", "temporary", "internship"];
const dates = ["24h", "3d", "7d", "30d"] as const;

function nonnegativeNumber(value: string | null): number | undefined {
  if (!value) return undefined;
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : undefined;
}

function positiveInteger(value: string | null): number | undefined {
  if (!value) return undefined;
  const parsed = Number(value);
  return Number.isInteger(parsed) && parsed > 0 ? parsed : undefined;
}

export function parseJobSearchParams(params: URLSearchParams): JobSearchParams {
  const rawSort = params.get("sort");
  const rawCity = params.get("city");
  const rawWorkModes = (params.get("work_mode") ?? "").split(",").filter((value): value is WorkMode => workModes.includes(value as WorkMode));
  const experience = params.get("experience");
  const employment = params.get("employment");
  const datePosted = params.get("date_posted");
  const salaryMin = nonnegativeNumber(params.get("salary_min"));
  const currency = params.get("currency")?.toUpperCase();
  const sort: JobSort = rawSort === "oldest" ? "oldest" : "newest";
  return {
    query: params.get("q") || undefined,
    city: isCityId(rawCity) ? rawCity : undefined,
    workModes: [...new Set(rawWorkModes)],
    salaryMin,
    currency: salaryMin != null && (currency === "KZT" || currency === "USD" || currency === "EUR") ? currency : undefined,
    experience: experiences.includes(experience as ExperienceLevel) ? experience as ExperienceLevel : undefined,
    employment: employments.includes(employment as EmploymentType) ? employment as EmploymentType : undefined,
    datePosted: dates.includes(datePosted as (typeof dates)[number]) ? datePosted as JobSearchParams["datePosted"] : undefined,
    sort,
    page: positiveInteger(params.get("page")) ?? 1,
  };
}

export function serializeJobSearchParams(search: JobSearchParams): URLSearchParams {
  const params = new URLSearchParams();
  if (search.query) params.set("q", search.query);
  if (search.city && isCityId(search.city)) params.set("city", search.city);
  const modes = workModes.filter((mode) => search.workModes?.includes(mode));
  if (modes.length) params.set("work_mode", modes.join(","));
  if (search.salaryMin != null && search.salaryMin >= 0 && search.currency) {
    params.set("salary_min", String(search.salaryMin));
    params.set("currency", search.currency);
  }
  if (search.experience) params.set("experience", search.experience);
  if (search.employment) params.set("employment", search.employment);
  if (search.datePosted) params.set("date_posted", search.datePosted);
  if (search.sort && search.sort !== "newest") params.set("sort", search.sort);
  if (search.page && search.page > 1) params.set("page", String(search.page));
  return params;
}
