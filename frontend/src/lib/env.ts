import { z } from "zod";

const browserEnvSchema = z.object({
  VITE_API_URL: z.string().url().default("http://localhost:8080/api/v1"),
  VITE_USE_MOCKS: z.enum(["true", "false"]).default("true"),
});

const parsedEnv = browserEnvSchema.parse({
  VITE_API_URL: import.meta.env.VITE_API_URL,
  VITE_USE_MOCKS: import.meta.env.VITE_USE_MOCKS,
});

export const env = {
  apiUrl: parsedEnv.VITE_API_URL.replace(/\/$/, ""),
  useMocks: parsedEnv.VITE_USE_MOCKS === "true",
} as const;
