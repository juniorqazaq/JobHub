import { useTranslation } from "react-i18next";
import type { AuthUser } from "../../api/models/auth";
import type { CandidateProfile } from "../../api/models/candidate";

interface CandidateIdentityHeaderProps {
  eyebrow: string;
  profile: CandidateProfile;
  user: AuthUser;
  titleId?: string;
}

export function CandidateIdentityHeader({
  eyebrow,
  profile,
  user,
  titleId,
}: CandidateIdentityHeaderProps) {
  const { t } = useTranslation();
  const position =
    profile.desiredPosition ||
    profile.currentPosition ||
    t("profile.positionMissing");

  return (
    <header className="candidate-identity-header">
      <div className="profile-avatar" aria-hidden="true">
        {initials(profile.fullName)}
      </div>
      <div className="candidate-identity-header__copy">
        <p className="auth-eyebrow">{eyebrow}</p>
        <h1 id={titleId}>{profile.fullName}</h1>
        <p className="candidate-identity-header__position">{position}</p>
        <dl className="candidate-account-meta">
          <div>
            <dt>{t("auth.email")}</dt>
            <dd>{user.email}</dd>
          </div>
          <div>
            <dt>{t("auth.role")}</dt>
            <dd>{t(`auth.roles.${user.role}`)}</dd>
          </div>
          <div>
            <dt>{t("auth.accountStatus")}</dt>
            <dd>
              <span className={`account-status account-status--${user.status}`}>
                {t(`auth.statuses.${user.status}`)}
              </span>
            </dd>
          </div>
        </dl>
      </div>
      <div
        className="profile-completion"
        aria-label={t("profile.completionLabel", {
          value: profile.completion.percentage,
        })}
      >
        <strong>{profile.completion.percentage}%</strong>
        <span>{t("profile.complete")}</span>
        <div aria-hidden="true">
          <span style={{ width: `${profile.completion.percentage}%` }} />
        </div>
      </div>
    </header>
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
