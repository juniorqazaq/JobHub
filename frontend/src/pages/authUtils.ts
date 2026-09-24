import axios from "axios";

export function authErrorCode(error: unknown): string {
  if (axios.isAxiosError(error)) {
    const code = (error.response?.data as { error?: { code?: string } } | undefined)?.error?.code;
    if (code) return code;
  }
  return "UNKNOWN";
}
