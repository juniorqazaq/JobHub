import { useTranslation } from "react-i18next";
import type { EmploymentType, ExperienceLevel, JobSearchParams, WorkMode } from "../../api/models/job";
import { CitySelect } from "../location/CitySelect";
import { Button } from "../ui/Button";
import { Checkbox, Input, Select } from "../ui/FormControls";

const modes: WorkMode[] = ["on_site", "hybrid", "remote"];
const experiences: ExperienceLevel[] = ["no_experience", "junior", "middle", "senior", "lead"];
const employments: EmploymentType[] = ["full_time", "part_time", "contract", "temporary", "internship"];

export function JobFilters({ search, onChange, onReset }: { search: JobSearchParams; onChange: (next: JobSearchParams) => void; onReset: () => void }) {
  const { t } = useTranslation();
  const update = (patch: Partial<JobSearchParams>) => onChange({ ...search, ...patch, page: 1 });
  const toggleMode = (mode: WorkMode, checked: boolean) => update({ workModes: checked ? [...(search.workModes ?? []), mode] : (search.workModes ?? []).filter((item) => item !== mode) });
  return (
    <form className="job-filters" onSubmit={(event) => { event.preventDefault(); const data = new FormData(event.currentTarget); const raw = String(data.get("salary_min") ?? ""); const parsed = raw === "" ? undefined : Number(raw); const amount = parsed != null && Number.isFinite(parsed) && parsed >= 0 ? parsed : undefined; update({ salaryMin: amount, currency: amount == null ? undefined : String(data.get("currency") ?? "KZT") as JobSearchParams["currency"] }); }}>
      <CitySelect label={t("search.locationLabel")} anyLabel={t("jobs.anyLocation")} value={search.city} onChange={(city) => update({ city: city || undefined })} />
      <fieldset className="filter-fieldset"><legend>{t("jobs.workMode")}</legend>{modes.map((mode) => <Checkbox key={mode} label={t(`employer.workModes.${mode}`)} checked={search.workModes?.includes(mode) ?? false} onChange={(event) => toggleMode(mode, event.target.checked)} />)}</fieldset>
      <div className="filter-salary">
        <Input key={search.salaryMin ?? "empty"} name="salary_min" label={t("jobs.minimumSalary")} type="number" min="0" step="10000" defaultValue={search.salaryMin?.toString() ?? ""} />
        <Select key={search.currency ?? "KZT"} name="currency" label={t("profile.fields.currency")} defaultValue={search.currency ?? "KZT"}><option value="KZT">KZT</option><option value="USD">USD</option><option value="EUR">EUR</option></Select>
        <Button type="submit" variant="secondary" size="sm">{t("jobs.applySalary")}</Button>
      </div>
      <Select label={t("employer.fields.experienceLevel")} value={search.experience ?? ""} onChange={(event) => update({ experience: (event.target.value || undefined) as ExperienceLevel | undefined })}><option value="">{t("jobs.anyOption")}</option>{experiences.map((value) => <option key={value} value={value}>{t(`employer.experienceLevels.${value}`)}</option>)}</Select>
      <Select label={t("employer.fields.employmentType")} value={search.employment ?? ""} onChange={(event) => update({ employment: (event.target.value || undefined) as EmploymentType | undefined })}><option value="">{t("jobs.anyOption")}</option>{employments.map((value) => <option key={value} value={value}>{t(`employer.employmentTypes.${value}`)}</option>)}</Select>
      <Select label={t("jobs.datePosted")} value={search.datePosted ?? ""} onChange={(event) => update({ datePosted: (event.target.value || undefined) as JobSearchParams["datePosted"] })}><option value="">{t("jobs.anyTime")}</option>{["24h", "3d", "7d", "30d"].map((value) => <option key={value} value={value}>{t(`jobs.dateOptions.${value}`)}</option>)}</Select>
      <Button type="button" variant="secondary" onClick={onReset}>{t("jobs.reset")}</Button>
    </form>
  );
}
