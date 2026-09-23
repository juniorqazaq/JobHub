import axios from "axios";
import { env } from "../lib/env";

export const apiClient = axios.create({
  baseURL: env.apiUrl,
  timeout: 8_000,
  headers: { Accept: "application/json" },
});
