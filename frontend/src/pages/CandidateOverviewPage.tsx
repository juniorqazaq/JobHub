import { useQuery } from "@tanstack/react-query";
import { ArrowRight, FileText } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { repositories } from "../api/repositories";
import { ErrorState, Skeleton } from "../components/ui/Feedback";
import { CandidateIdentityHeader } from "../components/candidate/CandidateIdentityHeader";
import { CandidateWorkspaceLayout } from "../components/candidate/CandidateWorkspaceLayout";

export function CandidateOverviewPage() {
  const { t, i18n } = useTranslation();
  const profile = useQuery({
    queryKey: ["candidate-profile"],
    queryFn: () => repositories.candidate.getProfile(),
  });
  const applications = useQuery({
    queryKey: ["applications"],
    queryFn: () => repositories.candidate.listApplications(),
  });
  const saved = useQuery({
    queryKey: ["saved-jobs"],
    queryFn: () => repositories.candidate.listSavedJobs(),
  });

  return (
    <CandidateWorkspaceLayout>
      <section
        className="workspace-page"
        aria-labelledby="workspace-overview-title"
      >
        {profile.isPending ? <OverviewSkeleton /> : null}
        {profile.isError ? (
          <ErrorState
            title={t("profile.loadErrorTitle")}
            description={t("profile.loadErrorDescription")}
            actionLabel={t("common.retry")}
            onAction={() => void profile.refetch()}
          />
        ) : null}
        {profile.data ? (
          <>
            <CandidateIdentityHeader
              eyebrow={t("workspace.eyebrow")}
              profile={profile.data}
              titleId="workspace-overview-title"
            />

            <div className="workspace-summary-list">
              <section>
                <h2>{t("workspace.profileStatus")}</h2>
                <dl>
                  <div>
                    <dt>{t("profile.fields.searchStatus")}</dt>
                    <dd>
                      {t(`profile.searchStatuses.${profile.data.searchStatus}`)}
                    </dd>
                  </div>
                  <div>
                    <dt>{t("workspace.resumeStatus")}</dt>
                    <dd>
                      {profile.data.resume
                        ? t("workspace.resumeReady")
                        : t("workspace.resumeMissing")}
                    </dd>
                  </div>
                  {saved.data ? (
                    <div>
                      <dt>{t("workspace.savedCount")}</dt>
                      <dd>{saved.data.length}</dd>
                    </div>
                  ) : null}
                </dl>
                {profile.data.completion.missing.length ? (
                  <Link className="text-link" to="/profile">
                    {t("workspace.completeProfile")}{" "}
                    <ArrowRight size={16} aria-hidden="true" />
                  </Link>
                ) : null}
              </section>
              <section>
                <div className="workspace-section-heading">
                  <h2>{t("workspace.recentApplications")}</h2>
                  <Link className="text-link" to="/applications">
                    {t("workspace.viewAll")}
                  </Link>
                </div>
                {applications.isPending ? <Skeleton /> : null}
                {applications.data?.length === 0 ? (
                  <p className="section-empty">
                    {t("applications.emptyDescription")}
                  </p>
                ) : null}
                {applications.data?.slice(0, 3).map((application) => (
                  <article
                    className="workspace-application"
                    key={application.id}
                  >
                    <FileText size={18} aria-hidden="true" />
                    <div>
                      <strong>
                        {application.jobAvailable
                          ? application.jobTitle
                          : t("applications.unavailableJob")}
                      </strong>
                      <span>{application.companyName}</span>
                    </div>
                    <div>
                      <span
                        className={`application-status status-${application.status}`}
                      >
                        {t(`applications.statuses.${application.status}`)}
                      </span>
                      <small>
                        {formatDate(application.latestStatusAt, i18n.language)}
                      </small>
                    </div>
                  </article>
                ))}
              </section>
            </div>
          </>
        ) : null}
      </section>
    </CandidateWorkspaceLayout>
  );
}

function OverviewSkeleton() {
  return (
    <div className="profile-skeleton" role="status">
      <Skeleton className="detail-skeleton__title" />
      <Skeleton />
      <Skeleton className="detail-skeleton__body" />
    </div>
  );
}
function formatDate(value: string, language: string) {
  return new Intl.DateTimeFormat(language, { dateStyle: "medium" }).format(
    new Date(value),
  );
}
