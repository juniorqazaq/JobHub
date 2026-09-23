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
  work_mode?: WorkMode;
  employment_type?: EmploymentType | string;
  experience_level?: ExperienceLevel;
  salary?: SalaryRange;
  salary_raw?: string;
  posted_at: string;
  first_seen_at?: string;
  last_seen_at?: string;
  last_synced_at?: string;
  external_published_at?: string;
  external_updated_at?: string;
  external_expires_at?: string;
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

export interface JobSearchResponseDto {
  items: JobDto[];
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
}
