import type { HTMLAttributes, ReactNode } from "react";

export function Card({ className = "", ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={`ui-card ${className}`} {...props} />;
}

export function Badge({ tone = "neutral", children }: { tone?: "neutral" | "success" | "warning" | "error"; children: ReactNode }) {
  return <span className={`ui-badge ui-badge--${tone}`}>{children}</span>;
}

export function Avatar({ name, src, size = "md" }: { name: string; src?: string; size?: "sm" | "md" | "lg" }) {
  const initials = name.split(/\s+/).slice(0, 2).map((part) => part[0]).join("");
  return (
    <span className={`ui-avatar ui-avatar--${size}`} aria-label={name}>
      {src ? <img src={src} alt="" /> : initials}
    </span>
  );
}

export function Progress({ value, label }: { value: number; label: string }) {
  const clamped = Math.min(100, Math.max(0, value));
  return (
    <div className="ui-progress-wrap">
      <div className="ui-progress-label"><span>{label}</span><strong>{clamped}%</strong></div>
      <div className="ui-progress" role="progressbar" aria-label={label} aria-valuemin={0} aria-valuemax={100} aria-valuenow={clamped}>
        <span style={{ width: `${clamped}%` }} />
      </div>
    </div>
  );
}
