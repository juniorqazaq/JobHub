import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import axios from "axios";
import { ArrowLeft, Send, Save } from "lucide-react";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { Link, useNavigate, useParams } from "react-router-dom";
import { z } from "zod";
import type { NativeJobInput } from "../api/models/job";
import { repositories } from "../api/repositories";
import { useAuth } from "../app/authContext";
import { EmployerWorkspaceNav } from "../components/employer/EmployerWorkspaceNav";
import { Button } from "../components/ui/Button";
import { ErrorState } from "../components/ui/Feedback";
import {
  Checkbox,
  Input,
  Select,
  Textarea,
} from "../components/ui/FormControls";
import { useToast } from "../components/ui/useToast";
import { SiteShell } from "../layouts/SiteShell";

const limits = {
  title: 200,
  category: 120,
  description: 20000,
  listText: 12000,
  location: 300,
  skill: 100,
  benefit: 500,
  items: 50,
} as const;
const nonnegativeNumber = z
  .string()
  .refine(
    (value) =>
      value === "" || (Number.isFinite(Number(value)) && Number(value) >= 0),
    "nonnegative",
  );
const schema = z
  .object({
    title: z.string().trim().min(2, "required").max(limits.title, "tooLong"),
    category: z
      .string()
      .trim()
      .min(2, "required")
      .max(limits.category, "tooLong"),
    description: z
      .string()
      .trim()
      .min(20, "required")
      .max(limits.description, "tooLong"),
    responsibilities: z
      .string()
      .trim()
      .min(10, "required")
      .max(limits.listText, "tooLong"),
    requirements: z
      .string()
      .trim()
      .min(10, "required")
      .max(limits.listText, "tooLong"),
    niceToHave: z.string().max(limits.listText, "tooLong"),
    skills: z.string().refine(validCommaList, "invalidList"),
    location: z
      .string()
      .trim()
      .min(2, "required")
      .max(limits.location, "tooLong"),
    workMode: z.enum(["on_site", "hybrid", "remote"]),
    employmentType: z.enum([
      "full_time",
      "part_time",
      "contract",
      "temporary",
      "internship",
    ]),
    experienceLevel: z.enum([
      "no_experience",
      "junior",
      "middle",
      "senior",
      "lead",
    ]),
    salaryMin: nonnegativeNumber,
    salaryMax: nonnegativeNumber,
    salaryCurrency: z
      .string()
      .trim()
      .regex(/^[A-Za-z]{3}$/, "currency"),
    salaryPeriod: z.enum(["month", "year"]),
    salaryVisible: z.boolean(),
    benefits: z.string().refine(validLineList, "invalidList"),
    expiresAt: z.string(),
  })
  .refine(
    (value) =>
      !value.salaryMin ||
      !value.salaryMax ||
      Number(value.salaryMax) >= Number(value.salaryMin),
    { path: ["salaryMax"], message: "range" },
  );
type FormValues = z.infer<typeof schema>;
const defaults: FormValues = {
  title: "",
  category: "",
  description: "",
  responsibilities: "",
  requirements: "",
  niceToHave: "",
  skills: "",
  location: "",
  workMode: "on_site",
  employmentType: "full_time",
  experienceLevel: "middle",
  salaryMin: "",
  salaryMax: "",
  salaryCurrency: "KZT",
  salaryPeriod: "month",
  salaryVisible: true,
  benefits: "",
  expiresAt: "",
};

export function EmployerVacancyFormPage() {
  const { t } = useTranslation();
  const { jobId } = useParams();
  const editing = Boolean(jobId);
  const { session } = useAuth();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const toast = useToast();
  const [submitError, setSubmitError] = useState("");
  const [intent, setIntent] = useState<"draft" | "published">("draft");
  const job = useQuery({
    queryKey: ["employer", "jobs", jobId],
    queryFn: () => repositories.employerJobs.getById(jobId!),
    enabled: editing,
  });
  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: defaults,
  });
  useEffect(() => {
    if (!job.data) return;
    form.reset({
      title: job.data.title,
      category: job.data.category || "",
      description: job.data.summary,
      responsibilities: job.data.responsibilities || "",
      requirements: job.data.requirements || "",
      niceToHave: job.data.niceToHave || "",
      skills: job.data.tags.join(", "),
      location: job.data.location,
      workMode: job.data.workMode || "on_site",
      employmentType:
        job.data.employmentType === "part_time" ||
        job.data.employmentType === "contract" ||
        job.data.employmentType === "temporary" ||
        job.data.employmentType === "internship"
          ? job.data.employmentType
          : "full_time",
      experienceLevel: job.data.experienceLevel || "middle",
      salaryMin: job.data.salary?.min?.toString() || "",
      salaryMax: job.data.salary?.max?.toString() || "",
      salaryCurrency: job.data.salary?.currency || "KZT",
      salaryPeriod: job.data.salary?.period || "month",
      salaryVisible: Boolean(job.data.salary),
      benefits: job.data.benefits?.join("\n") || "",
      expiresAt: job.data.expiresAt?.slice(0, 10) || "",
    });
  }, [form, job.data]);
  const save = useMutation({
    mutationFn: async ({
      values,
      publish,
    }: {
      values: FormValues;
      publish: boolean;
    }) => {
      const input = toInput(values);
      const saved = editing
        ? await repositories.employerJobs.update(
            jobId!,
            input,
            session!.csrfToken,
          )
        : await repositories.employerJobs.create(input, session!.csrfToken);
      return publish && saved.publicationStatus !== "published"
        ? repositories.employerJobs.transition(
            saved.id,
            "published",
            session!.csrfToken,
          )
        : saved;
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["employer", "jobs"] });
      await queryClient.invalidateQueries({ queryKey: ["jobs"] });
      toast.showToast({
        title:
          intent === "published"
            ? t("employer.publishedSuccess")
            : t("employer.savedSuccess"),
      });
      navigate("/employer/vacancies");
    },
  });
  const submit = form.handleSubmit(async (values) => {
    setSubmitError("");
    try {
      await save.mutateAsync({ values, publish: intent === "published" });
    } catch (error) {
      const fields = backendFields(error);
      Object.entries(fields).forEach(([name, message]) =>
        form.setError(name as keyof FormValues, { message }),
      );
      setSubmitError(
        Object.keys(fields).length
          ? t("employer.fixErrors")
          : t("employer.saveError"),
      );
    }
  });
  if (job.isError)
    return (
      <SiteShell>
        <main className="employer-page page-container">
          <ErrorState
            title={t("employer.loadErrorTitle")}
            description={t("employer.loadErrorDescription")}
            actionLabel={t("common.retry")}
            onAction={() => void job.refetch()}
          />
        </main>
      </SiteShell>
    );
  return (
    <SiteShell>
      <main className="employer-page page-container">
        <EmployerWorkspaceNav />
        <Link className="back-link" to="/employer/vacancies">
          <ArrowLeft size={17} />
          {t("employer.backToVacancies")}
        </Link>
        <header className="vacancy-form-header">
          <p className="auth-eyebrow">{t("auth.employerEyebrow")}</p>
          <h1>
            {editing ? t("employer.editVacancy") : t("employer.newVacancy")}
          </h1>
          <p>{t("employer.formDescription")}</p>
        </header>
        {editing && job.isPending ? (
          <p role="status">{t("employer.loadingVacancy")}</p>
        ) : (
          <form className="vacancy-form" onSubmit={submit} noValidate>
            <section>
              <h2>{t("employer.sections.basics")}</h2>
              <div className="vacancy-form__grid">
                <Input
                  maxLength={limits.title}
                  label={t("employer.fields.title")}
                  error={fieldError(form.formState.errors.title?.message, t)}
                  {...form.register("title")}
                />
                <Input
                  maxLength={limits.category}
                  label={t("employer.fields.category")}
                  error={fieldError(form.formState.errors.category?.message, t)}
                  {...form.register("category")}
                />
                <Input
                  maxLength={limits.location}
                  label={t("employer.fields.location")}
                  error={fieldError(form.formState.errors.location?.message, t)}
                  {...form.register("location")}
                />
                <Select
                  label={t("employer.fields.workMode")}
                  {...form.register("workMode")}
                >
                  {options("workModes", ["on_site", "hybrid", "remote"], t)}
                </Select>
                <Select
                  label={t("employer.fields.employmentType")}
                  {...form.register("employmentType")}
                >
                  {options(
                    "employmentTypes",
                    [
                      "full_time",
                      "part_time",
                      "contract",
                      "temporary",
                      "internship",
                    ],
                    t,
                  )}
                </Select>
                <Select
                  label={t("employer.fields.experienceLevel")}
                  {...form.register("experienceLevel")}
                >
                  {options(
                    "experienceLevels",
                    ["no_experience", "junior", "middle", "senior", "lead"],
                    t,
                  )}
                </Select>
              </div>
            </section>
            <section>
              <h2>{t("employer.sections.description")}</h2>
              <Textarea
                maxLength={limits.description}
                rows={7}
                label={t("employer.fields.description")}
                error={fieldError(
                  form.formState.errors.description?.message,
                  t,
                )}
                {...form.register("description")}
              />
              <Textarea
                maxLength={limits.listText}
                rows={5}
                label={t("employer.fields.responsibilities")}
                hint={t("employer.hints.onePerLine")}
                error={fieldError(
                  form.formState.errors.responsibilities?.message,
                  t,
                )}
                {...form.register("responsibilities")}
              />
              <Textarea
                maxLength={limits.listText}
                rows={5}
                label={t("employer.fields.requirements")}
                hint={t("employer.hints.onePerLine")}
                error={fieldError(
                  form.formState.errors.requirements?.message,
                  t,
                )}
                {...form.register("requirements")}
              />
              <Textarea
                maxLength={limits.listText}
                rows={4}
                label={t("employer.fields.niceToHave")}
                hint={t("employer.hints.onePerLine")}
                error={fieldError(form.formState.errors.niceToHave?.message, t)}
                {...form.register("niceToHave")}
              />
            </section>
            <section>
              <h2>{t("employer.sections.details")}</h2>
              <Input
                label={t("employer.fields.skills")}
                hint={t("employer.hints.skills")}
                error={fieldError(form.formState.errors.skills?.message, t)}
                {...form.register("skills")}
              />
              <Textarea
                rows={3}
                label={t("employer.fields.benefits")}
                hint={t("employer.hints.onePerLine")}
                error={fieldError(form.formState.errors.benefits?.message, t)}
                {...form.register("benefits")}
              />
              <div className="vacancy-form__grid">
                <Input
                  type="number"
                  min="0"
                  label={t("employer.fields.salaryMin")}
                  error={fieldError(
                    form.formState.errors.salaryMin?.message,
                    t,
                  )}
                  {...form.register("salaryMin")}
                />
                <Input
                  type="number"
                  min="0"
                  label={t("employer.fields.salaryMax")}
                  error={fieldError(
                    form.formState.errors.salaryMax?.message,
                    t,
                  )}
                  {...form.register("salaryMax")}
                />
                <Select
                  label={t("employer.fields.currency")}
                  error={fieldError(
                    form.formState.errors.salaryCurrency?.message,
                    t,
                  )}
                  {...form.register("salaryCurrency")}
                >
                  <option value="KZT">KZT</option>
                  <option value="USD">USD</option>
                  <option value="EUR">EUR</option>
                </Select>
                <Select
                  label={t("employer.fields.salaryPeriod")}
                  {...form.register("salaryPeriod")}
                >
                  <option value="month">
                    {t("employer.salaryPeriods.month")}
                  </option>
                  <option value="year">
                    {t("employer.salaryPeriods.year")}
                  </option>
                </Select>
                <Input
                  type="date"
                  label={t("employer.fields.expiresAt")}
                  {...form.register("expiresAt")}
                />
              </div>
              <Checkbox
                label={t("employer.fields.salaryVisible")}
                {...form.register("salaryVisible")}
              />
            </section>
            {submitError ? (
              <p className="auth-error" role="alert">
                {submitError}
              </p>
            ) : null}
            <div className="vacancy-form__actions">
              <Button
                type="submit"
                variant="secondary"
                isLoading={save.isPending && intent === "draft"}
                leadingIcon={<Save size={17} />}
                onClick={() => setIntent("draft")}
              >
                {t("employer.saveDraft")}
              </Button>
              <Button
                type="submit"
                isLoading={save.isPending && intent === "published"}
                leadingIcon={<Send size={17} />}
                onClick={() => setIntent("published")}
              >
                {t("employer.saveAndPublish")}
              </Button>
            </div>
          </form>
        )}
      </main>
    </SiteShell>
  );
}

function toInput(v: FormValues): NativeJobInput {
  return {
    title: v.title,
    category: v.category,
    description: v.description,
    responsibilities: v.responsibilities,
    requirements: v.requirements,
    niceToHave: v.niceToHave || undefined,
    skills: split(v.skills, /,/),
    location: v.location,
    workMode: v.workMode,
    employmentType: v.employmentType,
    experienceLevel: v.experienceLevel,
    salaryMin: numberOrUndefined(v.salaryMin),
    salaryMax: numberOrUndefined(v.salaryMax),
    salaryCurrency: v.salaryCurrency || undefined,
    salaryPeriod: v.salaryPeriod || undefined,
    salaryVisible: v.salaryVisible,
    benefits: split(v.benefits, /\n/),
    expiresAt: v.expiresAt
      ? new Date(`${v.expiresAt}T23:59:59`).toISOString()
      : undefined,
  };
}
function split(value: string, pattern: RegExp) {
  return value
    .split(pattern)
    .map((item) => item.trim())
    .filter(Boolean);
}
function numberOrUndefined(value: string) {
  const number = Number(value);
  return value === "" || Number.isNaN(number) ? undefined : number;
}
function backendFields(error: unknown): Record<string, string> {
  if (!axios.isAxiosError(error)) return {};
  return (
    (
      error.response?.data as
        | { error?: { fields?: Record<string, string> } }
        | undefined
    )?.error?.fields ?? {}
  );
}
function fieldError(message: string | undefined, t: (key: string) => string) {
  if (!message) return undefined;
  const key =
    message === "range" || message.includes("greater than")
      ? "salaryRange"
      : message === "nonnegative" || message.includes("non-negative")
        ? "nonnegative"
        : message === "tooLong" || message.includes("too long")
          ? "tooLong"
          : message === "invalidList" || message.includes("entries")
            ? "invalidList"
            : message === "currency" || message.includes("currency")
              ? "currency"
              : "required";
  return t(`employer.validation.${key}`);
}
function options(group: string, values: string[], t: (key: string) => string) {
  return values.map((value) => (
    <option key={value} value={value}>
      {t(`employer.${group}.${value}`)}
    </option>
  ));
}
function validCommaList(value: string) {
  if (!value.trim()) return true;
  const entries = value.split(",");
  return (
    entries.length <= limits.items &&
    entries.every(
      (entry) =>
        entry.trim().length > 0 && [...entry.trim()].length <= limits.skill,
    )
  );
}
function validLineList(value: string) {
  if (!value.trim()) return true;
  const entries = value.split(/\r?\n/);
  return (
    entries.length <= limits.items &&
    entries.every(
      (entry) =>
        entry.trim().length > 0 && [...entry.trim()].length <= limits.benefit,
    )
  );
}
