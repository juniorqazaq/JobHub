import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { ArrowLeft, BriefcaseBusiness, ExternalLink, Globe2, MapPin, Star, UserCheck, UserPlus } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link, useParams } from "react-router-dom";
import { useAuth } from "../app/authContext";
import { repositories } from "../api/repositories";
import { CompanyIdentity } from "../components/companies/CompanyIdentity";
import { Button } from "../components/ui/Button";
import { EmptyState, ErrorState, JobListSkeleton } from "../components/ui/Feedback";
import { SiteShell } from "../layouts/SiteShell";
import { useToast } from "../components/ui/useToast";

export function CompanyProfilePage() {
  const { companyId = "" } = useParams();
  const [expandedCompany, setExpandedCompany] = useState<string | null>(null);
  const showVacancies = expandedCompany === companyId;
  const { t, i18n } = useTranslation();
  const auth = useAuth();
  const toast = useToast();
  const queryClient = useQueryClient();
  const detail = useQuery({ queryKey: ["company", companyId], queryFn: () => repositories.companies.get(companyId), enabled: Boolean(companyId), staleTime: 5 * 60_000 });
  const vacancies = useInfiniteQuery({
    queryKey: ["company-jobs", companyId],
    queryFn: ({ pageParam }) => repositories.companies.getJobs(companyId, pageParam, 20),
    initialPageParam: 1,
    getNextPageParam: (lastPage) => lastPage.page < lastPage.totalPages ? lastPage.page + 1 : undefined,
    enabled: showVacancies && Boolean(companyId),
    staleTime: 2 * 60_000,
  });
  const companyJobs = vacancies.data?.pages.flatMap((page) => page.items) ?? [];
  const companyJobCount = vacancies.data?.pages[0]?.total ?? detail.data?.openJobsCount ?? 0;
  const canFollow = auth.session?.user.role === "job_seeker";
  const followState = useQuery({ queryKey: ["company-follow", companyId], queryFn: () => repositories.companies.getFollowState(companyId), enabled: canFollow && Boolean(companyId) });
  const follow = useMutation({
    mutationFn: async () => {
      const token = auth.session?.csrfToken;
      if (!token) throw new Error("missing csrf token");
      if (followState.data) await repositories.companies.unfollow(companyId, token); else await repositories.companies.follow(companyId, token);
    },
    onSuccess: async () => {
      await Promise.all([queryClient.invalidateQueries({ queryKey: ["company-follow", companyId] }), queryClient.invalidateQueries({ queryKey: ["company", companyId] }), queryClient.invalidateQueries({ queryKey: ["followed-companies"] })]);
      toast.showToast({ title: followState.data ? t("companies.unfollowed") : t("companies.followed") });
    },
    onError: () => toast.showToast({ title: t("companies.followError"), tone: "error" }),
  });
  return (
    <SiteShell>
      <div className="page-container company-profile-page">
        <Link className="back-link" to="/companies"><ArrowLeft size={17} aria-hidden="true" />{t("companies.back")}</Link>
        {detail.isPending ? <JobListSkeleton label={t("companies.loadingProfile")} /> : null}
        {detail.isError ? <ErrorState title={t("companies.profileErrorTitle")} description={t("companies.profileErrorDescription")} actionLabel={t("common.retry")} onAction={() => void detail.refetch()} /> : null}
        {detail.data ? <>
          <section className="company-hero">
            <CompanyIdentity company={detail.data} size="hero" verifiedLabel={t("companies.verified")} />
            <div className="company-hero__actions">
              {canFollow ? <Button variant={followState.data ? "secondary" : "primary"} leadingIcon={followState.data ? <UserCheck size={18} /> : <UserPlus size={18} />} isLoading={follow.isPending || followState.isPending} onClick={() => follow.mutate()}>{followState.data ? t("companies.following") : t("companies.follow")}</Button> : null}
              {detail.data.websiteUrl ? <a className="ui-button ui-button--secondary ui-button--md" href={detail.data.websiteUrl} target="_blank" rel="noreferrer"><ExternalLink size={18} aria-hidden="true" /><span>{t("companies.website")}</span></a> : null}
            </div>
          </section>
          <div className="company-profile-layout">
            <section className="company-about" id="company-about"><h2>{t("companies.about")}</h2><p>{detail.data.description || t("companies.noDescription")}</p><section className="company-rating-placeholder"><Star size={19} aria-hidden="true" /><div><h2>{t("companies.ratingTitle")}</h2><p>{t("companies.ratingPlaceholder")}</p></div></section></section>
            <aside className="company-details" aria-label={t("companies.companyDetails")}>
              {detail.data.city ? <div><MapPin size={20} aria-hidden="true" /><span><small>{t("companies.cityLabel")}</small><strong>{detail.data.city}</strong></span></div> : null}
              {detail.data.industry ? <div><BriefcaseBusiness size={20} aria-hidden="true" /><span><small>{t("companies.industryLabel")}</small><strong>{detail.data.industry}</strong></span></div> : null}
              {detail.data.websiteUrl ? <a href={detail.data.websiteUrl} target="_blank" rel="noreferrer"><Globe2 size={20} aria-hidden="true" /><span><small>{t("companies.websiteLabel")}</small><strong>{detail.data.websiteUrl.replace(/^https?:\/\//, "")}</strong></span><ExternalLink size={16} aria-hidden="true" /></a> : null}
            </aside>
          </div>
          <div className="company-vacancy-action">
            <span>{t("companies.jobCount", { count: companyJobCount })}</span>
            <Button aria-expanded={showVacancies} aria-controls="company-vacancies" onClick={() => setExpandedCompany(showVacancies ? null : companyId)}>{t(showVacancies ? "companies.hideJobs" : "companies.showJobs")}</Button>
          </div>
          <section hidden={!showVacancies} className="company-open-jobs" id="company-vacancies"><div className="section-heading"><div><h2>{t("companies.openJobs")}</h2><p>{t("companies.jobCount", { count: companyJobCount })}</p></div></div>
            {vacancies.isPending ? <JobListSkeleton label={t("companies.loadingJobs")} /> : null}
            {vacancies.isError ? <ErrorState title={t("companies.jobsErrorTitle")} description={t("companies.jobsErrorDescription")} actionLabel={t("common.retry")} onAction={() => void vacancies.refetch()} /> : null}
            {vacancies.isSuccess && companyJobs.length === 0 ? <EmptyState title={t("companies.noJobsTitle")} description={t("companies.noJobsDescription")} /> : null}
            {companyJobs.length > 0 ? <div className="company-vacancy-list">{companyJobs.map((job) => <Link key={job.id} to={`/jobs/${job.id}`}>
              <span className="company-vacancy-list__identity"><strong>{job.title}</strong><small>{[job.location, job.employmentType ? t(`employer.employmentTypes.${job.employmentType}`, { defaultValue: job.employmentType }) : ""].filter(Boolean).join(" · ")}</small></span>
              <span className={job.source === "external" ? "company-vacancy-list__source source-cue source-cue--external" : "company-vacancy-list__source source-cue"}>{job.sourceName}</span>
              <span className="company-vacancy-list__date">{job.publishedAt ? new Intl.DateTimeFormat(i18n.language, { dateStyle: "medium" }).format(new Date(job.publishedAt)) : ""}</span>
            </Link>)}</div> : null}
            {vacancies.hasNextPage ? <div className="company-vacancy-list__more"><Button variant="secondary" isLoading={vacancies.isFetchingNextPage} onClick={() => void vacancies.fetchNextPage()}>{t("companies.loadMoreJobs")}</Button></div> : null}
            </section>
        </> : null}
      </div>
    </SiteShell>
  );
}
