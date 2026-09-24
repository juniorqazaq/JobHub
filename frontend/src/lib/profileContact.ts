export function normalizeProfilePhone(value: string): string | undefined {
  const input = value.trim();
  if (!input) return undefined;
  if (!/^[+\d ()-]+$/.test(input)) return undefined;

  let normalized = input.replace(/[ ()-]/g, "");
  if (/^8[67]\d{9}$/.test(normalized)) normalized = `+7${normalized.slice(1)}`;
  if (/^[67]\d{9}$/.test(normalized)) normalized = `+7${normalized}`;
  if (!/^\+[1-9]\d{7,14}$/.test(normalized)) return undefined;
  if (normalized.startsWith("+7") && !/^\+7[67]\d{9}$/.test(normalized)) {
    return undefined;
  }
  return normalized;
}

export function formatProfilePhone(value: string): string {
  if (/^\+7\d{10}$/.test(value)) {
    return `${value.slice(0, 2)} ${value.slice(2, 5)} ${value.slice(5, 8)} ${value.slice(8, 10)} ${value.slice(10)}`;
  }
  return value;
}

export function safeProfileUrl(value?: string): string | undefined {
  if (!value) return undefined;
  try {
    const parsed = new URL(value);
    if (
      (parsed.protocol !== "http:" && parsed.protocol !== "https:") ||
      !parsed.hostname ||
      parsed.username ||
      parsed.password
    ) {
      return undefined;
    }
    return parsed.href;
  } catch {
    return undefined;
  }
}

export function isValidBirthDate(value: string): boolean {
  if (!value) return true;
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return false;
  const [year, month, day] = value.split("-").map(Number);
  const date = new Date(Date.UTC(year, month - 1, day));
  const today = new Date();
  const todayUTC = Date.UTC(today.getFullYear(), today.getMonth(), today.getDate());
  return (
    year >= 1900 &&
    date.getUTCFullYear() === year &&
    date.getUTCMonth() === month - 1 &&
    date.getUTCDate() === day &&
    date.getTime() <= todayUTC
  );
}
