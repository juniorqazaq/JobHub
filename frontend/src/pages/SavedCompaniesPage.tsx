import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowRight, MapPin, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { repositories } from "../api/repositories";
import { useAuth } from "../app/authContext";
import { CandidateJobsLayout } from "../components/candidate/CandidateJobsLayout";
import { CompanyIdentity } from "../components/companies/CompanyIdentity";
import { Button } from "../components/ui/Button";
import { EmptyState, ErrorState, JobListSkeleton } from "../components/ui/Feedback";
import { useToast } from "../components/ui/useToast";

export function SavedCompaniesPage() {
  const { t } = useTranslation();
  const { session } = useAuth();
  const client = useQueryClient();
  const toast = useToast();
  const companies = useQuery({ queryKey: ["followed-companies"], queryFn: () => repositories.companies.listFollowing() });
  const unfollow = useMutation({
    mutationFn: (companyId: string) => repositories.companies.unfollow(companyId, session!.csrfToken),
    onSuccess: async () => {
      await Promise.all([client.invalidateQueries({ queryKey: ["followed-companies"] }), client.invalidateQueries({ queryKey: ["company-follow"] })]);
      toast.showToast({ title: t("savedCompanies.removed") });
    },
    onError: () => toast.showToast({ tone: "error", title: t("savedCompanies.error") }),
  });
  return <CandidateJobsLayout tabTitle={t("savedCompanies.title")} tabDescription={t("savedCompanies.description")}>
    {companies.isPending ? <JobListSkeleton label={t("savedCompanies.loading")} /> : null}
    {companies.isError ? <ErrorState title={t("savedCompanies.loadErrorTitle")} description={t("savedCompanies.loadErrorDescription")} actionLabel={t("common.retry")} onAction={() => void companies.refetch()} /> : null}
    {companies.data?.length === 0 ? <EmptyState title={t("savedCompanies.emptyTitle")} description={t("savedCompanies.emptyDescription")} secondary={<Link className="ui-button ui-button--primary ui-button--md" to="/companies">{t("savedCompanies.browse")}</Link>} /> : null}
    {companies.data?.length ? <div className="saved-company-list">{companies.data.map((company) => <article key={company.id} className="saved-company-row">
      <CompanyIdentity company={company} verifiedLabel={t("companies.verified")} />
      <div className="saved-company-row__info"><p>{company.description || t("companies.noDescription")}</p><span>{company.industry || t("savedCompanies.industryMissing")}</span>{company.city ? <span><MapPin size={15} aria-hidden="true" />{company.city}</span> : null}</div>
      <div className="saved-company-row__actions"><Link className="ui-icon-button ui-icon-button--quiet" aria-label={t("companies.openProfile")} to={`/companies/${company.id}`}><ArrowRight size={19} /></Link><Button variant="quiet" size="sm" leadingIcon={<Trash2 size={16} />} isLoading={unfollow.isPending} onClick={() => unfollow.mutate(company.id)}>{t("savedCompanies.unfollow")}</Button></div>
    </article>)}</div> : null}
  </CandidateJobsLayout>;
}
