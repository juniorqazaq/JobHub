import { CheckCircle2, X, XCircle } from "lucide-react";
import { useCallback, useMemo, useRef, useState, type PropsWithChildren } from "react";
import { useTranslation } from "react-i18next";
import { IconButton } from "./Button";
import { ToastContext, type ToastMessage } from "./toastContext";

interface ActiveToast extends ToastMessage { id: number }

export function ToastProvider({ children }: PropsWithChildren) {
  const { t } = useTranslation();
  const [toasts, setToasts] = useState<ActiveToast[]>([]);
  const nextId = useRef(0);
  const showToast = useCallback((message: ToastMessage) => {
    const id = nextId.current++;
    setToasts((current) => [...current, { ...message, id }]);
    window.setTimeout(() => setToasts((current) => current.filter((toast) => toast.id !== id)), 4_500);
  }, []);
  const value = useMemo(() => ({ showToast }), [showToast]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="ui-toast-region" aria-live="polite" aria-atomic="false">
        {toasts.map((toast) => (
          <div className={`ui-toast ui-toast--${toast.tone ?? "success"}`} key={toast.id} role="status">
            {toast.tone === "error" ? <XCircle size={20} /> : <CheckCircle2 size={20} />}
            <div><strong>{toast.title}</strong>{toast.description ? <p>{toast.description}</p> : null}</div>
            <IconButton label={t("common.close")} variant="quiet" icon={<X size={17} />} onClick={() => setToasts((current) => current.filter((item) => item.id !== toast.id))} />
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}
