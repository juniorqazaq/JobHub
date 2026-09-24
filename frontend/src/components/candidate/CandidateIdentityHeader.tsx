import { useTranslation } from "react-i18next";
import type { CandidateProfile } from "../../api/models/candidate";

interface CandidateIdentityHeaderProps {
  eyebrow: string;
  profile: CandidateProfile;
  titleId?: string;
}

export function CandidateIdentityHeader({
  eyebrow,
  profile,
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
