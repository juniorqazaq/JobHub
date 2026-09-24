export function safeReturnTo(value: string | null, fallback = "/account") {
  if (!value || !value.startsWith("/") || value.startsWith("//")) return fallback;
  return value;
}
