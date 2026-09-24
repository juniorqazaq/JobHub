import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { BriefcaseBusiness, CalendarDays, ChevronRight } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { repositories } from "../api/repositories";
import { useAuth } from "../app/authContext";
import { Button } from "../components/ui/Button";
import {
  EmptyState,
  ErrorState,
  JobListSkeleton,
} from "../components/ui/Feedback";
import { useToast } from "../components/ui/useToast";
import { CandidateJobsLayout } from "../components/candidate/CandidateJobsLayout";
import type { ApplicationStatus } from "../api/models/candidate";

export function ApplicationsPage() {
  const { t, i18n } = useTranslation();
  const { session } = useAuth();
  const client = useQueryClient();
  const toast = useToast();
  const [status, setStatus] = useState<"all" | ApplicationStatus>("all");
  const apps = useQuery({
    queryKey: ["applications"],
    queryFn: () => repositories.candidate.listApplications(),
  });
  const withdraw = useMutation({
    mutationFn: (id: string) =>
      repositories.candidate.withdraw(id, session!.csrfToken),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ["applications"] });
      toast.showToast({ title: t("applications.withdrawn") });
    },
    onError: () =>
      toast.showToast({
        tone: "error",
        title: t("applications.withdrawError"),
      }),
  });
  const visibleApplications =
    status === "all"
      ? apps.data
      : apps.data?.filter((application) => application.status === status);
  return (
    <CandidateJobsLayout tabTitle={t("applications.title")} tabDescription={t("applications.description")}>
        {apps.data?.length ? (
          <label className="application-filter">
            <span>{t("applications.filterLabel")}</span>
            <select
              value={status}
              onChange={(event) =>
                setStatus(event.target.value as "all" | ApplicationStatus)
              }
            >
              <option value="all">{t("applications.filterAll")}</option>
              {applicationStatuses.map((value) => (
                <option key={value} value={value}>
                  {t(`applications.statuses.${value}`)}
                </option>
              ))}
            </select>
          </label>
        ) : null}
        {apps.isPending ? (
          <JobListSkeleton label={t("applications.loading")} />
        ) : null}
        {apps.isError ? (
          <ErrorState
            title={t("applications.loadErrorTitle")}
            description={t("applications.loadErrorDescription")}
            actionLabel={t("common.retry")}
            onAction={() => void apps.refetch()}
          />
        ) : null}
        {apps.data?.length === 0 ? (
          <EmptyState
            title={t("applications.emptyTitle")}
            description={t("applications.emptyDescription")}
            secondary={
              <Link className="text-link" to="/jobs">
                {t("applications.browse")}
              </Link>
            }
          />
        ) : null}
        {visibleApplications?.length ? (
          <div className="application-list">
            {visibleApplications.map((app) => (
              <article className="application-row" key={app.id}>
                <div className="application-row__icon">
                  <BriefcaseBusiness size={21} />
                </div>
                <div className="application-row__main">
                  <div>
                    <h2>
                      {app.jobAvailable
                        ? app.jobTitle
                        : t("applications.unavailableJob")}
                    </h2>
                    <span className={`application-status status-${app.status}`}>
                      {t(`applications.statuses.${app.status}`)}
                    </span>
                  </div>
                  {app.jobAvailable ? (
                    <p>{app.companyName}</p>
                  ) : (
                    <p>{t("applications.unavailableHint")}</p>
                  )}
                  <div className="application-row__meta">
                    <span>
                      <CalendarDays size={15} />
                      {t("applications.submittedAt", {
                        date: formatDate(app.createdAt, i18n.language),
                      })}
                    </span>
                    <span>
                      {t("applications.updatedAt", {
                        date: formatDate(app.latestStatusAt, i18n.language),
                      })}
                    </span>
                  </div>
                  {app.message ? (
                    <p className="application-message">{app.message}</p>
                  ) : null}
                </div>
                <div className="application-row__actions">
                  {app.jobAvailable ? (
                    <Link
                      className="ui-button ui-button--quiet ui-button--sm"
                      to={`/jobs/${app.jobId}`}
                    >
                      {t("applications.viewJob")}
                      <ChevronRight size={15} />
                    </Link>
                  ) : null}
                  {app.status !== "rejected" && app.status !== "withdrawn" ? (
                    <Button
                      size="sm"
                      variant="danger"
                      isLoading={withdraw.isPending}
                      onClick={() =>
                        window.confirm(t("applications.withdrawConfirm")) &&
                        withdraw.mutate(app.id)
                      }
                    >
                      {t("applications.withdraw")}
                    </Button>
                  ) : null}
                </div>
              </article>
            ))}
          </div>
        ) : null}
        {apps.data?.length && visibleApplications?.length === 0 ? (
          <p className="section-empty application-filter-empty">
            {t("applications.filterEmpty")}
          </p>
        ) : null}
    </CandidateJobsLayout>
  );
}
function formatDate(value: string, language: string) {
  return new Intl.DateTimeFormat(language, { dateStyle: "medium" }).format(
    new Date(value),
  );
}
const applicationStatuses: ApplicationStatus[] = [
  "sent",
  "viewed",
  "in_review",
  "contacted",
  "interview",
  "offer",
  "rejected",
  "withdrawn",
];
