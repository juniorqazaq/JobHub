import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ExternalLink, FilePlus2, Pencil, Play, Pause, XCircle, Trash2, Users } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import type { PublicationStatus } from "../api/models/job";
import { repositories } from "../api/repositories";
import { useAuth } from "../app/authContext";
import { EmployerWorkspaceNav } from "../components/employer/EmployerWorkspaceNav";
import { Button } from "../components/ui/Button";
import { EmptyState, ErrorState, JobListSkeleton } from "../components/ui/Feedback";
import { useToast } from "../components/ui/useToast";
import { SiteShell } from "../layouts/SiteShell";
import { formatSalary } from "../lib/formatSalary";

export function EmployerVacanciesPage() {
  const { t, i18n } = useTranslation();
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const toast = useToast();
  const jobs = useQuery({ queryKey: ["employer", "jobs"], queryFn: () => repositories.employerJobs.list() });
  const refresh = async () => { await queryClient.invalidateQueries({ queryKey: ["employer", "jobs"] }); await queryClient.invalidateQueries({ queryKey: ["jobs"] }); };
  const transition = useMutation({ mutationFn: ({ id, status }: { id: string; status: PublicationStatus }) => repositories.employerJobs.transition(id, status, session!.csrfToken), onSuccess: async () => { await refresh(); toast.showToast({ title: t("employer.statusUpdated") }); } });
  const remove = useMutation({ mutationFn: (id: string) => repositories.employerJobs.delete(id, session!.csrfToken), onSuccess: async () => { await refresh(); toast.showToast({ title: t("employer.vacancyDeleted") }); } });

  const doDelete = (id: string) => { if (window.confirm(t("employer.deleteConfirm"))) remove.mutate(id); };
  return <SiteShell><main className="employer-page page-container">
    <EmployerWorkspaceNav />
    <header className="employer-list-header"><div><p className="auth-eyebrow">{t("auth.employerEyebrow")}</p><h1>{t("employer.myVacancies")}</h1><p>{t("employer.myVacanciesDescription")}</p></div><Link className="ui-button ui-button--primary ui-button--md" to="/employer/vacancies/new"><FilePlus2 size={17} aria-hidden="true" /><span>{t("employer.createVacancy")}</span></Link></header>
    {jobs.isPending ? <JobListSkeleton label={t("employer.loadingVacancies")} /> : null}
    {jobs.isError ? <ErrorState title={t("employer.loadErrorTitle")} description={t("employer.loadErrorDescription")} actionLabel={t("common.retry")} onAction={() => void jobs.refetch()} /> : null}
    {jobs.data?.length === 0 ? <EmptyState title={t("employer.emptyTitle")} description={t("employer.emptyDescription")} secondary={<Link className="text-link" to="/employer/vacancies/new">{t("employer.createVacancy")}</Link>} /> : null}
    {jobs.data?.length ? <div className="employer-job-list">{jobs.data.map((job) => <article className="employer-job-row" key={job.id}>
      <div className="employer-job-row__main"><div className="employer-job-row__title"><h2>{job.title}</h2><span className={`status-badge status-badge--${job.publicationStatus}`}>{t(`employer.statuses.${job.publicationStatus}`)}</span></div><p>{job.location} · {t(`employer.workModes.${job.workMode}`)}</p><small>{t("employer.updatedAt", { date: formatDate(job.updatedAt || job.postedAt, i18n.language) })}</small></div>
      <div className="employer-job-row__salary">{job.salary ? formatSalary(job.salary, i18n.language, t(`employer.salaryPeriods.${job.salary.period}`)) : t("common.notProvided")}</div>
      <div className="employer-job-row__actions">
        {job.publicationStatus === "published" ? <Link className="ui-button ui-button--quiet ui-button--sm" to={`/jobs/${job.id}`}><ExternalLink size={15} aria-hidden="true" /><span>{t("employer.view")}</span></Link> : null}
        <Link className="ui-button ui-button--quiet ui-button--sm" to={`/employer/vacancies/${job.id}/applicants`}><Users size={15} aria-hidden="true" /><span>{t("employer.applicants")}</span></Link>
        {job.publicationStatus !== "closed" ? <Link className="ui-button ui-button--quiet ui-button--sm" to={`/employer/vacancies/${job.id}/edit`}><Pencil size={15} aria-hidden="true" /><span>{t("employer.edit")}</span></Link> : null}
        {job.publicationStatus === "draft" || job.publicationStatus === "paused" ? <Button size="sm" variant="secondary" leadingIcon={<Play size={15} />} disabled={transition.isPending} onClick={() => transition.mutate({ id: job.id, status: "published" })}>{t("employer.publish")}</Button> : null}
        {job.publicationStatus === "published" ? <Button size="sm" variant="secondary" leadingIcon={<Pause size={15} />} disabled={transition.isPending} onClick={() => transition.mutate({ id: job.id, status: "paused" })}>{t("employer.pause")}</Button> : null}
        {job.publicationStatus !== "closed" ? <Button size="sm" variant="danger" leadingIcon={<XCircle size={15} />} disabled={transition.isPending} onClick={() => transition.mutate({ id: job.id, status: "closed" })}>{t("employer.close")}</Button> : null}
        {job.publicationStatus === "draft" || job.publicationStatus === "closed" ? <Button size="sm" variant="danger" leadingIcon={<Trash2 size={15} />} disabled={remove.isPending} onClick={() => doDelete(job.id)}>{t("employer.delete")}</Button> : null}
      </div>
    </article>)}</div> : null}
  </main></SiteShell>;
}

function formatDate(value: string, language: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "" : new Intl.DateTimeFormat(language, { dateStyle: "medium" }).format(date); }
