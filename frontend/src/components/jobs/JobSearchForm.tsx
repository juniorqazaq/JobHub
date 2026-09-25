import { zodResolver } from "@hookform/resolvers/zod";
import { Search } from "lucide-react";
import { useEffect, useMemo } from "react";
import { Controller, useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { z } from "zod";
import { Button } from "../ui/Button";
import { CitySelect } from "../location/CitySelect";

export interface JobSearchValues {
  query: string;
  city: string;
}

interface JobSearchFormProps {
  initialValues?: Partial<JobSearchValues>;
  onSubmit: (values: JobSearchValues) => void;
  variant?: "hero" | "compact" | "stacked";
  showCity?: boolean;
}

export function JobSearchForm({ initialValues, onSubmit, variant = "compact", showCity = true }: JobSearchFormProps) {
  const { t } = useTranslation();
  const schema = useMemo(() => z.object({
    query: z.string().max(120, t("search.queryTooLong")),
    city: z.string(),
  }), [t]);
  const { control, register, handleSubmit, reset, formState: { errors } } = useForm<JobSearchValues>({
    resolver: zodResolver(schema),
    defaultValues: { query: initialValues?.query ?? "", city: initialValues?.city ?? "" },
  });

  useEffect(() => {
    reset({ query: initialValues?.query ?? "", city: initialValues?.city ?? "" });
  }, [initialValues?.city, initialValues?.query, reset]);

  return (
    <form className={`market-search market-search--${variant}${showCity ? "" : " market-search--without-city"}`} onSubmit={handleSubmit((values) => onSubmit({
      query: values.query.trim(),
      city: values.city,
    }))} noValidate>
      <label className="market-search__field">
        <span>{t("search.queryLabel")}</span>
        <span className="market-search__control">
          <Search size={20} aria-hidden="true" />
          <input placeholder={t("search.queryPlaceholder")} aria-invalid={Boolean(errors.query)} {...register("query")} />
        </span>
        {errors.query ? <small role="alert">{errors.query.message}</small> : null}
      </label>
      {showCity ? <Controller control={control} name="city" render={({ field }) => <CitySelect className="market-search__field" label={t("search.locationLabel")} anyLabel={t("jobs.anyLocation")} value={field.value} onChange={field.onChange} visuallyHiddenLabel={variant === "hero"} />} /> : null}
      <Button className="market-search__submit" size={variant === "hero" ? "lg" : "md"} type="submit" leadingIcon={<Search size={18} aria-hidden="true" />}>
        {t("search.submit")}
      </Button>
    </form>
  );
}
