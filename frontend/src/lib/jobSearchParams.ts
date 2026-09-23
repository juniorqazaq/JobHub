import type { JobSearchParams, JobSort } from "../api/models/job";

function positiveNumber(value: string | null): number | undefined {
  if (!value) return undefined;
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined;
}

export function parseJobSearchParams(params: URLSearchParams): JobSearchParams {
  const rawSort = params.get("sort");
  const sort: JobSort = rawSort === "oldest" ? "oldest" : "newest";
  return {
    query: params.get("q") || undefined,
    location: params.get("location") || undefined,
    sort,
    page: positiveNumber(params.get("page")) ?? 1,
  };
}

export function serializeJobSearchParams(search: JobSearchParams): URLSearchParams {
  const params = new URLSearchParams();
  if (search.query) params.set("q", search.query);
  if (search.location) params.set("location", search.location);
  if (search.sort && search.sort !== "newest") params.set("sort", search.sort);
  if (search.page && search.page > 1) params.set("page", String(search.page));
  return params;
}
