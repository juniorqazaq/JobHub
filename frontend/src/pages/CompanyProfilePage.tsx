import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, BriefcaseBusiness, ExternalLink, MapPin, UserCheck, UserPlus } from "lucide-react";
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
  const { t, i18n } = useTranslation();
  const auth = useAuth();
  const toast = useToast();
  const queryClient = useQueryClient();
  const detail = useQuery({ queryKey: ["company", companyId], queryFn: () => repositories.companies.get(companyId), enabled: Boolean(companyId) });
  const canFollow = auth.session?.user.role === "job_seeker";
  const followState = useQuery({ queryKey: ["company-follow", companyId], queryFn: () => repositories.companies.getFollowState(companyId), enabled: canFollow && Boolean(companyId) });
  const follow = useMutation({
    mutationFn: async () => {
      const token = auth.session?.csrfToken;
      if (!token) throw new Error("missing csrf token");
      if (followState.data) await repositories.companies.unfollow(companyId, token); else await repositories.companies.follow(companyId, token);
    },
    onSuccess: async () => {
      await Promise.all([queryClient.invalidateQueries({ queryKey: ["company-follow", companyId] }), queryClient.invalidateQueries({ queryKey: ["company", companyId] })]);
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
            <div><CompanyIdentity company={detail.data.company} size="hero" verifiedLabel={t("companies.verified")} /><div className="company-hero__meta">{detail.data.company.industry ? <span><BriefcaseBusiness size={16} aria-hidden="true" />{detail.data.company.industry}</span> : null}{detail.data.company.city ? <span><MapPin size={16} aria-hidden="true" />{detail.data.company.city}</span> : null}</div></div>
            <div className="company-hero__actions">
              {canFollow ? <Button variant={followState.data ? "secondary" : "primary"} leadingIcon={followState.data ? <UserCheck size={18} /> : <UserPlus size={18} />} isLoading={follow.isPending || followState.isPending} onClick={() => follow.mutate()}>{followState.data ? t("companies.following") : t("companies.follow")}</Button> : null}
              {detail.data.company.websiteUrl ? <a className="ui-button ui-button--secondary ui-button--md" href={detail.data.company.websiteUrl} target="_blank" rel="noreferrer"><ExternalLink size={18} aria-hidden="true" /><span>{t("companies.website")}</span></a> : null}
            </div>
          </section>
          <div className="company-profile-layout">
            <section className="company-about"><h2>{t("companies.about")}</h2><p>{detail.data.company.description || t("companies.noDescription")}</p></section>
            <section className="company-open-jobs"><div className="section-heading"><div><h2>{t("companies.openJobs")}</h2><p>{t("companies.jobCount", { count: detail.data.jobs.length })}</p></div></div>
              {detail.data.jobs.length === 0 ? <EmptyState title={t("companies.noJobsTitle")} description={t("companies.noJobsDescription")} /> : <div className="company-vacancy-list">{detail.data.jobs.map((job) => <Link key={job.id} to={`/jobs/${job.id}`}><span><strong>{job.title}</strong><small>{[job.location, job.employmentType ? t(`employer.employmentTypes.${job.employmentType}`, { defaultValue: job.employmentType }) : ""].filter(Boolean).join(" · ")}</small></span><span>{job.publishedAt ? new Intl.DateTimeFormat(i18n.language, { dateStyle: "medium" }).format(new Date(job.publishedAt)) : ""}</span></Link>)}</div>}
            </section>
          </div>
        </> : null}
      </div>
    </SiteShell>
  );
}
