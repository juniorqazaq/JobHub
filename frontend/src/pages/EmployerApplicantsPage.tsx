import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Download, FileText, UserRound } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link, useParams } from "react-router-dom";
import type { ApplicationStatus } from "../api/models/candidate";
import { repositories } from "../api/repositories";
import { useAuth } from "../app/authContext";
import { EmployerWorkspaceNav } from "../components/employer/EmployerWorkspaceNav";
import { Button } from "../components/ui/Button";
import {
  EmptyState,
  ErrorState,
  JobListSkeleton,
} from "../components/ui/Feedback";
import { useToast } from "../components/ui/useToast";
import { SiteShell } from "../layouts/SiteShell";

const actions: Exclude<ApplicationStatus, "sent" | "withdrawn">[] = [
  "viewed",
  "in_review",
  "contacted",
  "interview",
  "offer",
  "rejected",
];
export function EmployerApplicantsPage() {
  const { jobId = "" } = useParams();
  const { t, i18n } = useTranslation();
  const { session } = useAuth();
  const client = useQueryClient();
  const toast = useToast();
  const apps = useQuery({
    queryKey: ["employer", "applications", jobId],
    queryFn: () => repositories.candidate.listEmployerApplications(jobId),
    enabled: Boolean(jobId),
  });
  const update = useMutation({
    mutationFn: ({
      id,
      status,
    }: {
      id: string;
      status: Exclude<ApplicationStatus, "sent" | "withdrawn">;
    }) =>
      repositories.candidate.updateApplicationStatus(
        id,
        status,
        session!.csrfToken,
      ),
    onSuccess: async () => {
      await client.invalidateQueries({
        queryKey: ["employer", "applications", jobId],
      });
      toast.showToast({ title: t("employerApplicants.updated") });
    },
    onError: () =>
      toast.showToast({
        tone: "error",
        title: t("employerApplicants.updateError"),
      }),
  });
  const download = async (id: string, name: string) => {
    try {
      const blob = await repositories.candidate.downloadApplicationResume(id);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = name;
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      toast.showToast({
        tone: "error",
        title: t("employerApplicants.resumeError"),
      });
    }
  };
  return (
    <SiteShell>
      <main className="employer-page page-container">
        <EmployerWorkspaceNav />
        <header className="employer-list-header">
          <div>
            <p className="auth-eyebrow">{t("employerApplicants.eyebrow")}</p>
            <h1>{t("employerApplicants.title")}</h1>
            <p>{t("employerApplicants.description")}</p>
          </div>
          <Link className="text-link" to="/employer/vacancies">
            {t("employerApplicants.back")}
          </Link>
        </header>
        {apps.isPending ? (
          <JobListSkeleton label={t("employerApplicants.loading")} />
        ) : null}
        {apps.isError ? (
          <ErrorState
            title={t("employerApplicants.loadErrorTitle")}
            description={t("employerApplicants.loadErrorDescription")}
            actionLabel={t("common.retry")}
            onAction={() => void apps.refetch()}
          />
        ) : null}
        {apps.data?.length === 0 ? (
          <EmptyState
            title={t("employerApplicants.emptyTitle")}
            description={t("employerApplicants.emptyDescription")}
          />
        ) : null}
        {apps.data?.length ? (
          <div className="applicant-list">
            {apps.data.map((app) => (
              <article className="applicant-row" key={app.id}>
                <div className="applicant-row__identity">
                  <span>
                    <UserRound size={20} />
                  </span>
                  <div>
                    <h2>{app.candidateName}</h2>
                    <p>{app.candidatePosition || t("common.notProvided")}</p>
                    <small>{formatDate(app.createdAt, i18n.language)}</small>
                  </div>
                </div>
                <div className="applicant-row__message">
                  <strong>{t("employerApplicants.message")}</strong>
                  <p>{app.message || t("employerApplicants.noMessage")}</p>
                  <Button
                    size="sm"
                    variant="secondary"
                    leadingIcon={<Download size={15} />}
                    onClick={() =>
                      void download(app.id, app.resume.originalFilename)
                    }
                  >
                    <FileText size={15} />
                    {app.resume.originalFilename}
                  </Button>
                </div>
                <div className="applicant-row__status">
                  <span className={`application-status status-${app.status}`}>
                    {t(`applications.statuses.${app.status}`)}
                  </span>
                  <label>
                    <span>{t("employerApplicants.changeStatus")}</span>
                    <select
                      value={app.status}
                      disabled={
                        app.status === "withdrawn" ||
                        app.status === "rejected" ||
                        update.isPending
                      }
                      onChange={(event) =>
                        update.mutate({
                          id: app.id,
                          status: event.target.value as Exclude<
                            ApplicationStatus,
                            "sent" | "withdrawn"
                          >,
                        })
                      }
                    >
                      <option value={app.status}>
                        {t(`applications.statuses.${app.status}`)}
                      </option>
                      {nextStatuses(app.status).map((status) => (
                        <option value={status} key={status}>
                          {t(`applications.statuses.${status}`)}
                        </option>
                      ))}
                    </select>
                  </label>
                </div>
              </article>
            ))}
          </div>
        ) : null}
      </main>
    </SiteShell>
  );
}
function formatDate(value: string, language: string) {
  return new Intl.DateTimeFormat(language, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

function nextStatuses(current: ApplicationStatus) {
  if (current === "rejected" || current === "withdrawn") return [];
  const rank: Partial<Record<ApplicationStatus, number>> = {
    sent: 0,
    viewed: 1,
    in_review: 2,
    contacted: 3,
    interview: 4,
    offer: 5,
  };
  return actions.filter(
    (status) =>
      status === "rejected" ||
      (rank[status] ?? -1) > (rank[current] ?? Number.MAX_SAFE_INTEGER),
  );
}
