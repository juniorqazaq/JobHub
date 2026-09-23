import { createContext } from "react";

export interface ToastMessage {
  title: string;
  description?: string;
  tone?: "success" | "error";
}

export interface ToastContextValue {
  showToast: (message: ToastMessage) => void;
}

export const ToastContext = createContext<ToastContextValue | null>(null);
