import { useQuery } from "@tanstack/react-query";
import { ArrowRight, FileText } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { repositories } from "../api/repositories";
import { ErrorState, Skeleton } from "../components/ui/Feedback";
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
            <header className="workspace-overview-header">
              <div className="profile-avatar" aria-hidden="true">
                {initials(profile.data.fullName)}
              </div>
              <div>
                <p className="auth-eyebrow">{t("workspace.eyebrow")}</p>
                <h1 id="workspace-overview-title">{profile.data.fullName}</h1>
                <p>
                  {profile.data.desiredPosition ||
                    profile.data.currentPosition ||
                    t("profile.positionMissing")}
                </p>
              </div>
              <div className="profile-completion">
                <strong>{profile.data.completion.percentage}%</strong>
                <span>{t("profile.complete")}</span>
                <div aria-hidden="true">
                  <span
                    style={{ width: `${profile.data.completion.percentage}%` }}
                  />
                </div>
              </div>
            </header>

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
function initials(value: string) {
  return value
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toLocaleUpperCase())
    .join("");
}
function formatDate(value: string, language: string) {
  return new Intl.DateTimeFormat(language, { dateStyle: "medium" }).format(
    new Date(value),
  );
}
