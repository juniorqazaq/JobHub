import { useId } from "react";
import { useTranslation } from "react-i18next";
import { cityName, kazakhstanCities } from "../../lib/cities";

interface CitySelectProps {
  value?: string;
  onChange: (value: string) => void;
  label: string;
  anyLabel?: string;
  required?: boolean;
  error?: string;
  legacyLabel?: string;
  className?: string;
  visuallyHiddenLabel?: boolean;
}

export function CitySelect({ value = "", onChange, label, anyLabel, required, error, legacyLabel, className = "", visuallyHiddenLabel = false }: CitySelectProps) {
  const id = useId();
  const { i18n } = useTranslation();
  const unknownValue = Boolean(value) && !kazakhstanCities.some((city) => city.id === value);
  return (
    <div className={`ui-field city-select ${className}`}>
      <label className={visuallyHiddenLabel ? "ui-label visually-hidden" : "ui-label"} htmlFor={id}>{label}</label>
      <select id={id} className="ui-input ui-select" value={value} required={required} aria-invalid={Boolean(error)} aria-describedby={error ? `${id}-error` : undefined} onChange={(event) => onChange(event.target.value)}>
        <option value="">{anyLabel ?? "—"}</option>
        {unknownValue ? <option value={value}>{legacyLabel ?? value}</option> : null}
        {kazakhstanCities.map((city) => <option key={city.id} value={city.id}>{cityName(city.id, i18n.resolvedLanguage ?? i18n.language)}</option>)}
      </select>
      {error ? <span id={`${id}-error`} className="ui-field-error">{error}</span> : null}
    </div>
  );
}
