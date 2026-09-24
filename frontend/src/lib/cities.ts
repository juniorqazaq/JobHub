export type AppLanguage = "kk" | "ru";

export interface KazakhstanCity {
  id: string;
  names: Record<AppLanguage, string> & { en?: string };
}

export const kazakhstanCities = [
  { id: "astana", names: { kk: "Астана", ru: "Астана", en: "Astana" } },
  { id: "almaty", names: { kk: "Алматы", ru: "Алматы", en: "Almaty" } },
  { id: "shymkent", names: { kk: "Шымкент", ru: "Шымкент", en: "Shymkent" } },
  { id: "karaganda", names: { kk: "Қарағанды", ru: "Караганда", en: "Karaganda" } },
  { id: "aktobe", names: { kk: "Ақтөбе", ru: "Актобе", en: "Aktobe" } },
  { id: "taraz", names: { kk: "Тараз", ru: "Тараз", en: "Taraz" } },
  { id: "pavlodar", names: { kk: "Павлодар", ru: "Павлодар", en: "Pavlodar" } },
  { id: "oskemen", names: { kk: "Өскемен", ru: "Усть-Каменогорск", en: "Oskemen" } },
  { id: "semey", names: { kk: "Семей", ru: "Семей", en: "Semey" } },
  { id: "atyrau", names: { kk: "Атырау", ru: "Атырау", en: "Atyrau" } },
  { id: "kostanay", names: { kk: "Қостанай", ru: "Костанай", en: "Kostanay" } },
  { id: "kyzylorda", names: { kk: "Қызылорда", ru: "Кызылорда", en: "Kyzylorda" } },
  { id: "aktau", names: { kk: "Ақтау", ru: "Актау", en: "Aktau" } },
  { id: "oral", names: { kk: "Орал", ru: "Уральск", en: "Oral" } },
  { id: "petropavl", names: { kk: "Петропавл", ru: "Петропавловск", en: "Petropavl" } },
  { id: "turkistan", names: { kk: "Түркістан", ru: "Туркестан", en: "Turkistan" } },
] as const satisfies readonly KazakhstanCity[];

export type CityId = (typeof kazakhstanCities)[number]["id"];

export function isCityId(value: unknown): value is CityId {
  return typeof value === "string" && kazakhstanCities.some((city) => city.id === value);
}

export function cityName(id: string | undefined, language: string): string {
  const city = kazakhstanCities.find((item) => item.id === id);
  const locale: AppLanguage = language === "ru" ? "ru" : "kk";
  return city?.names[locale] ?? "";
}
