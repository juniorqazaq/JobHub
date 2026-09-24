import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { Bookmark, BriefcaseBusiness, Search } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { z } from "zod";
import { repositories } from "../api/repositories";
import { Button, IconButton } from "../components/ui/Button";
import { Avatar, Badge, Card, Progress } from "../components/ui/DataDisplay";
import { EmptyState, ErrorState, JobListSkeleton } from "../components/ui/Feedback";
import { Checkbox, Input, RadioGroup, SearchInput, Select, Switch, Textarea } from "../components/ui/FormControls";
import { Chip, Pagination, Tabs } from "../components/ui/Navigation";
import { Drawer, Dropdown, Modal, Tooltip } from "../components/ui/Overlays";
import { useToast } from "../components/ui/useToast";
import { SourceAwareJobRow } from "../components/jobs/SourceAwareJobRow";
import { env } from "../lib/env";

type Language = "kk" | "ru";

export function ComponentShowcasePage() {
  const { t, i18n } = useTranslation();
  const { showToast } = useToast();
  const [searchValue, setSearchValue] = useState("");
  const [radioValue, setRadioValue] = useState("full-time");
  const [alerts, setAlerts] = useState(true);
  const [activeTab, setActiveTab] = useState("jobs");
  const [page, setPage] = useState(1);
  const [modalOpen, setModalOpen] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);

  const schema = useMemo(() => z.object({ keyword: z.string().min(2, t("dev.formError")) }), [t]);
  type DemoForm = z.infer<typeof schema>;
  const { register, handleSubmit, formState: { errors } } = useForm<DemoForm>({
    resolver: zodResolver(schema),
    defaultValues: { keyword: "" },
  });

  const jobs = useQuery({
    queryKey: ["dev", "jobs-repository"],
    queryFn: () => repositories.jobs.search({ pageSize: 5 }),
  });

  useEffect(() => {
    document.documentElement.lang = supportedLanguage(i18n.language);
  }, [i18n.language]);

  const language = supportedLanguage(i18n.language);

  return (
    <main className="showcase">
      <header className="showcase-header">
        <div className="wide-container showcase-nav">
          <a className="showcase-brand" href="/dev/ui" aria-label="JobHub">
            <span><BriefcaseBusiness size={19} aria-hidden="true" /></span>JobHub
          </a>
          <div className="showcase-language">
            <label htmlFor="showcase-language">{t("dev.language")}</label>
            <select id="showcase-language" value={language} onChange={(event) => void i18n.changeLanguage(event.target.value)}>
              <option value="kk">Қазақша</option>
              <option value="ru">Русский</option>
            </select>
          </div>
        </div>
      </header>

      <div className="page-container showcase-content">
        <section className="showcase-intro">
          <div>
            <p className="showcase-eyebrow">{t("dev.eyebrow")}</p>
            <h1>{t("dev.title")}</h1>
            <p>{t("dev.description")}</p>
          </div>
          <Badge tone={env.useMocks ? "warning" : "success"}>{env.useMocks ? t("dev.mockMode") : t("dev.httpMode")}</Badge>
        </section>

        <section className="showcase-section" aria-labelledby="actions-title">
          <div className="showcase-section-heading"><span>01</span><h2 id="actions-title">{t("dev.actions")}</h2></div>
          <Card className="showcase-panel showcase-actions">
            <Button leadingIcon={<Search size={18} />}>{t("dev.primaryAction")}</Button>
            <Button variant="secondary">{t("dev.secondaryAction")}</Button>
            <Button variant="quiet">{t("dev.secondaryAction")}</Button>
            <Button isLoading>{t("dev.primaryAction")}</Button>
            <Tooltip label={t("dev.tooltip")}><IconButton label={t("dev.iconAction")} icon={<Bookmark size={19} />} /></Tooltip>
          </Card>
        </section>

        <section className="showcase-section" aria-labelledby="fields-title">
          <div className="showcase-section-heading"><span>02</span><h2 id="fields-title">{t("dev.fields")}</h2></div>
          <Card className="showcase-panel">
            <form className="showcase-form-grid" onSubmit={handleSubmit(() => showToast({ title: t("dev.toastTitle") }))} noValidate>
              <Input label={t("dev.keyword")} placeholder={t("dev.keywordPlaceholder")} error={errors.keyword?.message} {...register("keyword")} />
              <SearchInput label={t("dev.location")} placeholder={t("dev.locationPlaceholder")} clearLabel={t("common.clearSearch")} value={searchValue} onChange={(event) => setSearchValue(event.target.value)} onClear={() => setSearchValue("")} />
              <Select label={t("dev.experience")} defaultValue="">
                <option value="" disabled>{t("dev.chooseExperience")}</option>
                <option value="junior">{t("dev.junior")}</option>
                <option value="middle">{t("dev.middle")}</option>
                <option value="senior">{t("dev.senior")}</option>
              </Select>
              <Textarea label={t("dev.about")} placeholder={t("dev.aboutPlaceholder")} />
              <div className="showcase-choice-stack">
                <Checkbox label={t("dev.remote")} defaultChecked />
                <RadioGroup label={t("dev.experience")} name="employment" value={radioValue} onChange={setRadioValue} options={[{ value: "full-time", label: t("dev.fullTime") }, { value: "part-time", label: t("dev.partTime") }]} />
              </div>
              <Switch label={t("dev.alerts")} description={t("dev.alertsDescription")} checked={alerts} onChange={(event) => setAlerts(event.target.checked)} />
              <div className="showcase-submit"><Button type="submit">{t("dev.primaryAction")}</Button></div>
            </form>
          </Card>
        </section>

        <section className="showcase-section" aria-labelledby="feedback-title">
          <div className="showcase-section-heading"><span>03</span><h2 id="feedback-title">{t("dev.feedback")}</h2></div>
          <div className="showcase-status-grid">
            <Card className="showcase-panel showcase-badges">
              <div><Badge tone="success">{t("dev.verified")}</Badge><Badge tone="warning">{t("dev.review")}</Badge><Chip onRemove={() => undefined} removeLabel={t("dev.activeFilterRemove")}>{t("dev.activeFilter")}</Chip></div>
              <div className="showcase-profile"><Avatar name="Aruzhan Saparova" /><Progress value={72} label={t("dev.profileComplete")} /></div>
            </Card>
            <EmptyState title={t("dev.emptyTitle")} description={t("dev.emptyDescription")} actionLabel={t("dev.emptyAction")} onAction={() => undefined} />
            <ErrorState title={t("dev.errorTitle")} description={t("dev.errorDescription")} actionLabel={t("common.retry")} onAction={() => void jobs.refetch()} />
          </div>
          <Card className="showcase-panel showcase-data-status">
            <div><h3>{t("dev.dataLayer")}</h3><p>{t("dev.dataLayerDescription")}</p></div>
            {jobs.isPending ? <JobListSkeleton label={t("dev.skeletonLabel")} /> : jobs.isError ? <Badge tone="error">{t("dev.errorTitle")}</Badge> : <strong>{t("dev.repositoryCount", { count: jobs.data.total })}</strong>}
          </Card>
        </section>

        <section className="showcase-section" aria-labelledby="overlay-title">
          <div className="showcase-section-heading"><span>04</span><h2 id="overlay-title">{t("dev.overlays")}</h2></div>
          <Card className="showcase-panel showcase-overlay-demo">
            <Tabs label={t("dev.overlays")} activeId={activeTab} onChange={setActiveTab} tabs={[{ id: "jobs", label: t("dev.tabJobs") }, { id: "searches", label: t("dev.tabSearches") }]} />
            <div className="showcase-actions">
              <Button variant="secondary" onClick={() => setModalOpen(true)}>{t("dev.openModal")}</Button>
              <Button variant="secondary" onClick={() => setDrawerOpen(true)}>{t("dev.openDrawer")}</Button>
              <Button variant="secondary" onClick={() => showToast({ title: t("dev.toastTitle"), description: t("dev.toastDescription") })}>{t("dev.showToast")}</Button>
              <Dropdown label={t("dev.dropdown")} items={[{ id: "edit", label: t("dev.edit"), onSelect: () => undefined }, { id: "delete", label: t("dev.delete"), destructive: true, onSelect: () => undefined }]} />
            </div>
            <Pagination page={page} totalPages={5} onChange={setPage} label={t("dev.pagination")} previousLabel={t("common.previous")} nextLabel={t("common.next")} pageLabel={(item) => t("common.page", { page: item })} />
          </Card>
        </section>

        <section className="showcase-section" aria-labelledby="source-jobs-title">
          <div className="showcase-section-heading"><span>05</span><h2 id="source-jobs-title">{t("dev.sourceJobs")}</h2></div>
          <Card className="showcase-panel source-job-list">
            <div className="source-job-list__intro"><h3>{t("dev.sourceJobsTitle")}</h3><p>{t("dev.sourceJobsDescription")}</p></div>
            {jobs.isPending ? <JobListSkeleton label={t("dev.skeletonLabel")} /> : null}
            {jobs.isError ? <ErrorState title={t("dev.errorTitle")} description={t("dev.errorDescription")} actionLabel={t("common.retry")} onAction={() => void jobs.refetch()} /> : null}
            {jobs.data?.items.map((job) => (
              <SourceAwareJobRow key={job.id} job={job} externalLabel={t("dev.viewOnSource")} sourceLabel={t("dev.sourceLabel")} snippetLabel={t("dev.snippetLabel")} />
            ))}
          </Card>
        </section>
      </div>

      <Modal open={modalOpen} onClose={() => setModalOpen(false)} closeLabel={t("common.close")} title={t("dev.modalTitle")} description={t("dev.modalDescription")} footer={<><Button variant="quiet" onClick={() => setModalOpen(false)}>{t("common.close")}</Button><Button onClick={() => setModalOpen(false)}>{t("dev.secondaryAction")}</Button></>} />
      <Drawer open={drawerOpen} onClose={() => setDrawerOpen(false)} closeLabel={t("common.close")} title={t("dev.drawerTitle")} description={t("dev.drawerDescription")}><Checkbox label={t("dev.remote")} /><Switch label={t("dev.alerts")} /></Drawer>
    </main>
  );
}

function supportedLanguage(language: string): Language {
  return language === "ru" ? "ru" : "kk";
}
