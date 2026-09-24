import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { EmptyState } from "../components/ui/Feedback";
import { SiteShell } from "../layouts/SiteShell";

export function NotFoundPage() {
  const { t } = useTranslation();

  return (
    <SiteShell>
      <div className="page-container not-found-page">
        <EmptyState
          illustration="/illustrations/jobhub-404.png"
          illustrationLoading="eager"
          illustrationWidth={960}
          illustrationHeight={720}
          title={t("errors.notFound.title")}
          description={t("errors.notFound.description")}
          secondary={
            <>
              <Link className="ui-button ui-button--primary ui-button--md" to="/">
                {t("errors.notFound.home")}
              </Link>
              <Link className="ui-button ui-button--secondary ui-button--md" to="/jobs">
                {t("errors.notFound.jobs")}
              </Link>
            </>
          }
        />
      </div>
    </SiteShell>
  );
}
