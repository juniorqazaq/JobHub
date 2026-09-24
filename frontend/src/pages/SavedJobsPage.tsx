import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Bookmark, ChevronRight, MapPin, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { repositories } from "../api/repositories";
import { useAuth } from "../app/authContext";
import { CandidateJobsLayout } from "../components/candidate/CandidateJobsLayout";
import { Button } from "../components/ui/Button";
import {
  EmptyState,
  ErrorState,
  JobListSkeleton,
} from "../components/ui/Feedback";
import { useToast } from "../components/ui/useToast";

export function SavedJobsPage() {
  const { t } = useTranslation();
  const { session } = useAuth();
  const client = useQueryClient();
  const toast = useToast();
  const saved = useQuery({
    queryKey: ["saved-jobs"],
    queryFn: () => repositories.candidate.listSavedJobs(),
  });
  const unsave = useMutation({
    mutationFn: (jobId: string) =>
      repositories.candidate.unsaveJob(jobId, session!.csrfToken),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ["saved-jobs"] });
      toast.showToast({ title: t("saved.removed") });
    },
    onError: () => toast.showToast({ tone: "error", title: t("saved.error") }),
  });
  return (
    <CandidateJobsLayout tabTitle={t("saved.title")} tabDescription={t("saved.description")}>
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
            illustration="/illustrations/empty-saved-jobs.png"
            title={t("saved.emptyTitle")}
            description={t("saved.emptyDescription")}
            secondary={
              <Link className="ui-button ui-button--primary ui-button--md" to="/jobs">
                {t("saved.browse")}
              </Link>
            }
          />
        ) : null}
        {saved.data?.length ? (
          <div className="candidate-job-list">
            {saved.data.map((item) =>
              item.available ? (
                <article className="candidate-job-row" key={item.jobId}>
                  <Bookmark size={20} fill="currentColor" aria-hidden="true" />
                  <div>
                    <h2>
                      <Link to={`/jobs/${item.jobId}`}>{item.title}</Link>
                    </h2>
                    <p>{item.companyName}</p>
                    <span>
                      <MapPin size={15} />
                      {item.location}
                    </span>
                    {item.sourceName ? (
                      <small>
                        {t("saved.source", { source: item.sourceName })}
                      </small>
                    ) : null}
                  </div>
                  <div className="candidate-job-row__actions">
                    <Link
                      className="ui-icon-button ui-icon-button--quiet"
                      aria-label={t("saved.openJob", { title: item.title })}
                      to={`/jobs/${item.jobId}`}
                    >
                      <ChevronRight size={20} />
                    </Link>
                    <Button
                      variant="quiet"
                      size="sm"
                      leadingIcon={<Trash2 size={16} />}
                      isLoading={unsave.isPending}
                      onClick={() => unsave.mutate(item.jobId)}
                    >
                      {t("saved.unsave")}
                    </Button>
                  </div>
                </article>
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
    </CandidateJobsLayout>
  );
}
