import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Pencil, Save, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { z } from "zod";
import type { CandidateProfile } from "../api/models/candidate";
import type { EmploymentType, WorkMode } from "../api/models/job";
import { repositories } from "../api/repositories";
import { useAuth } from "../app/authContext";
import { Button } from "../components/ui/Button";
import { CityMultiSelect } from "../components/location/CityMultiSelect";
import { Checkbox, Input, Select } from "../components/ui/FormControls";
import { TagInput } from "../components/ui/TagInput";
import { useToast } from "../components/ui/useToast";
import { editableProfile } from "../lib/candidateProfile";
import { cityName, kazakhstanCities } from "../lib/cities";

const schema = z.object({
  desiredSalary: z.string().refine(
    (value) => value === "" || (Number.isFinite(Number(value)) && Number(value) >= 0),
    "nonnegative",
  ),
  currency: z.string().trim().regex(/^[A-Za-z]{3}$/),
  salaryPeriod: z.enum(["month", "year"]),
  preferredCityIds: z.array(z.string()),
  preferredEmploymentTypes: z.array(z.string()),
  preferredWorkModes: z.array(z.string()),
  preferredCategories: z.array(z.string()),
  preferredRoles: z.array(z.string()),
  searchStatus: z.enum(["actively_looking", "open_to_offers", "not_looking"]),
});
type FormValues = z.infer<typeof schema>;

export function JobPreferencesSection({ profile }: { profile: CandidateProfile }) {
  const { t, i18n } = useTranslation();
  const { session } = useAuth();
  const client = useQueryClient();
  const toast = useToast();
  const [isEditing, setIsEditing] = useState(false);
  const editButtonRef = useRef<HTMLButtonElement>(null);
  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: toForm(profile),
  });

  useEffect(() => {
    if (!isEditing || !form.formState.isDirty) form.reset(toForm(profile));
  }, [form, form.formState.isDirty, isEditing, profile]);
  useEffect(() => {
    if (!isEditing || !form.formState.isDirty) return;
    const warn = (event: BeforeUnloadEvent) => event.preventDefault();
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [form.formState.isDirty, isEditing]);

  const save = useMutation({
    mutationFn: (values: FormValues) =>
      repositories.candidate.updateProfile(
        {
          ...editableProfile(profile),
          desiredSalary: values.desiredSalary === "" ? undefined : Number(values.desiredSalary),
          currency: values.currency.toUpperCase(),
          salaryPeriod: values.salaryPeriod,
          preferredLocations: profile.preferredLocations,
          preferredCityIds: values.preferredCityIds,
          preferredEmploymentTypes: values.preferredEmploymentTypes as EmploymentType[],
          preferredWorkModes: values.preferredWorkModes as WorkMode[],
          preferredCategories: values.preferredCategories,
          preferredRoles: values.preferredRoles,
          searchStatus: values.searchStatus,
        },
        session!.csrfToken,
      ),
    onSuccess: (data) => {
      client.setQueryData(["candidate-profile"], data);
      form.reset(toForm(data));
      setIsEditing(false);
      toast.showToast({ title: t("preferences.saved") });
      requestAnimationFrame(() => editButtonRef.current?.focus());
    },
    onError: () => toast.showToast({ tone: "error", title: t("preferences.saveError") }),
  });

  const startEditing = () => {
    form.reset(toForm(profile));
    setIsEditing(true);
  };
  const cancelEditing = () => {
    if (form.formState.isDirty && !window.confirm(t("preferences.discardConfirm"))) return;
    form.reset(toForm(profile));
    setIsEditing(false);
    requestAnimationFrame(() => editButtonRef.current?.focus());
  };

  return (
    <section id="preferences" className="profile-section resume-preferences" aria-labelledby="preferences-title">
      <header>
        <h2 id="preferences-title">{t("profile.sections.preferences")}</h2>
        {!isEditing ? (
          <Button ref={editButtonRef} type="button" variant="secondary" size="sm" leadingIcon={<Pencil size={16} />} onClick={startEditing}>
            {t("preferences.edit")}
          </Button>
        ) : null}
      </header>
      {!isEditing ? (
        <PreferenceView profile={profile} locale={i18n.resolvedLanguage ?? i18n.language} />
      ) : (
        <form
          className="profile-form"
          onSubmit={form.handleSubmit((values) => {
            if (!save.isPending) save.mutate(values);
          })}
          noValidate
        >
          <div className="profile-section__body">
            <p className="section-intro">{t("workspace.preferencesDescription")}</p>
            <div className="profile-grid">
              <Input
                label={t("profile.fields.desiredSalary")}
                type="number"
                min="0"
                error={form.formState.errors.desiredSalary ? t("profile.validation.nonnegative") : undefined}
                {...form.register("desiredSalary")}
              />
              <Input
                label={t("profile.fields.currency")}
                maxLength={3}
                error={form.formState.errors.currency ? t("profile.validation.currency") : undefined}
                {...form.register("currency")}
              />
              <Select label={t("profile.fields.salaryPeriod")} {...form.register("salaryPeriod")}>
                <option value="month">{t("employer.salaryPeriods.month")}</option>
                <option value="year">{t("employer.salaryPeriods.year")}</option>
              </Select>
              <Select label={t("profile.fields.searchStatus")} {...form.register("searchStatus")}>
                {["actively_looking", "open_to_offers", "not_looking"].map((value) => (
                  <option key={value} value={value}>{t(`profile.searchStatuses.${value}`)}</option>
                ))}
              </Select>
            </div>
            <div className="preference-tag-fields">
              <Controller control={form.control} name="preferredCityIds" render={({ field }) => (
                <CityMultiSelect label={t("profile.fields.preferredLocations")} addLabel={t("preferences.addCity")} value={field.value} onChange={field.onChange} legacyValues={profile.preferredLocations} removeLabel={(value) => t("preferences.removeTag", { value })} />
              )} />
              <Controller control={form.control} name="preferredRoles" render={({ field }) => (
                <TagInput label={t("profile.fields.preferredRoles")} value={field.value} onChange={field.onChange} placeholder={t("preferences.tagPlaceholder")} hint={t("preferences.tagHint")} removeLabel={(value) => t("preferences.removeTag", { value })} />
              )} />
              <Controller control={form.control} name="preferredCategories" render={({ field }) => (
                <TagInput label={t("profile.fields.preferredCategories")} value={field.value} onChange={field.onChange} placeholder={t("preferences.tagPlaceholder")} hint={t("preferences.tagHint")} removeLabel={(value) => t("preferences.removeTag", { value })} />
              )} />
            </div>
            <fieldset className="profile-checks">
              <legend>{t("profile.fields.preferredWorkModes")}</legend>
              {workModes.map((value) => (
                <Checkbox key={value} label={t(`employer.workModes.${value}`)} value={value} {...form.register("preferredWorkModes")} />
              ))}
            </fieldset>
            <fieldset className="profile-checks">
              <legend>{t("profile.fields.preferredEmploymentTypes")}</legend>
              {employmentTypes.map((value) => (
                <Checkbox key={value} label={t(`employer.employmentTypes.${value}`)} value={value} {...form.register("preferredEmploymentTypes")} />
              ))}
            </fieldset>
            {save.isError ? <p className="auth-error" role="alert">{t("preferences.saveError")}</p> : null}
          </div>
          <div className="preference-edit-actions">
            <span>{form.formState.isDirty ? t("profile.unsaved") : t("profile.upToDate")}</span>
            <div>
              <Button type="button" variant="quiet" leadingIcon={<X size={17} />} disabled={save.isPending} onClick={cancelEditing}>{t("preferences.cancel")}</Button>
              <Button type="submit" leadingIcon={<Save size={17} />} isLoading={save.isPending}>{t("preferences.save")}</Button>
            </div>
          </div>
        </form>
      )}
    </section>
  );
}

function PreferenceView({ profile, locale }: { profile: CandidateProfile; locale: string }) {
  const { t } = useTranslation();
  const missing = t("preferences.notSpecified");
  return (
    <div className="profile-section__body preference-view">
      <dl className="preference-summary">
        <PreferenceValue label={t("profile.fields.desiredSalary")} value={formatSalary(profile, locale, t)} />
        <PreferenceValue label={t("profile.fields.searchStatus")} value={t(`profile.searchStatuses.${profile.searchStatus}`)} />
      </dl>
      <PreferenceTags label={t("profile.fields.preferredLocations")} values={preferredCityNames(profile, locale)} missing={missing} />
      <PreferenceTags label={t("profile.fields.preferredRoles")} values={profile.preferredRoles} missing={missing} />
      <PreferenceTags label={t("profile.fields.preferredCategories")} values={profile.preferredCategories} missing={missing} />
      <PreferenceTags label={t("profile.fields.preferredWorkModes")} values={profile.preferredWorkModes.map((value) => t(`employer.workModes.${value}`))} missing={missing} />
      <PreferenceTags label={t("profile.fields.preferredEmploymentTypes")} values={profile.preferredEmploymentTypes.map((value) => t(`employer.employmentTypes.${value}`))} missing={missing} />
    </div>
  );
}

function PreferenceValue({ label, value }: { label: string; value: string }) {
  return <div><dt>{label}</dt><dd>{value}</dd></div>;
}
function PreferenceTags({ label, values, missing }: { label: string; values: string[]; missing: string }) {
  return (
    <div className="preference-value-group">
      <h3>{label}</h3>
      {values.length ? <ul className="preference-chips">{values.map((value) => <li key={value}>{value}</li>)}</ul> : <p>{missing}</p>}
    </div>
  );
}
function formatSalary(profile: CandidateProfile, locale: string, t: (key: string) => string) {
  if (profile.desiredSalary == null) return t("preferences.notSpecified");
  const amount = new Intl.NumberFormat(locale, { maximumFractionDigits: 2 }).format(profile.desiredSalary);
  const currency = profile.currency ? ` ${profile.currency}` : "";
  const period = profile.salaryPeriod ? ` / ${t(`preferences.periods.${profile.salaryPeriod}`)}` : "";
  return `${amount}${currency}${period}`;
}
function preferredCityNames(profile: CandidateProfile, locale: string) {
  const canonical = profile.preferredCityIds.map((id) => cityName(id, locale)).filter(Boolean);
  const legacy = profile.preferredLocations.filter(
    (value) => !kazakhstanCities.some((city) => city.names.en === value),
  );
  return [...canonical, ...legacy];
}
function toForm(profile: CandidateProfile): FormValues {
  return {
    desiredSalary: profile.desiredSalary?.toString() ?? "",
    currency: profile.currency ?? "KZT",
    salaryPeriod: profile.salaryPeriod ?? "month",
    preferredCityIds: profile.preferredCityIds,
    preferredEmploymentTypes: profile.preferredEmploymentTypes,
    preferredWorkModes: profile.preferredWorkModes,
    preferredCategories: profile.preferredCategories,
    preferredRoles: profile.preferredRoles,
    searchStatus: profile.searchStatus,
  };
}
const workModes: WorkMode[] = ["on_site", "hybrid", "remote"];
const employmentTypes: EmploymentType[] = ["full_time", "part_time", "contract", "temporary", "internship"];
