import { Globe2, MapPin } from "lucide-react";
import { useEffect, type PropsWithChildren } from "react";
import { useTranslation } from "react-i18next";
import { Link, NavLink } from "react-router-dom";

type Language = "kk" | "ru" | "en";

export function SiteShell({ children }: PropsWithChildren) {
  const { t, i18n } = useTranslation();
  const language = supportedLanguage(i18n.language);

  useEffect(() => {
    document.documentElement.lang = language;
  }, [language]);

  return (
    <div className="site-shell">
      <a className="skip-link" href="#main-content">{t("common.skipToContent")}</a>
      <header className="site-header">
        <div className="page-container site-header__inner">
          <Link className="brand" to="/" aria-label={t("nav.homeLabel")}>
            <span>Job</span><strong>Hub</strong>
          </Link>
          <nav className="site-nav" aria-label={t("nav.primaryLabel")}>
            <NavLink to="/jobs">{t("nav.jobs")}</NavLink>
            <span>{t("nav.companies")}</span>
            <span className="site-nav__location"><MapPin size={16} aria-hidden="true" />{t("nav.location")}</span>
          </nav>
          <div className="site-header__actions">
            <label className="language-select">
              <span className="visually-hidden">{t("common.language")}</span>
              <Globe2 size={17} aria-hidden="true" />
              <select value={language} onChange={(event) => void i18n.changeLanguage(event.target.value)}>
                <option value="kk">{t("languages.kk")}</option>
                <option value="ru">{t("languages.ru")}</option>
                <option value="en">{t("languages.en")}</option>
              </select>
            </label>
            <span className="auth-link">{t("nav.login")}</span>
            <span className="auth-link auth-link--primary">{t("nav.register")}</span>
          </div>
        </div>
      </header>
      <main id="main-content">{children}</main>
      <footer className="site-footer">
        <div className="page-container site-footer__inner">
          <Link className="brand brand--small" to="/" aria-label={t("nav.homeLabel")}>
            <span>Job</span><strong>Hub</strong>
          </Link>
          <nav aria-label={t("footer.label")}>
            <Link to="/jobs">{t("nav.jobs")}</Link>
            <span>{t("nav.companies")}</span>
            <span>{t("nav.location")}</span>
          </nav>
          <p>{t("footer.copyright", { year: new Date().getFullYear() })}</p>
        </div>
      </footer>
    </div>
  );
}

function supportedLanguage(language: string): Language {
  return language === "kk" || language === "ru" ? language : "en";
}
