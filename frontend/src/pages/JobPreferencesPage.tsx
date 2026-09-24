import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Save } from "lucide-react";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { z } from "zod";
import type { EmploymentType, WorkMode } from "../api/models/job";
import { repositories } from "../api/repositories";
import { useAuth } from "../app/authContext";
import { CandidateWorkspaceLayout } from "../components/candidate/CandidateWorkspaceLayout";
import { Button } from "../components/ui/Button";
import { Checkbox, Input, Select } from "../components/ui/FormControls";
import { ErrorState, Skeleton } from "../components/ui/Feedback";
import { useToast } from "../components/ui/useToast";
import { editableProfile } from "../lib/candidateProfile";

const schema = z.object({
  desiredSalary: z
    .string()
    .refine(
      (value) =>
        value === "" || (Number.isFinite(Number(value)) && Number(value) >= 0),
      "nonnegative",
    ),
  currency: z
    .string()
    .trim()
    .regex(/^[A-Za-z]{3}$/),
  salaryPeriod: z.enum(["month", "year"]),
  preferredLocations: z.string(),
  preferredEmploymentTypes: z.array(z.string()),
  preferredWorkModes: z.array(z.string()),
  preferredCategories: z.string(),
  preferredRoles: z.string(),
  searchStatus: z.enum(["actively_looking", "open_to_offers", "not_looking"]),
});
type FormValues = z.infer<typeof schema>;

export function JobPreferencesPage() {
  const { t } = useTranslation();
  const { session } = useAuth();
  const client = useQueryClient();
  const toast = useToast();
  const profile = useQuery({
    queryKey: ["candidate-profile"],
    queryFn: () => repositories.candidate.getProfile(),
  });
  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      desiredSalary: "",
      currency: "KZT",
      salaryPeriod: "month",
      preferredLocations: "",
      preferredEmploymentTypes: [],
      preferredWorkModes: [],
      preferredCategories: "",
      preferredRoles: "",
      searchStatus: "actively_looking",
    },
  });
  useEffect(() => {
    if (!profile.data) return;
    form.reset({
      desiredSalary: profile.data.desiredSalary?.toString() ?? "",
      currency: profile.data.currency ?? "KZT",
      salaryPeriod: profile.data.salaryPeriod ?? "month",
      preferredLocations: profile.data.preferredLocations.join(", "),
      preferredEmploymentTypes: profile.data.preferredEmploymentTypes,
      preferredWorkModes: profile.data.preferredWorkModes,
      preferredCategories: profile.data.preferredCategories.join(", "),
      preferredRoles: profile.data.preferredRoles.join(", "),
      searchStatus: profile.data.searchStatus,
    });
  }, [form, profile.data]);
  const save = useMutation({
    mutationFn: (values: FormValues) =>
      repositories.candidate.updateProfile(
        {
          ...editableProfile(profile.data!),
          desiredSalary:
            values.desiredSalary === ""
              ? undefined
              : Number(values.desiredSalary),
          currency: values.currency.toUpperCase(),
          salaryPeriod: values.salaryPeriod,
          preferredLocations: list(values.preferredLocations),
          preferredEmploymentTypes:
            values.preferredEmploymentTypes as EmploymentType[],
          preferredWorkModes: values.preferredWorkModes as WorkMode[],
          preferredCategories: list(values.preferredCategories),
          preferredRoles: list(values.preferredRoles),
          searchStatus: values.searchStatus,
        },
        session!.csrfToken,
      ),
    onSuccess: (data) => {
      client.setQueryData(["candidate-profile"], data);
      toast.showToast({ title: t("profile.saved") });
      form.reset({
        ...form.getValues(),
        desiredSalary: data.desiredSalary?.toString() ?? "",
        currency: data.currency ?? "KZT",
      });
    },
    onError: () =>
      toast.showToast({ tone: "error", title: t("profile.saveError") }),
  });

  return (
    <CandidateWorkspaceLayout>
      <section
        className="workspace-page"
        aria-labelledby="preferences-page-title"
      >
        <header className="workspace-page__header">
          <p className="auth-eyebrow">{t("workspace.title")}</p>
          <h1 id="preferences-page-title">{t("workspace.nav.preferences")}</h1>
          <p>{t("workspace.preferencesDescription")}</p>
        </header>
        {profile.isPending ? (
          <Skeleton className="detail-skeleton__body" />
        ) : null}
        {profile.isError ? (
          <ErrorState
            title={t("profile.loadErrorTitle")}
            description={t("profile.loadErrorDescription")}
            actionLabel={t("common.retry")}
            onAction={() => void profile.refetch()}
          />
        ) : null}
        {profile.data ? (
          <form
            className="profile-form"
            onSubmit={form.handleSubmit((values) => save.mutate(values))}
            noValidate
          >
            <section className="profile-section">
              <div className="profile-section__body">
                <div className="profile-grid">
                  <Input
                    label={t("profile.fields.desiredSalary")}
                    type="number"
                    min="0"
                    error={
                      form.formState.errors.desiredSalary
                        ? t("profile.validation.nonnegative")
                        : undefined
                    }
                    {...form.register("desiredSalary")}
                  />
                  <Input
                    label={t("profile.fields.currency")}
                    maxLength={3}
                    error={
                      form.formState.errors.currency
                        ? t("profile.validation.currency")
                        : undefined
                    }
                    {...form.register("currency")}
                  />
                  <Select
                    label={t("profile.fields.salaryPeriod")}
                    {...form.register("salaryPeriod")}
                  >
                    <option value="month">
                      {t("employer.salaryPeriods.month")}
                    </option>
                    <option value="year">
                      {t("employer.salaryPeriods.year")}
                    </option>
                  </Select>
                  <Select
                    label={t("profile.fields.searchStatus")}
                    {...form.register("searchStatus")}
                  >
                    {["actively_looking", "open_to_offers", "not_looking"].map(
                      (value) => (
                        <option key={value} value={value}>
                          {t(`profile.searchStatuses.${value}`)}
                        </option>
                      ),
                    )}
                  </Select>
                  <Input
                    label={t("profile.fields.preferredLocations")}
                    hint={t("profile.hints.commaSeparated")}
                    {...form.register("preferredLocations")}
                  />
                  <Input
                    label={t("profile.fields.preferredCategories")}
                    hint={t("profile.hints.commaSeparated")}
                    {...form.register("preferredCategories")}
                  />
                  <Input
                    label={t("profile.fields.preferredRoles")}
                    hint={t("profile.hints.commaSeparated")}
                    {...form.register("preferredRoles")}
                  />
                </div>
                <fieldset className="profile-checks">
                  <legend>{t("profile.fields.preferredWorkModes")}</legend>
                  {workModes.map((value) => (
                    <Checkbox
                      key={value}
                      label={t(`employer.workModes.${value}`)}
                      value={value}
                      {...form.register("preferredWorkModes")}
                    />
                  ))}
                </fieldset>
                <fieldset className="profile-checks">
                  <legend>
                    {t("profile.fields.preferredEmploymentTypes")}
                  </legend>
                  {employmentTypes.map((value) => (
                    <Checkbox
                      key={value}
                      label={t(`employer.employmentTypes.${value}`)}
                      value={value}
                      {...form.register("preferredEmploymentTypes")}
                    />
                  ))}
                </fieldset>
              </div>
            </section>
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
        ) : null}
      </section>
    </CandidateWorkspaceLayout>
  );
}

function list(value: string) {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}
const workModes: WorkMode[] = ["on_site", "hybrid", "remote"];
const employmentTypes: EmploymentType[] = [
  "full_time",
  "part_time",
  "contract",
  "temporary",
  "internship",
];
