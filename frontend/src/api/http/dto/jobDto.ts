import type {
  EmploymentType,
  ExperienceLevel,
  SalaryRange,
  WorkMode,
} from "../../models/job";

export interface JobDto {
  id: string;
  title: string;
  company: {
    id?: string | null;
    name: string;
    logo_url?: string;
    verified: boolean;
  };
  location: string;
  city_id?: string;
  work_mode?: WorkMode;
  employment_type?: EmploymentType | string;
  experience_level?: ExperienceLevel;
  category?: string;
  responsibilities?: string;
  requirements?: string;
  nice_to_have?: string;
  skills?: string[];
  salary?: SalaryRange;
  salary_raw?: string;
  posted_at: string;
  first_seen_at?: string;
  last_seen_at?: string;
  last_synced_at?: string;
  external_published_at?: string;
  external_updated_at?: string;
  external_expires_at?: string;
  expires_at?: string;
  published_at?: string;
  created_at?: string;
  updated_at?: string;
  publication_status?: "draft" | "published" | "paused" | "closed";
  moderation_status?: "approved" | "pending" | "rejected";
  salary_min?: number;
  salary_max?: number;
  salary_currency?: string;
  salary_period?: string;
  salary_visible?: boolean;
  benefits?: string[];
  tags?: string[];
  summary: string;
  is_saved?: boolean;
  description_kind?: "full" | "snippet";
  source: {
    id: string;
    name: string;
    type: "native" | "external";
    url?: string;
    upstream_name?: string;
  };
  application: {
    method: "internal" | "external";
    cta_url?: string;
  };
}

export interface EmployerJobsResponseDto { items: JobDto[] }

export interface JobSearchResponseDto {
  items: JobDto[];
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
}
