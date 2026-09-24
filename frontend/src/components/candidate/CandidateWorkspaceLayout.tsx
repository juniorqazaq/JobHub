import type { PropsWithChildren } from "react";
import { useTranslation } from "react-i18next";
import { NavLink, useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../../app/authContext";
import { SiteShell } from "../../layouts/SiteShell";

const destinations = [
  ["/workspace", "overview"],
  ["/profile", "profile"],
  ["/applications", "applications"],
  ["/saved", "saved"],
  ["/resume", "resume"],
  ["/preferences", "preferences"],
  ["/account", "account"],
] as const;

export function CandidateWorkspaceLayout({ children }: PropsWithChildren) {
  const { t } = useTranslation();
  const { session } = useAuth();
  const location = useLocation();
  const navigate = useNavigate();
  const user = session!.user;
  const activePath =
    destinations.find(([path]) => location.pathname === path)?.[0] ??
    "/workspace";

  return (
    <SiteShell>
      <div className="candidate-workspace page-container">
        <aside className="candidate-workspace__sidebar">
          <div className="candidate-workspace__identity">
            <span aria-hidden="true">{initials(user.fullName)}</span>
            <div>
              <strong>{user.fullName}</strong>
              <small>{t("workspace.title")}</small>
            </div>
          </div>
          <nav aria-label={t("workspace.navigationLabel")}>
            {destinations.map(([path, key]) => (
              <NavLink
                key={path}
                to={path}
                end
                aria-current={location.pathname === path ? "page" : undefined}
              >
                {t(`workspace.nav.${key}`)}
              </NavLink>
            ))}
          </nav>
        </aside>
        <label className="candidate-workspace__mobile-nav">
          <span>{t("workspace.navigationLabel")}</span>
          <select
            value={activePath}
            onChange={(event) => navigate(event.target.value)}
          >
            {destinations.map(([path, key]) => (
              <option key={path} value={path}>
                {t(`workspace.nav.${key}`)}
              </option>
            ))}
          </select>
        </label>
        <div className="candidate-workspace__content">{children}</div>
      </div>
    </SiteShell>
  );
}

function initials(value: string) {
  return value
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toLocaleUpperCase())
    .join("");
}
