import { useQuery } from "@tanstack/react-query";
import { Bookmark, ChevronRight, MapPin } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { repositories } from "../api/repositories";
import {
  EmptyState,
  ErrorState,
  JobListSkeleton,
} from "../components/ui/Feedback";
import { SiteShell } from "../layouts/SiteShell";

export function SavedJobsPage() {
  const { t } = useTranslation();
  const saved = useQuery({
    queryKey: ["saved-jobs"],
    queryFn: () => repositories.candidate.listSavedJobs(),
  });
  return (
    <SiteShell>
      <main className="candidate-list-page page-container">
        <header>
          <p className="auth-eyebrow">{t("saved.eyebrow")}</p>
          <h1>{t("saved.title")}</h1>
          <p>{t("saved.description")}</p>
        </header>
        {saved.isPending ? (
          <JobListSkeleton label={t("saved.loading")} />
        ) : null}
        {saved.isError ? (
          <ErrorState
            title={t("saved.loadErrorTitle")}
            description={t("saved.loadErrorDescription")}
            actionLabel={t("common.retry")}
            onAction={() => void saved.refetch()}
          />
        ) : null}
        {saved.data?.length === 0 ? (
          <EmptyState
            title={t("saved.emptyTitle")}
            description={t("saved.emptyDescription")}
            secondary={
              <Link className="text-link" to="/jobs">
                {t("saved.browse")}
              </Link>
            }
          />
        ) : null}
        {saved.data?.length ? (
          <div className="candidate-job-list">
            {saved.data.map((item) =>
              item.available ? (
                <Link
                  className="candidate-job-row"
                  to={`/jobs/${item.jobId}`}
                  key={item.jobId}
                >
                  <Bookmark size={20} fill="currentColor" aria-hidden="true" />
                  <div>
                    <h2>{item.title}</h2>
                    <p>{item.companyName}</p>
                    <span>
                      <MapPin size={15} />
                      {item.location}
                    </span>
                  </div>
                  <ChevronRight size={20} />
                </Link>
              ) : (
                <article
                  className="candidate-job-row is-unavailable"
                  key={item.jobId}
                >
                  <Bookmark size={20} />
                  <div>
                    <h2>{t("saved.unavailable")}</h2>
                    <p>{t("saved.unavailableHint")}</p>
                  </div>
                </article>
              ),
            )}
          </div>
        ) : null}
      </main>
    </SiteShell>
  );
}
