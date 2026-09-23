import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from "react";

type ButtonVariant = "primary" | "secondary" | "quiet" | "danger";

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: "sm" | "md" | "lg";
  leadingIcon?: ReactNode;
  isLoading?: boolean;
}

export function Button({
  variant = "primary",
  size = "md",
  leadingIcon,
  isLoading = false,
  className = "",
  children,
  disabled,
  ...props
}: ButtonProps) {
  return (
    <button
      className={`ui-button ui-button--${variant} ui-button--${size} ${className}`}
      disabled={disabled || isLoading}
      {...props}
    >
      {isLoading ? <span className="ui-spinner" aria-hidden="true" /> : leadingIcon}
      <span>{children}</span>
    </button>
  );
}

export interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  label: string;
  icon: ReactNode;
  variant?: "default" | "quiet";
}

export const IconButton = forwardRef<HTMLButtonElement, IconButtonProps>(function IconButton(
  { label, icon, variant = "default", className = "", ...props },
  ref,
) {
  return (
    <button
      ref={ref}
      className={`ui-icon-button ui-icon-button--${variant} ${className}`}
      aria-label={label}
      title={label}
      {...props}
    >
      {icon}
    </button>
  );
});
