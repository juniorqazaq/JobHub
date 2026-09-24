import { useTranslation } from "react-i18next";

export function JobHubLoader() {
  const { t } = useTranslation();

  return (
    <div className="jobhub-loader" role="status" aria-live="polite">
      <span className="jobhub-loader__stage" aria-hidden="true">
        <img
          className="jobhub-loader__logo"
          src="/brand/jobhub-mark.png"
          width="256"
          height="256"
          alt=""
        />
        <span className="jobhub-loader__orbit">
          <span className="jobhub-loader__dot" />
        </span>
      </span>
      <span className="visually-hidden">{t("common.loading")}</span>
    </div>
  );
}
