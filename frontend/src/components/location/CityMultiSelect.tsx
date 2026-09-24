import { X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cityName, kazakhstanCities } from "../../lib/cities";
import { CitySelect } from "./CitySelect";

interface Props { value: string[]; onChange: (value: string[]) => void; label: string; addLabel: string; removeLabel: (name: string) => string; legacyValues?: string[]; }

export function CityMultiSelect({ value, onChange, label, addLabel, removeLabel, legacyValues = [] }: Props) {
  const { i18n } = useTranslation();
  const language = i18n.resolvedLanguage ?? i18n.language;
  return (
    <div className="city-multi-select">
      <CitySelect value="" label={label} anyLabel={addLabel} onChange={(id) => { if (id && !value.includes(id)) onChange([...value, id]); }} />
      {value.length || legacyValues.length ? <ul className="preference-chips">
        {value.map((id) => { const name = cityName(id, language) || id; return <li key={id}>{name}<button type="button" aria-label={removeLabel(name)} onClick={() => onChange(value.filter((item) => item !== id))}><X size={14} /></button></li>; })}
        {legacyValues.filter((legacy) => !kazakhstanCities.some((city) => city.names.kk === legacy || city.names.ru === legacy || city.names.en === legacy)).map((legacy) => <li key={`legacy-${legacy}`} className="is-legacy">{legacy}</li>)}
      </ul> : null}
    </div>
  );
}
