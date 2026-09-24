import type { SalaryRange } from "../api/models/job";

export function formatSalary(
  salary: SalaryRange,
  locale: string,
  periodLabel: string,
) {
  const number = new Intl.NumberFormat(locale, { maximumFractionDigits: 2 });
  const range =
    salary.max == null
      ? number.format(salary.min)
      : `${number.format(salary.min)}–${number.format(salary.max)}`;
  return `${range} ${salary.currency} / ${periodLabel}`;
}
