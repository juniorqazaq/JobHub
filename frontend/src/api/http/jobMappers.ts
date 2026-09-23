import type { JobSummary } from "../models/job";
import type { JobDto } from "./dto/jobDto";

export function mapJobDto(dto: JobDto): JobSummary {
  return {
    id: dto.id,
    title: dto.title,
    company: {
      id: dto.company.id ?? undefined,
      name: dto.company.name,
      logoUrl: dto.company.logo_url,
      verified: dto.company.verified,
    },
    location: dto.location,
    workMode: dto.work_mode,
    employmentType: dto.employment_type,
    experienceLevel: dto.experience_level,
    salary: dto.salary,
    salaryRaw: dto.salary_raw,
    postedAt: dto.posted_at,
    firstSeenAt: dto.first_seen_at,
    lastSeenAt: dto.last_seen_at,
    lastSyncedAt: dto.last_synced_at,
    externalPublishedAt: dto.external_published_at,
    externalUpdatedAt: dto.external_updated_at,
    externalExpiresAt: dto.external_expires_at,
    tags: dto.tags ?? [],
    summary: dto.summary,
    isSaved: dto.is_saved ?? false,
    descriptionKind: dto.description_kind,
    source: {
      id: dto.source.id,
      name: dto.source.name,
      type: dto.source.type,
      url: dto.source.url,
      upstreamName: dto.source.upstream_name,
    },
    application: {
      method: dto.application.method,
      ctaUrl: dto.application.cta_url,
    },
  };
}
