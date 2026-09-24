import type {
  ApplicationStatus,
  CandidateApplication,
  CandidateProfile,
  CandidateProfileInput,
  ResumeMetadata,
  SavedJob,
} from "../models/candidate";

export interface CandidateRepository {
  getProfile(): Promise<CandidateProfile>;
  updateProfile(
    input: CandidateProfileInput,
    csrfToken: string,
  ): Promise<CandidateProfile>;
  uploadResume(file: File, csrfToken: string): Promise<ResumeMetadata>;
  deleteResume(csrfToken: string): Promise<void>;
  downloadOwnResume(): Promise<Blob>;
  listSavedJobs(): Promise<SavedJob[]>;
  saveJob(jobId: string, csrfToken: string): Promise<void>;
  unsaveJob(jobId: string, csrfToken: string): Promise<void>;
  apply(
    jobId: string,
    resumeId: string,
    message: string,
    csrfToken: string,
  ): Promise<CandidateApplication>;
  listApplications(): Promise<CandidateApplication[]>;
  withdraw(
    applicationId: string,
    csrfToken: string,
  ): Promise<CandidateApplication>;
  listEmployerApplications(jobId: string): Promise<CandidateApplication[]>;
  updateApplicationStatus(
    applicationId: string,
    status: Exclude<ApplicationStatus, "sent" | "withdrawn">,
    csrfToken: string,
  ): Promise<CandidateApplication>;
  downloadApplicationResume(applicationId: string): Promise<Blob>;
}
