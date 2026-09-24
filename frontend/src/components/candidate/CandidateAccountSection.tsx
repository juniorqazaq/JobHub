import { LogOut } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../../app/authContext";
import { Button } from "../ui/Button";

export function CandidateAccountSection() {
  const { t } = useTranslation();
  const auth = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [logoutPending, setLogoutPending] = useState(false);
  const [logoutError, setLogoutError] = useState("");

  useEffect(() => {
    if (location.hash === "#account") {
      requestAnimationFrame(() => document.getElementById("account")?.scrollIntoView());
    }
  }, [location.hash]);

  const logout = async () => {
    if (logoutPending) return;
    setLogoutError("");
    setLogoutPending(true);
    try {
      await auth.logout();
      navigate("/");
    } catch {
      setLogoutError(t("auth.logoutError"));
      setLogoutPending(false);
    }
  };

  return (
    <section id="account" className="candidate-account-actions" aria-label={t("auth.accountSummary")}>
      <Button variant="secondary" leadingIcon={<LogOut size={17} />} isLoading={logoutPending} onClick={() => void logout()}>
        {t("auth.logout")}
      </Button>
      {logoutError ? <p className="auth-error account-logout-error" role="alert">{logoutError}</p> : null}
    </section>
  );
}
