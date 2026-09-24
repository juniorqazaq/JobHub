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
  const user = auth.session!.user;
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
    <section id="account" className="profile-section candidate-account-section" aria-labelledby="candidate-account-title">
      <header><h2 id="candidate-account-title">{t("auth.accountSummary")}</h2></header>
      <div className="profile-section__body">
        <dl className="profile-view-grid">
          <div><dt>{t("auth.email")}</dt><dd>{user.email}</dd></div>
          <div><dt>{t("auth.role")}</dt><dd>{t(`auth.roles.${user.role}`)}</dd></div>
          <div><dt>{t("auth.accountStatus")}</dt><dd><span className={`account-status account-status--${user.status}`}>{t(`auth.statuses.${user.status}`)}</span></dd></div>
        </dl>
        <div className="account-actions">
          <Button variant="secondary" leadingIcon={<LogOut size={17} />} isLoading={logoutPending} onClick={() => void logout()}>
            {t("auth.logout")}
          </Button>
          {logoutError ? <p className="auth-error account-logout-error" role="alert">{logoutError}</p> : null}
        </div>
      </div>
    </section>
  );
}
