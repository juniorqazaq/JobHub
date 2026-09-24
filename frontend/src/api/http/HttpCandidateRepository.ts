import type { CandidateRepository } from "../repositories/CandidateRepository";
import type {
  ApplicationStatus,
  CandidateApplication,
  CandidateProfile,
  CandidateProfileInput,
  ResumeMetadata,
  SavedJob,
} from "../models/candidate";
import { apiClient } from "../client";

type Snake = Record<string, unknown>;
const csrf = (token: string) => ({ headers: { "X-CSRF-Token": token } });

export class HttpCandidateRepository implements CandidateRepository {
  async getProfile() {
    const { data } = await apiClient.get<Snake>("/profile");
    return mapProfile(data);
  }
  async updateProfile(input: CandidateProfileInput, token: string) {
    const { data } = await apiClient.patch<Snake>(
      "/profile",
      profilePayload(input),
      csrf(token),
    );
    return mapProfile(data);
  }
  async uploadResume(file: File, token: string) {
    const body = new FormData();
    body.append("file", file);
    const { data } = await apiClient.post<Snake>(
      "/profile/resume",
      body,
      csrf(token),
    );
    return mapResume(data);
  }
  async deleteResume(token: string) {
    await apiClient.delete("/profile/resume", csrf(token));
  }
  async downloadOwnResume() {
    const { data } = await apiClient.get<Blob>("/profile/resume/content", {
      responseType: "blob",
    });
    return data;
  }
  async listSavedJobs() {
    const { data } = await apiClient.get<{ items: Snake[] }>("/saved-jobs");
    return data.items.map(mapSaved);
  }
  async saveJob(jobId: string, token: string) {
    await apiClient.put(`/jobs/${jobId}/saved`, undefined, csrf(token));
  }
  async unsaveJob(jobId: string, token: string) {
    await apiClient.delete(`/jobs/${jobId}/saved`, csrf(token));
  }
  async apply(jobId: string, resumeId: string, message: string, token: string) {
    const { data } = await apiClient.post<Snake>(
      `/jobs/${jobId}/applications`,
      { resume_id: resumeId, message },
      csrf(token),
    );
    return mapApplication(data);
  }
  async listApplications() {
    const { data } = await apiClient.get<{ items: Snake[] }>("/applications");
    return data.items.map(mapApplication);
  }
  async withdraw(applicationId: string, token: string) {
    const { data } = await apiClient.patch<Snake>(
      `/applications/${applicationId}/withdraw`,
      undefined,
      csrf(token),
    );
    return mapApplication(data);
  }
  async listEmployerApplications(jobId: string) {
    const { data } = await apiClient.get<{ items: Snake[] }>(
      `/employer/jobs/${jobId}/applications`,
    );
    return data.items.map(mapApplication);
  }
  async updateApplicationStatus(
    applicationId: string,
    status: Exclude<ApplicationStatus, "sent" | "withdrawn">,
    token: string,
  ) {
    const { data } = await apiClient.patch<Snake>(
      `/employer/applications/${applicationId}`,
      { status },
      csrf(token),
    );
    return mapApplication(data);
  }
  async downloadApplicationResume(applicationId: string) {
    const { data } = await apiClient.get<Blob>(
      `/employer/applications/${applicationId}/resume`,
      { responseType: "blob" },
    );
    return data;
  }
}

function mapProfile(d: Snake): CandidateProfile {
  return {
    userId: String(d.user_id ?? ""),
    fullName: String(d.full_name ?? ""),
    photoUrl: optional(d.photo_url),
    city: optional(d.city),
    birthYear: numberOptional(d.birth_year),
    phone: optional(d.phone),
    about: optional(d.about),
    currentPosition: optional(d.current_position),
    desiredPosition: optional(d.desired_position),
    yearsExperience: numberOptional(d.years_experience),
    experienceLevel: optional(
      d.experience_level,
    ) as CandidateProfile["experienceLevel"],
    skills: strings(d.skills),
    certifications: strings(d.certifications),
    languages: array(d.languages).map((item) => ({
      id: optional(item.id),
      language: String(item.language ?? ""),
      proficiency: String(
        item.proficiency ?? "B1",
      ) as CandidateProfile["languages"][number]["proficiency"],
    })),
    desiredSalary: numberOptional(d.desired_salary),
    currency: optional(d.currency),
    salaryPeriod: optional(d.salary_period) as CandidateProfile["salaryPeriod"],
    preferredLocations: strings(d.preferred_locations),
    preferredEmploymentTypes: strings(
      d.preferred_employment_types,
    ) as CandidateProfile["preferredEmploymentTypes"],
    preferredWorkModes: strings(
      d.preferred_work_modes,
    ) as CandidateProfile["preferredWorkModes"],
    preferredCategories: strings(d.preferred_categories),
    preferredRoles: strings(d.preferred_roles),
    searchStatus: String(
      d.search_status ?? "actively_looking",
    ) as CandidateProfile["searchStatus"],
    github: optional(d.github),
    linkedin: optional(d.linkedin),
    portfolio: optional(d.portfolio),
    website: optional(d.website),
    allowEmployerContact: Boolean(d.allow_employer_contact),
    showProfileToEmployers: Boolean(d.show_profile_to_employers),
    showSalaryExpectations: Boolean(d.show_salary_expectations),
    workExperience: array(d.work_experience).map((item) => ({
      id: optional(item.id),
      company: String(item.company ?? ""),
      position: String(item.position ?? ""),
      employmentType: String(
        item.employment_type ?? "full_time",
      ) as CandidateProfile["workExperience"][number]["employmentType"],
      startDate: String(item.start_date ?? ""),
      endDate: optional(item.end_date),
      isCurrent: Boolean(item.is_current),
      description: optional(item.description),
      achievements: optional(item.achievements),
      skills: strings(item.skills),
    })),
    education: array(d.education).map((item) => ({
      id: optional(item.id),
      institution: String(item.institution ?? ""),
      degree: String(item.degree ?? ""),
      fieldOfStudy: String(item.field_of_study ?? ""),
      startYear: Number(item.start_year ?? 0),
      graduationYear: numberOptional(item.graduation_year),
      description: optional(item.description),
    })),
    resume:
      d.resume && typeof d.resume === "object"
        ? mapResume(d.resume as Snake)
        : undefined,
    completion: {
      percentage: Number((d.completion as Snake | undefined)?.percentage ?? 0),
      missing: strings((d.completion as Snake | undefined)?.missing),
    },
  };
}

function profilePayload(p: CandidateProfileInput) {
  return {
    full_name: p.fullName,
    photo_url: p.photoUrl,
    city: p.city,
    birth_year: p.birthYear,
    phone: p.phone,
    about: p.about,
    current_position: p.currentPosition,
    desired_position: p.desiredPosition,
    years_experience: p.yearsExperience,
    experience_level: p.experienceLevel,
    skills: p.skills,
    certifications: p.certifications,
    languages: p.languages.map((x) => ({
      id: x.id,
      language: x.language,
      proficiency: x.proficiency,
    })),
    desired_salary: p.desiredSalary,
    currency: p.currency,
    salary_period: p.salaryPeriod,
    preferred_locations: p.preferredLocations,
    preferred_employment_types: p.preferredEmploymentTypes,
    preferred_work_modes: p.preferredWorkModes,
    preferred_categories: p.preferredCategories,
    preferred_roles: p.preferredRoles,
    search_status: p.searchStatus,
    github: p.github,
    linkedin: p.linkedin,
    portfolio: p.portfolio,
    website: p.website,
    allow_employer_contact: p.allowEmployerContact,
    show_profile_to_employers: p.showProfileToEmployers,
    show_salary_expectations: p.showSalaryExpectations,
    work_experience: p.workExperience.map((x) => ({
      id: x.id,
      company: x.company,
      position: x.position,
      employment_type: x.employmentType,
      start_date: x.startDate,
      end_date: x.endDate,
      is_current: x.isCurrent,
      description: x.description,
      achievements: x.achievements,
      skills: x.skills,
    })),
    education: p.education.map((x) => ({
      id: x.id,
      institution: x.institution,
      degree: x.degree,
      field_of_study: x.fieldOfStudy,
      start_year: x.startYear,
      graduation_year: x.graduationYear,
      description: x.description,
    })),
  };
}
function mapResume(d: Snake): ResumeMetadata {
  return {
    id: String(d.id ?? ""),
    originalFilename: String(d.original_filename ?? ""),
    contentType: "application/pdf",
    sizeBytes: Number(d.size_bytes ?? 0),
    uploadedAt: String(d.uploaded_at ?? ""),
    status: String(d.status ?? "ready") as ResumeMetadata["status"],
  };
}
function mapSaved(d: Snake): SavedJob {
  return {
    jobId: String(d.job_id ?? ""),
    savedAt: String(d.saved_at ?? ""),
    available: Boolean(d.available),
    title: optional(d.title),
    companyName: optional(d.company_name),
    location: optional(d.location),
    source: optional(d.source),
    sourceName: optional(d.source_name),
    applicationMethod: optional(
      d.application_method,
    ) as SavedJob["applicationMethod"],
  };
}
function mapApplication(d: Snake): CandidateApplication {
  return {
    id: String(d.id ?? ""),
    jobId: String(d.job_id ?? ""),
    companyId: String(d.company_id ?? ""),
    candidateId: optional(d.candidate_id),
    candidateName: optional(d.candidate_name),
    candidatePosition: optional(d.candidate_position),
    jobTitle: optional(d.job_title),
    companyName: optional(d.company_name),
    jobAvailable: Boolean(d.job_available),
    status: String(d.status ?? "sent") as ApplicationStatus,
    message: optional(d.message),
    resume: mapResume((d.resume as Snake) ?? {}),
    createdAt: String(d.created_at ?? ""),
    updatedAt: String(d.updated_at ?? ""),
    latestStatusAt: String(d.latest_status_at ?? ""),
  };
}
function optional(v: unknown) {
  return typeof v === "string" && v ? v : undefined;
}
function numberOptional(v: unknown) {
  return typeof v === "number" ? v : v == null ? undefined : Number(v);
}
function strings(v: unknown) {
  return Array.isArray(v) ? v.map(String) : [];
}
function array(v: unknown): Snake[] {
  return Array.isArray(v)
    ? v.filter((x): x is Snake => Boolean(x) && typeof x === "object")
    : [];
}
