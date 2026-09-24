import { QueryClientProvider } from "@tanstack/react-query";
import type { PropsWithChildren } from "react";
import { ToastProvider } from "../components/ui/Toast";
import "../i18n/i18n";
import { queryClient } from "../lib/queryClient";
import { AuthProvider } from "./AuthProvider";

export function AppProviders({ children }: PropsWithChildren) {
  return (
    <QueryClientProvider client={queryClient}>
      <ToastProvider>
        <AuthProvider>{children}</AuthProvider>
      </ToastProvider>
    </QueryClientProvider>
  );
}
