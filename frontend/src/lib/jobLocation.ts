import type { JobSummary } from "../api/models/job";
import { cityName } from "./cities";

export function displayJobLocation(job: Pick<JobSummary, "cityId" | "location">, language: string) {
  return cityName(job.cityId, language) || job.location;
}
