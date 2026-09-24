import axios from "axios";
import { env } from "../lib/env";

export const apiClient = axios.create({
  baseURL: env.apiUrl,
  timeout: 30_000,
  withCredentials: true,
  headers: { Accept: "application/json" },
});
