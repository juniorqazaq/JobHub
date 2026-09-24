import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { BriefcaseBusiness, CalendarDays, ChevronRight } from "lucide-react";
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
import { SiteShell } from "../layouts/SiteShell";

export function ApplicationsPage() {
  const { t, i18n } = useTranslation();
  const { session } = useAuth();
  const client = useQueryClient();
  const toast = useToast();
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
  return (
    <SiteShell>
      <main className="candidate-list-page page-container">
        <header>
          <p className="auth-eyebrow">{t("applications.eyebrow")}</p>
          <h1>{t("applications.title")}</h1>
          <p>{t("applications.description")}</p>
        </header>
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
        {apps.data?.length ? (
          <div className="application-list">
            {apps.data.map((app) => (
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
      </main>
    </SiteShell>
  );
}
function formatDate(value: string, language: string) {
  return new Intl.DateTimeFormat(language, { dateStyle: "medium" }).format(
    new Date(value),
  );
}
