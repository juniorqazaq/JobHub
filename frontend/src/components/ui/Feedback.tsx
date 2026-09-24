import { AlertCircle, BriefcaseBusiness } from "lucide-react";
import type { ReactNode } from "react";
import { Button } from "./Button";

export function Skeleton({ className = "" }: { className?: string }) {
  return <span className={`ui-skeleton ${className}`} aria-hidden="true" />;
}

export function JobListSkeleton({ label }: { label: string }) {
  return (
    <div className="ui-skeleton-list" role="status" aria-label={label}>
      {[0, 1, 2].map((item) => (
        <div className="ui-skeleton-row" key={item}>
          <Skeleton className="ui-skeleton-logo" />
          <div><Skeleton className="ui-skeleton-title" /><Skeleton /><Skeleton className="ui-skeleton-short" /></div>
        </div>
      ))}
    </div>
  );
}

interface StateProps {
  title: string;
  description: string;
  illustration?: string;
  illustrationAlt?: string;
  illustrationLoading?: "eager" | "lazy";
  illustrationWidth?: number;
  illustrationHeight?: number;
  actionLabel?: string;
  onAction?: () => void;
  secondary?: ReactNode;
}

export function EmptyState({
  title,
  description,
  illustration,
  illustrationAlt = "",
  illustrationLoading = "lazy",
  illustrationWidth = 800,
  illustrationHeight = 800,
  actionLabel,
  onAction,
  secondary,
}: StateProps) {
  return (
    <div className={`ui-state${illustration ? " ui-state--illustrated" : ""}`}>
      {illustration ? (
        <img
          className="ui-state__illustration"
          src={illustration}
          width={illustrationWidth}
          height={illustrationHeight}
          loading={illustrationLoading}
          alt={illustrationAlt}
        />
      ) : (
        <BriefcaseBusiness size={24} aria-hidden="true" />
      )}
      <h3>{title}</h3><p>{description}</p>
      {actionLabel && onAction ? <Button onClick={onAction}>{actionLabel}</Button> : null}
      {secondary ? <div className="ui-state__actions">{secondary}</div> : null}
    </div>
  );
}

export function ErrorState({ title, description, actionLabel, onAction, secondary }: StateProps) {
  return (
    <div className="ui-state ui-state--error" role="alert">
      <AlertCircle size={24} aria-hidden="true" />
      <h3>{title}</h3><p>{description}</p>
      {actionLabel && onAction ? <Button variant="secondary" onClick={onAction}>{actionLabel}</Button> : null}
      {secondary}
    </div>
  );
}
