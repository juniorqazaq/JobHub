import type { EmploymentType, ExperienceLevel, WorkMode } from "./job";

export type SearchStatus =
  | "actively_looking"
  | "open_to_offers"
  | "not_looking";
export type LanguageProficiency =
  | "native"
  | "fluent"
  | "A1"
  | "A2"
  | "B1"
  | "B2"
  | "C1"
  | "C2";
export type ApplicationStatus =
  | "sent"
  | "viewed"
  | "in_review"
  | "contacted"
  | "interview"
  | "offer"
  | "rejected"
  | "withdrawn";

export interface WorkExperience {
  id?: string;
  company: string;
  position: string;
  employmentType: EmploymentType;
  startDate: string;
  endDate?: string;
  isCurrent: boolean;
  description?: string;
  achievements?: string;
  skills: string[];
}

export interface Education {
  id?: string;
  institution: string;
  degree: string;
  fieldOfStudy: string;
  startYear: number;
  graduationYear?: number;
  description?: string;
}

export interface CandidateLanguage {
  id?: string;
  language: string;
  proficiency: LanguageProficiency;
}

export interface ResumeMetadata {
  id: string;
  originalFilename: string;
  contentType: "application/pdf";
  sizeBytes: number;
  uploadedAt: string;
  status: "ready" | "retired" | "deleted";
}

export interface ProfileCompletion {
  percentage: number;
  missing: string[];
}

export interface CandidateProfile {
  userId: string;
  fullName: string;
  photoUrl?: string;
  city?: string;
  birthYear?: number;
  birthDate?: string;
  phone?: string;
  about?: string;
  currentPosition?: string;
  desiredPosition?: string;
  yearsExperience?: number;
  experienceLevel?: Exclude<ExperienceLevel, "no_experience"> | "internship";
  skills: string[];
  certifications: string[];
  languages: CandidateLanguage[];
  desiredSalary?: number;
  currency?: string;
  salaryPeriod?: "month" | "year";
  preferredLocations: string[];
  preferredEmploymentTypes: EmploymentType[];
  preferredWorkModes: WorkMode[];
  preferredCategories: string[];
  preferredRoles: string[];
  searchStatus: SearchStatus;
  github?: string;
  linkedin?: string;
  portfolio?: string;
  website?: string;
  allowEmployerContact: boolean;
  showProfileToEmployers: boolean;
  showSalaryExpectations: boolean;
  workExperience: WorkExperience[];
  education: Education[];
  resume?: ResumeMetadata;
  completion: ProfileCompletion;
}

export type CandidateProfileInput = Omit<
  CandidateProfile,
  "userId" | "resume" | "completion"
>;

export interface SavedJob {
  jobId: string;
  savedAt: string;
  available: boolean;
  title?: string;
  companyName?: string;
  location?: string;
  source?: string;
  sourceName?: string;
  applicationMethod?: "internal" | "external";
}

export interface CandidateApplication {
  id: string;
  jobId: string;
  companyId: string;
  candidateId?: string;
  candidateName?: string;
  candidatePosition?: string;
  jobTitle?: string;
  companyName?: string;
  jobAvailable: boolean;
  status: ApplicationStatus;
  message?: string;
  resume: ResumeMetadata;
  createdAt: string;
  updatedAt: string;
  latestStatusAt: string;
}
