import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Save } from "lucide-react";
import { useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { repositories } from "../api/repositories";
import type { Company, CompanyProfileInput } from "../api/models/company";
import { useAuth } from "../app/authContext";
import { EmployerWorkspaceNav } from "../components/employer/EmployerWorkspaceNav";
import { Button } from "../components/ui/Button";
import { ErrorState, JobListSkeleton } from "../components/ui/Feedback";
import { Input, Textarea } from "../components/ui/FormControls";
import { useToast } from "../components/ui/useToast";
import { SiteShell } from "../layouts/SiteShell";

export function EmployerCompanyPage() {
  const { t } = useTranslation();
  const company = useQuery({ queryKey: ["employer-company"], queryFn: () => repositories.companies.getEmployerCompany() });
  return <SiteShell><div className="page-container employer-page"><EmployerWorkspaceNav />
    <header className="vacancy-form-header"><p className="auth-eyebrow">{t("companyEdit.eyebrow")}</p><h1>{t("companyEdit.title")}</h1><p>{t("companyEdit.description")}</p></header>
    {company.isPending ? <JobListSkeleton label={t("companyEdit.loading")} /> : null}
    {company.isError ? <ErrorState title={t("companyEdit.errorTitle")} description={t("companyEdit.errorDescription")} actionLabel={t("common.retry")} onAction={() => void company.refetch()} /> : null}
    {company.data ? <CompanyEditForm key={company.data.updatedAt} company={company.data} /> : null}
  </div></SiteShell>;
}

function CompanyEditForm({ company }: { company: Company }) {
  const { t } = useTranslation();
  const auth = useAuth();
  const toast = useToast();
  const queryClient = useQueryClient();
  const [form, setForm] = useState<CompanyProfileInput>(() => ({ name: company.name, description: company.description ?? "", websiteUrl: company.websiteUrl ?? "", logoUrl: company.logoUrl ?? "", industry: company.industry ?? "", city: company.city ?? "" }));
  const save = useMutation({
    mutationFn: () => repositories.companies.updateEmployerCompany(form, auth.session!.csrfToken),
    onSuccess: async () => { await Promise.all([queryClient.invalidateQueries({ queryKey: ["employer-company"] }), queryClient.invalidateQueries({ queryKey: ["auth", "me"] })]); toast.showToast({ title: t("companyEdit.saved") }); },
    onError: () => toast.showToast({ title: t("companyEdit.saveError"), tone: "error" }),
  });
  const set = (field: keyof CompanyProfileInput) => (value: string) => setForm((current) => ({ ...current, [field]: value }));
  const submit = (event: FormEvent) => { event.preventDefault(); save.mutate(); };
  return <form className="company-edit-form" onSubmit={submit}>
      <Input label={t("companyEdit.name")} value={form.name} minLength={2} maxLength={160} required onChange={(e) => set("name")(e.target.value)} />
      <div className="company-edit-form__row"><Input label={t("companyEdit.industry")} value={form.industry} maxLength={120} onChange={(e) => set("industry")(e.target.value)} /><Input label={t("companyEdit.city")} value={form.city} maxLength={120} onChange={(e) => set("city")(e.target.value)} /></div>
      <Textarea label={t("companyEdit.about")} hint={t("companyEdit.aboutHint")} value={form.description} maxLength={5000} rows={8} onChange={(e) => set("description")(e.target.value)} />
      <Input label={t("companyEdit.website")} type="url" value={form.websiteUrl} maxLength={500} placeholder="https://" onChange={(e) => set("websiteUrl")(e.target.value)} />
      <Input label={t("companyEdit.logo")} type="url" value={form.logoUrl} maxLength={500} placeholder="https://" hint={t("companyEdit.logoHint")} onChange={(e) => set("logoUrl")(e.target.value)} />
      <div><Button type="submit" leadingIcon={<Save size={18} />} isLoading={save.isPending}>{t("companyEdit.save")}</Button></div>
    </form>;
}
