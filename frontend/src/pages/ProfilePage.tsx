import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Download, FileText, Plus, Save, Trash2, Upload } from "lucide-react";
import { useEffect, useRef } from "react";
import { useFieldArray, useForm, useWatch } from "react-hook-form";
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
import { SiteShell } from "../layouts/SiteShell";

const optionalUrl = z.union([z.literal(""), z.url()]);
const schema = z.object({
  fullName: z.string().trim().min(1),
  photoUrl: optionalUrl,
  city: z.string(),
  birthYear: z.string(),
  phone: z.string(),
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
  const { t, i18n } = useTranslation();
  const { session } = useAuth();
  const client = useQueryClient();
  const toast = useToast();
  const fileRef = useRef<HTMLInputElement>(null);
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
    if (profile.data) form.reset(toForm(profile.data));
  }, [profile.data, form]);
  const save = useMutation({
    mutationFn: (input: CandidateProfileInput) =>
      repositories.candidate.updateProfile(input, session!.csrfToken),
    onSuccess: (data) => {
      client.setQueryData(["candidate-profile"], data);
      form.reset(toForm(data));
      toast.showToast({ title: t("profile.saved") });
    },
    onError: () =>
      toast.showToast({ tone: "error", title: t("profile.saveError") }),
  });
  const upload = useMutation({
    mutationFn: (file: File) =>
      repositories.candidate.uploadResume(file, session!.csrfToken),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ["candidate-profile"] });
      toast.showToast({ title: t("resume.uploaded") });
    },
    onError: () =>
      toast.showToast({ tone: "error", title: t("resume.uploadError") }),
  });
  const remove = useMutation({
    mutationFn: () => repositories.candidate.deleteResume(session!.csrfToken),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ["candidate-profile"] });
      toast.showToast({ title: t("resume.deleted") });
    },
  });
  const submit = form.handleSubmit(
    (values) => save.mutate(toInput(values)),
    () => toast.showToast({ tone: "error", title: t("profile.fixErrors") }),
  );
  const chooseFile = (files: FileList | null) => {
    const file = files?.[0];
    if (!file) return;
    if (file.type !== "application/pdf" || file.size > 10 * 1024 * 1024) {
      toast.showToast({
        tone: "error",
        title: t(
          file.size > 10 * 1024 * 1024 ? "resume.tooLarge" : "resume.pdfOnly",
        ),
      });
      return;
    }
    upload.mutate(file);
  };
  const download = async () => {
    const blob = await repositories.candidate.downloadOwnResume();
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = profile.data?.resume?.originalFilename ?? "resume.pdf";
    link.click();
    URL.revokeObjectURL(url);
  };
  if (profile.isPending)
    return (
      <SiteShell>
        <main className="profile-page page-container">
          <ProfileSkeleton label={t("profile.loading")} />
        </main>
      </SiteShell>
    );
  if (profile.isError)
    return (
      <SiteShell>
        <main className="profile-page page-container">
          <ErrorState
            title={t("profile.loadErrorTitle")}
            description={t("profile.loadErrorDescription")}
            actionLabel={t("common.retry")}
            onAction={() => void profile.refetch()}
          />
        </main>
      </SiteShell>
    );
  return (
    <SiteShell>
      <main className="profile-page page-container">
        <header className="profile-header">
          <div className="profile-avatar" aria-hidden="true">
            {initials(profile.data!.fullName)}
          </div>
          <div>
            <p className="auth-eyebrow">{t("profile.eyebrow")}</p>
            <h1>{profile.data!.fullName}</h1>
            <p>
              {profile.data!.desiredPosition ||
                profile.data!.currentPosition ||
                t("profile.positionMissing")}
            </p>
          </div>
          <div
            className="profile-completion"
            aria-label={t("profile.completionLabel", {
              value: profile.data!.completion.percentage,
            })}
          >
            <strong>{profile.data!.completion.percentage}%</strong>
            <span>{t("profile.complete")}</span>
            <div>
              <span
                style={{ width: `${profile.data!.completion.percentage}%` }}
              />
            </div>
          </div>
        </header>
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
              <Input
                label={t("profile.fields.city")}
                {...form.register("city")}
              />
              <Input
                label={t("profile.fields.birthYear")}
                inputMode="numeric"
                {...form.register("birthYear")}
              />
              <Input
                label={t("profile.fields.phone")}
                type="tel"
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
                  onClick={() => languages.remove(index)}
                >
                  {t("common.remove")}
                </Button>
              </div>
            ))}
          </ProfileSection>
          <ProfileSection title={t("profile.sections.preferences")}>
            <div className="profile-grid">
              <Input
                label={t("profile.fields.desiredSalary")}
                inputMode="decimal"
                {...form.register("desiredSalary")}
              />
              <Input
                label={t("profile.fields.currency")}
                {...form.register("currency")}
              />
              <Select
                label={t("profile.fields.salaryPeriod")}
                {...form.register("salaryPeriod")}
              >
                <option value="">{t("common.notProvided")}</option>
                <option value="month">
                  {t("employer.salaryPeriods.month")}
                </option>
                <option value="year">{t("employer.salaryPeriods.year")}</option>
              </Select>
              <Select
                label={t("profile.fields.searchStatus")}
                {...form.register("searchStatus")}
              >
                {["actively_looking", "open_to_offers", "not_looking"].map(
                  (x) => (
                    <option key={x} value={x}>
                      {t(`profile.searchStatuses.${x}`)}
                    </option>
                  ),
                )}
              </Select>
              <Input
                label={t("profile.fields.preferredLocations")}
                {...form.register("preferredLocations")}
              />
              <Input
                label={t("profile.fields.preferredCategories")}
                {...form.register("preferredCategories")}
              />
              <Input
                label={t("profile.fields.preferredRoles")}
                {...form.register("preferredRoles")}
              />
            </div>
            <fieldset className="profile-checks">
              <legend>{t("profile.fields.preferredWorkModes")}</legend>
              {workModes.map((x) => (
                <Checkbox
                  key={x}
                  label={t(`employer.workModes.${x}`)}
                  value={x}
                  {...form.register("preferredWorkModes")}
                />
              ))}
            </fieldset>
            <fieldset className="profile-checks">
              <legend>{t("profile.fields.preferredEmploymentTypes")}</legend>
              {employmentTypes.map((x) => (
                <Checkbox
                  key={x}
                  label={t(`employer.employmentTypes.${x}`)}
                  value={x}
                  {...form.register("preferredEmploymentTypes")}
                />
              ))}
            </fieldset>
          </ProfileSection>
          <ProfileSection title={t("profile.sections.linksPrivacy")}>
            <div className="profile-grid">
              <Input
                label={t("profile.fields.github")}
                type="url"
                {...form.register("github")}
              />
              <Input
                label={t("profile.fields.linkedin")}
                type="url"
                {...form.register("linkedin")}
              />
              <Input
                label={t("profile.fields.portfolio")}
                type="url"
                {...form.register("portfolio")}
              />
              <Input
                label={t("profile.fields.website")}
                type="url"
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
          <ProfileSection title={t("profile.sections.resume")}>
            <input
              ref={fileRef}
              className="visually-hidden"
              type="file"
              accept="application/pdf,.pdf"
              onChange={(event) => chooseFile(event.target.files)}
            />
            {profile.data!.resume ? (
              <div className="resume-row">
                <FileText size={26} />
                <div>
                  <strong>{profile.data!.resume.originalFilename}</strong>
                  <span>
                    {formatBytes(profile.data!.resume.sizeBytes)} ·{" "}
                    {formatDate(profile.data!.resume.uploadedAt, i18n.language)}
                  </span>
                </div>
                <div>
                  <Button
                    type="button"
                    variant="secondary"
                    size="sm"
                    leadingIcon={<Download size={15} />}
                    onClick={() => void download()}
                  >
                    {t("resume.download")}
                  </Button>
                  <Button
                    type="button"
                    variant="secondary"
                    size="sm"
                    leadingIcon={<Upload size={15} />}
                    onClick={() => fileRef.current?.click()}
                  >
                    {t("resume.replace")}
                  </Button>
                  <Button
                    type="button"
                    variant="danger"
                    size="sm"
                    leadingIcon={<Trash2 size={15} />}
                    onClick={() =>
                      window.confirm(t("resume.deleteConfirm")) &&
                      remove.mutate()
                    }
                  >
                    {t("resume.delete")}
                  </Button>
                </div>
              </div>
            ) : (
              <div className="resume-upload">
                <FileText size={28} />
                <div>
                  <strong>{t("resume.emptyTitle")}</strong>
                  <p>{t("resume.emptyDescription")}</p>
                </div>
                <Button
                  type="button"
                  leadingIcon={<Upload size={17} />}
                  isLoading={upload.isPending}
                  onClick={() => fileRef.current?.click()}
                >
                  {t("resume.upload")}
                </Button>
              </div>
            )}
          </ProfileSection>
          <div className="profile-save-bar">
            <span>
              {form.formState.isDirty
                ? t("profile.unsaved")
                : t("profile.upToDate")}
            </span>
            <Button
              type="submit"
              leadingIcon={<Save size={17} />}
              isLoading={save.isPending}
            >
              {t("profile.save")}
            </Button>
          </div>
        </form>
      </main>
    </SiteShell>
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
function emptyValues(): FormValues {
  return {
    fullName: "",
    photoUrl: "",
    city: "",
    birthYear: "",
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
    city: p.city ?? "",
    birthYear: p.birthYear?.toString() ?? "",
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
    city: v.city || undefined,
    birthYear: number(v.birthYear),
    phone: v.phone || undefined,
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
function initials(value: string) {
  return value
    .split(/\s+/)
    .slice(0, 2)
    .map((x) => x[0])
    .join("")
    .toUpperCase();
}
function formatBytes(value: number) {
  return `${(value / 1024 / 1024).toFixed(1)} MB`;
}
function formatDate(value: string, language: string) {
  return new Intl.DateTimeFormat(language, { dateStyle: "medium" }).format(
    new Date(value),
  );
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
const workModes: WorkMode[] = ["on_site", "hybrid", "remote"];
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
