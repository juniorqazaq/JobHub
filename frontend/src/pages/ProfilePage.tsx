import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ExternalLink, Pencil, Plus, Save, Trash2, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Controller, useFieldArray, useForm, useWatch } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { z } from "zod";
import type {
  CandidateProfile,
  CandidateProfileInput,
  LanguageProficiency,
} from "../api/models/candidate";
import type { EmploymentType, WorkMode } from "../api/models/job";
import { repositories } from "../api/repositories";
import { useAuth } from "../app/authContext";
import { Button } from "../components/ui/Button";
import {
  Checkbox,
  Input,
  Select,
  Switch,
  Textarea,
} from "../components/ui/FormControls";
import { ErrorState, Skeleton } from "../components/ui/Feedback";
import { useToast } from "../components/ui/useToast";
import { CandidateWorkspaceLayout } from "../components/candidate/CandidateWorkspaceLayout";
import { CandidateIdentityHeader } from "../components/candidate/CandidateIdentityHeader";
import { CandidateAccountSection } from "../components/candidate/CandidateAccountSection";
import { CitySelect } from "../components/location/CitySelect";
import { cityName, isCityId } from "../lib/cities";
import {
  formatProfilePhone,
  isValidBirthDate,
  normalizeProfilePhone,
  safeProfileUrl,
} from "../lib/profileContact";

const optionalUrl = z.string().refine((value) => !value || safeProfileUrl(value));
const schema = z.object({
  fullName: z.string().trim().min(1),
  photoUrl: optionalUrl,
  cityId: z.string(),
  birthYear: z.string(),
  birthDate: z.string().refine(isValidBirthDate),
  phone: z.string().refine((value) => !value || normalizeProfilePhone(value)),
  about: z.string(),
  currentPosition: z.string(),
  desiredPosition: z.string(),
  yearsExperience: z.string(),
  experienceLevel: z.enum([
    "",
    "internship",
    "junior",
    "middle",
    "senior",
    "lead",
  ]),
  skills: z.string(),
  certifications: z.string(),
  desiredSalary: z.string(),
  currency: z.string(),
  salaryPeriod: z.enum(["", "month", "year"]),
  preferredLocations: z.string(),
  preferredCityIds: z.array(z.string()),
  preferredEmploymentTypes: z.array(z.string()),
  preferredWorkModes: z.array(z.string()),
  preferredCategories: z.string(),
  preferredRoles: z.string(),
  searchStatus: z.enum(["actively_looking", "open_to_offers", "not_looking"]),
  github: optionalUrl,
  linkedin: optionalUrl,
  portfolio: optionalUrl,
  website: optionalUrl,
  allowEmployerContact: z.boolean(),
  showProfileToEmployers: z.boolean(),
  showSalaryExpectations: z.boolean(),
  workExperience: z.array(
    z.object({
      id: z.string().optional(),
      company: z.string().trim().min(1),
      position: z.string().trim().min(1),
      employmentType: z.enum([
        "full_time",
        "part_time",
        "contract",
        "temporary",
        "internship",
      ]),
      startDate: z.string().min(1),
      endDate: z.string(),
      isCurrent: z.boolean(),
      description: z.string(),
      achievements: z.string(),
      skills: z.string(),
    }),
  ),
  education: z.array(
    z.object({
      id: z.string().optional(),
      institution: z.string().trim().min(1),
      degree: z.string().trim().min(1),
      fieldOfStudy: z.string().trim().min(1),
      startYear: z.string().min(4),
      graduationYear: z.string(),
      description: z.string(),
    }),
  ),
  languages: z.array(
    z.object({
      id: z.string().optional(),
      language: z.string().trim().min(1),
      proficiency: z.enum([
        "native",
        "fluent",
        "A1",
        "A2",
        "B1",
        "B2",
        "C1",
        "C2",
      ]),
    }),
  ),
});
type FormValues = z.infer<typeof schema>;

export function ProfilePage() {
  const { t } = useTranslation();
  const { session } = useAuth();
  const client = useQueryClient();
  const toast = useToast();
  const [isEditing, setIsEditing] = useState(false);
  const editButtonRef = useRef<HTMLButtonElement>(null);
  const profile = useQuery({
    queryKey: ["candidate-profile"],
    queryFn: () => repositories.candidate.getProfile(),
  });
  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: emptyValues(),
  });
  const experience = useFieldArray({
    control: form.control,
    name: "workExperience",
  });
  const education = useFieldArray({ control: form.control, name: "education" });
  const languages = useFieldArray({ control: form.control, name: "languages" });
  const watchedExperience = useWatch({
    control: form.control,
    name: "workExperience",
  });
  useEffect(() => {
    if (profile.data && !isEditing) form.reset(toForm(profile.data));
  }, [profile.data, form, isEditing]);
  useEffect(() => {
    if (!isEditing || !form.formState.isDirty) return;
    const warn = (event: BeforeUnloadEvent) => event.preventDefault();
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [form.formState.isDirty, isEditing]);
  const save = useMutation({
    mutationFn: (input: CandidateProfileInput) =>
      repositories.candidate.updateProfile(input, session!.csrfToken),
    onSuccess: (data) => {
      client.setQueryData(["candidate-profile"], data);
      form.reset(toForm(data));
      setIsEditing(false);
      toast.showToast({ title: t("profile.saved") });
      requestAnimationFrame(() => editButtonRef.current?.focus());
    },
    onError: () =>
      toast.showToast({ tone: "error", title: t("profile.saveError") }),
  });
  const submit = form.handleSubmit(
    (values) => {
      if (!save.isPending) save.mutate(toInput(values));
    },
    () => toast.showToast({ tone: "error", title: t("profile.fixErrors") }),
  );
  const startEditing = () => {
    form.reset(toForm(profile.data!));
    setIsEditing(true);
    requestAnimationFrame(() => form.setFocus("fullName"));
  };
  const cancelEditing = () => {
    if (form.formState.isDirty && !window.confirm(t("profile.discardConfirm"))) {
      return;
    }
    form.reset(toForm(profile.data!));
    setIsEditing(false);
    requestAnimationFrame(() => editButtonRef.current?.focus());
  };
  if (profile.isPending)
    return (
      <CandidateWorkspaceLayout>
        <section className="profile-page">
          <ProfileSkeleton label={t("profile.loading")} />
        </section>
      </CandidateWorkspaceLayout>
    );
  if (profile.isError)
    return (
      <CandidateWorkspaceLayout>
        <section className="profile-page">
          <ErrorState
            title={t("profile.loadErrorTitle")}
            description={t("profile.loadErrorDescription")}
            actionLabel={t("common.retry")}
            onAction={() => void profile.refetch()}
          />
        </section>
      </CandidateWorkspaceLayout>
    );
  return (
    <CandidateWorkspaceLayout>
      <section className="profile-page">
        <CandidateIdentityHeader
          eyebrow={t("profile.eyebrow")}
          profile={profile.data!}
          user={session!.user}
        />
        {!isEditing ? (
          <div className="profile-view-actions">
            <Button
              ref={editButtonRef}
              type="button"
              variant="secondary"
              leadingIcon={<Pencil size={17} />}
              onClick={startEditing}
            >
              {t("profile.edit")}
            </Button>
          </div>
        ) : null}
        {!isEditing ? (
          <ProfileView profile={profile.data!} />
        ) : (
        <form className="profile-form" onSubmit={submit} noValidate>
          <ProfileSection title={t("profile.sections.about")}>
            <div className="profile-grid">
              <Input
                label={t("profile.fields.fullName")}
                error={
                  form.formState.errors.fullName
                    ? t("profile.validation.required")
                    : undefined
                }
                {...form.register("fullName")}
              />
              <Controller
                control={form.control}
                name="cityId"
                render={({ field }) => (
                  <CitySelect
                    label={t("profile.fields.city")}
                    anyLabel={t("common.notProvided")}
                    value={field.value}
                    legacyLabel={profile.data?.city}
                    onChange={field.onChange}
                  />
                )}
              />
              <Input
                label={t("profile.fields.birthDate")}
                type="date"
                max={new Date().toISOString().slice(0, 10)}
                hint={
                  profile.data?.birthYear && !profile.data.birthDate
                    ? t("profile.hints.legacyBirthYear", {
                        year: profile.data.birthYear,
                      })
                    : t("profile.hints.birthDatePrivate")
                }
                error={
                  form.formState.errors.birthDate
                    ? t("profile.validation.birthDate")
                    : undefined
                }
                {...form.register("birthDate")}
              />
              <Input
                label={t("profile.fields.phone")}
                type="tel"
                placeholder="+7 700 000 00 00"
                error={
                  form.formState.errors.phone
                    ? t("profile.validation.phone")
                    : undefined
                }
                {...form.register("phone")}
              />
              <Input
                label={t("profile.fields.currentPosition")}
                {...form.register("currentPosition")}
              />
              <Input
                label={t("profile.fields.desiredPosition")}
                {...form.register("desiredPosition")}
              />
              <Input
                label={t("profile.fields.yearsExperience")}
                inputMode="decimal"
                {...form.register("yearsExperience")}
              />
              <Select
                label={t("profile.fields.experienceLevel")}
                {...form.register("experienceLevel")}
              >
                <option value="">{t("common.notProvided")}</option>
                {["internship", "junior", "middle", "senior", "lead"].map(
                  (x) => (
                    <option key={x} value={x}>
                      {t(`profile.experienceLevels.${x}`)}
                    </option>
                  ),
                )}
              </Select>
            </div>
            <Textarea
              label={t("profile.fields.about")}
              rows={5}
              {...form.register("about")}
            />
            <div className="profile-grid">
              <Input
                label={t("profile.fields.skills")}
                hint={t("profile.hints.commaSeparated")}
                {...form.register("skills")}
              />
              <Input
                label={t("profile.fields.certifications")}
                hint={t("profile.hints.commaSeparated")}
                {...form.register("certifications")}
              />
            </div>
          </ProfileSection>
          <ProfileSection
            title={t("profile.sections.experience")}
            action={
              <Button
                type="button"
                variant="secondary"
                size="sm"
                leadingIcon={<Plus size={16} />}
                onClick={() =>
                  experience.append({
                    company: "",
                    position: "",
                    employmentType: "full_time",
                    startDate: "",
                    endDate: "",
                    isCurrent: false,
                    description: "",
                    achievements: "",
                    skills: "",
                  })
                }
              >
                {t("profile.addExperience")}
              </Button>
            }
          >
            {experience.fields.length === 0 ? (
              <p className="section-empty">{t("profile.noExperience")}</p>
            ) : (
              experience.fields.map((field, index) => (
                <div className="repeatable-row" key={field.id}>
                  <div className="profile-grid">
                    <Input
                      label={t("profile.fields.company")}
                      error={
                        form.formState.errors.workExperience?.[index]?.company
                          ? t("profile.validation.required")
                          : undefined
                      }
                      {...form.register(`workExperience.${index}.company`)}
                    />
                    <Input
                      label={t("profile.fields.position")}
                      {...form.register(`workExperience.${index}.position`)}
                    />
                    <Select
                      label={t("profile.fields.employmentType")}
                      {...form.register(
                        `workExperience.${index}.employmentType`,
                      )}
                    >
                      {employmentTypes.map((x) => (
                        <option key={x} value={x}>
                          {t(`employer.employmentTypes.${x}`)}
                        </option>
                      ))}
                    </Select>
                    <Input
                      label={t("profile.fields.startDate")}
                      type="date"
                      {...form.register(`workExperience.${index}.startDate`)}
                    />
                    <Input
                      label={t("profile.fields.endDate")}
                      type="date"
                      disabled={watchedExperience?.[index]?.isCurrent}
                      {...form.register(`workExperience.${index}.endDate`)}
                    />
                    <Checkbox
                      label={t("profile.fields.currentJob")}
                      {...form.register(`workExperience.${index}.isCurrent`)}
                    />
                  </div>
                  <Textarea
                    label={t("profile.fields.description")}
                    {...form.register(`workExperience.${index}.description`)}
                  />
                  <Textarea
                    label={t("profile.fields.achievements")}
                    {...form.register(`workExperience.${index}.achievements`)}
                  />
                  <Input
                    label={t("profile.fields.skills")}
                    {...form.register(`workExperience.${index}.skills`)}
                  />
                  <Button
                    type="button"
                    variant="quiet"
                    size="sm"
                    leadingIcon={<Trash2 size={15} />}
                    onClick={() => experience.remove(index)}
                  >
                    {t("common.remove")}
                  </Button>
                </div>
              ))
            )}
          </ProfileSection>
          <ProfileSection
            title={t("profile.sections.education")}
            action={
              <Button
                type="button"
                variant="secondary"
                size="sm"
                leadingIcon={<Plus size={16} />}
                onClick={() =>
                  education.append({
                    institution: "",
                    degree: "",
                    fieldOfStudy: "",
                    startYear: "",
                    graduationYear: "",
                    description: "",
                  })
                }
              >
                {t("profile.addEducation")}
              </Button>
            }
          >
            {education.fields.length === 0 ? (
              <p className="section-empty">{t("profile.noEducation")}</p>
            ) : (
              education.fields.map((field, index) => (
                <div className="repeatable-row" key={field.id}>
                  <div className="profile-grid">
                    <Input
                      label={t("profile.fields.institution")}
                      {...form.register(`education.${index}.institution`)}
                    />
                    <Input
                      label={t("profile.fields.degree")}
                      {...form.register(`education.${index}.degree`)}
                    />
                    <Input
                      label={t("profile.fields.fieldOfStudy")}
                      {...form.register(`education.${index}.fieldOfStudy`)}
                    />
                    <Input
                      label={t("profile.fields.startYear")}
                      inputMode="numeric"
                      {...form.register(`education.${index}.startYear`)}
                    />
                    <Input
                      label={t("profile.fields.graduationYear")}
                      inputMode="numeric"
                      {...form.register(`education.${index}.graduationYear`)}
                    />
                  </div>
                  <Textarea
                    label={t("profile.fields.description")}
                    {...form.register(`education.${index}.description`)}
                  />
                  <Button
                    type="button"
                    variant="quiet"
                    size="sm"
                    leadingIcon={<Trash2 size={15} />}
                    onClick={() => education.remove(index)}
                  >
                    {t("common.remove")}
                  </Button>
                </div>
              ))
            )}
          </ProfileSection>
          <ProfileSection
            title={t("profile.sections.languages")}
            action={
              <Button
                type="button"
                variant="secondary"
                size="sm"
                leadingIcon={<Plus size={16} />}
                onClick={() =>
                  languages.append({ language: "", proficiency: "B1" })
                }
              >
                {t("profile.addLanguage")}
              </Button>
            }
          >
            {languages.fields.map((field, index) => (
              <div className="language-row" key={field.id}>
                <Input
                  label={t("profile.fields.language")}
                  {...form.register(`languages.${index}.language`)}
                />
                <Select
                  label={t("profile.fields.proficiency")}
                  {...form.register(`languages.${index}.proficiency`)}
                >
                  {languageLevels.map((x) => (
                    <option value={x} key={x}>
                      {t(`profile.proficiency.${x}`)}
                    </option>
                  ))}
                </Select>
                <Button
                  type="button"
                  variant="quiet"
                  size="sm"
                  className="language-row__remove"
                  onClick={() => languages.remove(index)}
                >
                  {t("common.remove")}
                </Button>
              </div>
            ))}
          </ProfileSection>
          <ProfileSection title={t("profile.sections.linksPrivacy")}>
            <div className="profile-grid">
              <Input
                label={t("profile.fields.github")}
                type="url"
                error={form.formState.errors.github ? t("profile.validation.url") : undefined}
                {...form.register("github")}
              />
              <Input
                label={t("profile.fields.linkedin")}
                type="url"
                error={form.formState.errors.linkedin ? t("profile.validation.url") : undefined}
                {...form.register("linkedin")}
              />
              <Input
                label={t("profile.fields.portfolio")}
                type="url"
                error={form.formState.errors.portfolio ? t("profile.validation.url") : undefined}
                {...form.register("portfolio")}
              />
              <Input
                label={t("profile.fields.website")}
                type="url"
                error={form.formState.errors.website ? t("profile.validation.url") : undefined}
                {...form.register("website")}
              />
            </div>
            <div className="privacy-list">
              <Switch
                label={t("profile.fields.allowEmployerContact")}
                {...form.register("allowEmployerContact")}
              />
              <Switch
                label={t("profile.fields.showProfileToEmployers")}
                {...form.register("showProfileToEmployers")}
              />
              <Switch
                label={t("profile.fields.showSalaryExpectations")}
                {...form.register("showSalaryExpectations")}
              />
            </div>
          </ProfileSection>
          <div className="profile-save-bar">
            <span>
              {form.formState.isDirty
                ? t("profile.unsaved")
                : t("profile.upToDate")}
            </span>
            <div className="profile-save-bar__actions">
              <Button
                type="button"
                variant="quiet"
                leadingIcon={<X size={17} />}
                disabled={save.isPending}
                onClick={cancelEditing}
              >
                {t("profile.cancel")}
              </Button>
              <Button
                type="submit"
                leadingIcon={<Save size={17} />}
                isLoading={save.isPending}
              >
                {t("profile.save")}
              </Button>
            </div>
          </div>
        </form>
        )}
        <CandidateAccountSection />
      </section>
    </CandidateWorkspaceLayout>
  );
}

function ProfileSection({
  title,
  action,
  children,
}: {
  title: string;
  action?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <section className="profile-section">
      <header>
        <h2>{title}</h2>
        {action}
      </header>
      <div className="profile-section__body">{children}</div>
    </section>
  );
}

function ProfileView({ profile }: { profile: CandidateProfile }) {
  const { t, i18n } = useTranslation();
  const empty = t("common.notProvided");
  const dateLocale = i18n.resolvedLanguage === "kk" ? "kk-KZ" : i18n.resolvedLanguage === "ru" ? "ru-RU" : "en-US";
  const birthDate = profile.birthDate
    ? new Intl.DateTimeFormat(dateLocale, { dateStyle: "long", timeZone: "UTC" }).format(
        new Date(`${profile.birthDate}T00:00:00Z`),
      )
    : profile.birthYear
      ? t("profile.legacyBirthYearValue", { year: profile.birthYear })
      : empty;

  return (
    <div className="profile-view">
      <ProfileSection title={t("profile.sections.about")}>
        <dl className="profile-view-grid">
          <ProfileFact label={t("profile.fields.fullName")} value={profile.fullName} />
          <ProfileFact label={t("profile.fields.city")} value={cityName(profile.cityId, i18n.resolvedLanguage ?? i18n.language) || profile.city || empty} />
          <ProfileFact label={t("profile.fields.birthDate")} value={birthDate} />
          <ProfileFact
            label={t("profile.fields.phone")}
            value={
              profile.phone ? (
                <a href={`tel:${profile.phone}`}>{formatProfilePhone(profile.phone)}</a>
              ) : empty
            }
          />
          <ProfileFact label={t("profile.fields.currentPosition")} value={profile.currentPosition || empty} />
          <ProfileFact label={t("profile.fields.desiredPosition")} value={profile.desiredPosition || empty} />
          <ProfileFact label={t("profile.fields.yearsExperience")} value={profile.yearsExperience?.toString() || empty} />
          <ProfileFact
            label={t("profile.fields.experienceLevel")}
            value={profile.experienceLevel ? t(`profile.experienceLevels.${profile.experienceLevel}`) : empty}
          />
        </dl>
        <div className="profile-view-copy">
          <h3>{t("profile.fields.about")}</h3>
          <p>{profile.about || empty}</p>
        </div>
        <TagGroup label={t("profile.fields.skills")} values={profile.skills} empty={empty} />
        <TagGroup label={t("profile.fields.certifications")} values={profile.certifications} empty={empty} />
      </ProfileSection>

      <ProfileSection title={t("profile.sections.experience")}>
        {profile.workExperience.length ? profile.workExperience.map((item) => (
          <article className="profile-timeline-item" key={item.id ?? `${item.company}-${item.startDate}`}>
            <div>
              <h3>{item.position}</h3>
              <p>{item.company} · {t(`employer.employmentTypes.${item.employmentType}`)}</p>
            </div>
            <span>{item.startDate} — {item.isCurrent ? t("profile.present") : item.endDate}</span>
            {item.description ? <p>{item.description}</p> : null}
            {item.achievements ? <p>{item.achievements}</p> : null}
            <TagGroup label={t("profile.fields.skills")} values={item.skills} empty={empty} compact />
          </article>
        )) : <p className="section-empty">{t("profile.noExperience")}</p>}
      </ProfileSection>

      <ProfileSection title={t("profile.sections.education")}>
        {profile.education.length ? profile.education.map((item) => (
          <article className="profile-timeline-item" key={item.id ?? `${item.institution}-${item.startYear}`}>
            <div><h3>{item.degree}, {item.fieldOfStudy}</h3><p>{item.institution}</p></div>
            <span>{item.startYear} — {item.graduationYear || t("profile.present")}</span>
            {item.description ? <p>{item.description}</p> : null}
          </article>
        )) : <p className="section-empty">{t("profile.noEducation")}</p>}
      </ProfileSection>

      <ProfileSection title={t("profile.sections.languages")}>
        {profile.languages.length ? (
          <dl className="profile-view-grid">
            {profile.languages.map((item) => (
              <ProfileFact
                key={item.id ?? item.language}
                label={item.language}
                value={t(`profile.proficiency.${item.proficiency}`)}
              />
            ))}
          </dl>
        ) : <p className="section-empty">{t("profile.noLanguages")}</p>}
      </ProfileSection>

      <ProfileSection title={t("profile.sections.linksPrivacy")}>
        <div className="profile-link-list">
          {([
            [t("profile.fields.github"), profile.github],
            [t("profile.fields.linkedin"), profile.linkedin],
            [t("profile.fields.portfolio"), profile.portfolio],
            [t("profile.fields.website"), profile.website],
          ] as const).map(([label, value]) => (
            <SafeProfileLink key={label} label={label} value={value} empty={empty} />
          ))}
        </div>
        <dl className="profile-view-grid profile-privacy-view">
          <ProfileFact label={t("profile.fields.allowEmployerContact")} value={t(profile.allowEmployerContact ? "common.yes" : "common.no")} />
          <ProfileFact label={t("profile.fields.showProfileToEmployers")} value={t(profile.showProfileToEmployers ? "common.yes" : "common.no")} />
          <ProfileFact label={t("profile.fields.showSalaryExpectations")} value={t(profile.showSalaryExpectations ? "common.yes" : "common.no")} />
        </dl>
      </ProfileSection>
    </div>
  );
}

function ProfileFact({ label, value }: { label: string; value: React.ReactNode }) {
  return <div><dt>{label}</dt><dd>{value}</dd></div>;
}

function TagGroup({ label, values, empty, compact = false }: { label: string; values: string[]; empty: string; compact?: boolean }) {
  return (
    <div className={compact ? "profile-tags profile-tags--compact" : "profile-tags"}>
      <h3>{label}</h3>
      {values.length ? <ul>{values.map((value) => <li key={value}>{value}</li>)}</ul> : <p>{empty}</p>}
    </div>
  );
}

function SafeProfileLink({ label, value, empty }: { label: string; value?: string; empty: string }) {
  const { t } = useTranslation();
  const href = safeProfileUrl(value);
  return (
    <div>
      <span>{label}</span>
      {href ? (
        <a href={href} target="_blank" rel="noopener noreferrer" aria-label={`${label}. ${t("profile.opensNewTab")}`}>
          <span>{value}</span><ExternalLink size={15} aria-hidden="true" />
        </a>
      ) : <strong>{value || empty}</strong>}
    </div>
  );
}

function emptyValues(): FormValues {
  return {
    fullName: "",
    photoUrl: "",
    cityId: "",
    birthYear: "",
    birthDate: "",
    phone: "",
    about: "",
    currentPosition: "",
    desiredPosition: "",
    yearsExperience: "",
    experienceLevel: "",
    skills: "",
    certifications: "",
    desiredSalary: "",
    currency: "KZT",
    salaryPeriod: "month",
    preferredLocations: "",
    preferredCityIds: [],
    preferredEmploymentTypes: [],
    preferredWorkModes: [],
    preferredCategories: "",
    preferredRoles: "",
    searchStatus: "actively_looking",
    github: "",
    linkedin: "",
    portfolio: "",
    website: "",
    allowEmployerContact: false,
    showProfileToEmployers: false,
    showSalaryExpectations: false,
    workExperience: [],
    education: [],
    languages: [],
  };
}
function toForm(p: CandidateProfile): FormValues {
  return {
    fullName: p.fullName,
    photoUrl: p.photoUrl ?? "",
    cityId: p.cityId ?? (p.city ? `legacy:${p.city}` : ""),
    birthYear: p.birthYear?.toString() ?? "",
    birthDate: p.birthDate ?? "",
    phone: p.phone ?? "",
    about: p.about ?? "",
    currentPosition: p.currentPosition ?? "",
    desiredPosition: p.desiredPosition ?? "",
    yearsExperience: p.yearsExperience?.toString() ?? "",
    experienceLevel: p.experienceLevel ?? "",
    skills: p.skills.join(", "),
    certifications: p.certifications.join(", "),
    desiredSalary: p.desiredSalary?.toString() ?? "",
    currency: p.currency ?? "KZT",
    salaryPeriod: p.salaryPeriod ?? "month",
    preferredLocations: p.preferredLocations.join(", "),
    preferredCityIds: p.preferredCityIds,
    preferredEmploymentTypes: p.preferredEmploymentTypes,
    preferredWorkModes: p.preferredWorkModes,
    preferredCategories: p.preferredCategories.join(", "),
    preferredRoles: p.preferredRoles.join(", "),
    searchStatus: p.searchStatus,
    github: p.github ?? "",
    linkedin: p.linkedin ?? "",
    portfolio: p.portfolio ?? "",
    website: p.website ?? "",
    allowEmployerContact: p.allowEmployerContact,
    showProfileToEmployers: p.showProfileToEmployers,
    showSalaryExpectations: p.showSalaryExpectations,
    workExperience: p.workExperience.map((x) => ({
      ...x,
      endDate: x.endDate ?? "",
      description: x.description ?? "",
      achievements: x.achievements ?? "",
      skills: x.skills.join(", "),
    })),
    education: p.education.map((x) => ({
      ...x,
      startYear: String(x.startYear),
      graduationYear: x.graduationYear?.toString() ?? "",
      description: x.description ?? "",
    })),
    languages: p.languages,
  };
}
function toInput(v: FormValues): CandidateProfileInput {
  return {
    fullName: v.fullName,
    photoUrl: v.photoUrl || undefined,
    city: isCityId(v.cityId) ? cityName(v.cityId, "en") : v.cityId.startsWith("legacy:") ? v.cityId.slice(7) : undefined,
    cityId: isCityId(v.cityId) ? v.cityId : undefined,
    birthYear: number(v.birthYear),
    birthDate: v.birthDate || undefined,
    phone: normalizeProfilePhone(v.phone),
    about: v.about || undefined,
    currentPosition: v.currentPosition || undefined,
    desiredPosition: v.desiredPosition || undefined,
    yearsExperience: number(v.yearsExperience),
    experienceLevel: v.experienceLevel || undefined,
    skills: list(v.skills),
    certifications: list(v.certifications),
    languages: v.languages,
    desiredSalary: number(v.desiredSalary),
    currency: v.currency || undefined,
    salaryPeriod: v.salaryPeriod || undefined,
    preferredLocations: list(v.preferredLocations),
    preferredCityIds: v.preferredCityIds,
    preferredEmploymentTypes: v.preferredEmploymentTypes as EmploymentType[],
    preferredWorkModes: v.preferredWorkModes as WorkMode[],
    preferredCategories: list(v.preferredCategories),
    preferredRoles: list(v.preferredRoles),
    searchStatus: v.searchStatus,
    github: v.github || undefined,
    linkedin: v.linkedin || undefined,
    portfolio: v.portfolio || undefined,
    website: v.website || undefined,
    allowEmployerContact: v.allowEmployerContact,
    showProfileToEmployers: v.showProfileToEmployers,
    showSalaryExpectations: v.showSalaryExpectations,
    workExperience: v.workExperience.map((x) => ({
      ...x,
      endDate: x.isCurrent ? undefined : x.endDate || undefined,
      description: x.description || undefined,
      achievements: x.achievements || undefined,
      skills: list(x.skills),
    })),
    education: v.education.map((x) => ({
      ...x,
      startYear: Number(x.startYear),
      graduationYear: number(x.graduationYear),
      description: x.description || undefined,
    })),
  };
}
function list(value: string) {
  return value
    .split(",")
    .map((x) => x.trim())
    .filter(Boolean);
}
function number(value: string) {
  const n = Number(value);
  return value.trim() && Number.isFinite(n) ? n : undefined;
}
function ProfileSkeleton({ label }: { label: string }) {
  return (
    <div className="profile-skeleton" role="status" aria-label={label}>
      <Skeleton className="detail-skeleton__title" />
      <Skeleton />
      <Skeleton className="detail-skeleton__body" />
    </div>
  );
}
const employmentTypes: EmploymentType[] = [
  "full_time",
  "part_time",
  "contract",
  "temporary",
  "internship",
];
const languageLevels: LanguageProficiency[] = [
  "native",
  "fluent",
  "A1",
  "A2",
  "B1",
  "B2",
  "C1",
  "C2",
];
