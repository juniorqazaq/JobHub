import { zodResolver } from "@hookform/resolvers/zod";
import { MapPin, Search } from "lucide-react";
import { useEffect, useMemo } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { z } from "zod";
import { Button } from "../ui/Button";

export interface JobSearchValues {
  query: string;
  location: string;
}

interface JobSearchFormProps {
  initialValues?: Partial<JobSearchValues>;
  onSubmit: (values: JobSearchValues) => void;
  variant?: "hero" | "compact" | "stacked";
}

export function JobSearchForm({ initialValues, onSubmit, variant = "compact" }: JobSearchFormProps) {
  const { t } = useTranslation();
  const schema = useMemo(() => z.object({
    query: z.string().max(120, t("search.queryTooLong")),
    location: z.string().max(80, t("search.locationTooLong")),
  }), [t]);
  const { register, handleSubmit, reset, formState: { errors } } = useForm<JobSearchValues>({
    resolver: zodResolver(schema),
    defaultValues: { query: initialValues?.query ?? "", location: initialValues?.location ?? "" },
  });

  useEffect(() => {
    reset({ query: initialValues?.query ?? "", location: initialValues?.location ?? "" });
  }, [initialValues?.location, initialValues?.query, reset]);

  return (
    <form className={`market-search market-search--${variant}`} onSubmit={handleSubmit((values) => onSubmit({
      query: values.query.trim(),
      location: values.location.trim(),
    }))} noValidate>
      <label className="market-search__field">
        <span>{t("search.queryLabel")}</span>
        <span className="market-search__control">
          <Search size={20} aria-hidden="true" />
          <input placeholder={t("search.queryPlaceholder")} aria-invalid={Boolean(errors.query)} {...register("query")} />
        </span>
        {errors.query ? <small role="alert">{errors.query.message}</small> : null}
      </label>
      <label className="market-search__field">
        <span>{t("search.locationLabel")}</span>
        <span className="market-search__control">
          <MapPin size={20} aria-hidden="true" />
          <input placeholder={t("search.locationPlaceholder")} aria-invalid={Boolean(errors.location)} {...register("location")} />
        </span>
        {errors.location ? <small role="alert">{errors.location.message}</small> : null}
      </label>
      <Button className="market-search__submit" size={variant === "hero" ? "lg" : "md"} type="submit" leadingIcon={<Search size={18} aria-hidden="true" />}>
        {t("search.submit")}
      </Button>
    </form>
  );
}
