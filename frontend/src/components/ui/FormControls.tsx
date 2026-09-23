import { Search, X } from "lucide-react";
import { forwardRef, useId, type InputHTMLAttributes, type ReactNode, type SelectHTMLAttributes, type TextareaHTMLAttributes } from "react";
import { IconButton } from "./Button";

interface FieldShellProps {
  id: string;
  label: string;
  hint?: string;
  error?: string;
  children: ReactNode;
}

function FieldShell({ id, label, hint, error, children }: FieldShellProps) {
  const detailId = `${id}-detail`;
  return (
    <div className="ui-field">
      <label className="ui-label" htmlFor={id}>{label}</label>
      {children}
      {(error || hint) && (
        <span id={detailId} className={error ? "ui-field-error" : "ui-field-hint"}>
          {error || hint}
        </span>
      )}
    </div>
  );
}

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
  hint?: string;
  error?: string;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { label, hint, error, id: providedId, className = "", ...props },
  ref,
) {
  const generatedId = useId();
  const id = providedId ?? generatedId;
  return (
    <FieldShell id={id} label={label} hint={hint} error={error}>
      <input
        ref={ref}
        id={id}
        className={`ui-input ${className}`}
        aria-invalid={Boolean(error)}
        aria-describedby={error || hint ? `${id}-detail` : undefined}
        {...props}
      />
    </FieldShell>
  );
});

export interface SearchInputProps extends InputProps {
  clearLabel: string;
  onClear?: () => void;
}

export const SearchInput = forwardRef<HTMLInputElement, SearchInputProps>(function SearchInput(
  { clearLabel, onClear, value, ...props },
  ref,
) {
  return (
    <div className="ui-search-field">
      <Search size={19} aria-hidden="true" />
      <Input ref={ref} value={value} {...props} />
      {onClear && value ? (
        <IconButton label={clearLabel} icon={<X size={17} />} variant="quiet" onClick={onClear} />
      ) : null}
    </div>
  );
});

export interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  label: string;
  hint?: string;
  error?: string;
}

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(function Textarea(
  { label, hint, error, id: providedId, className = "", ...props },
  ref,
) {
  const generatedId = useId();
  const id = providedId ?? generatedId;
  return (
    <FieldShell id={id} label={label} hint={hint} error={error}>
      <textarea
        ref={ref}
        id={id}
        className={`ui-input ui-textarea ${className}`}
        aria-invalid={Boolean(error)}
        aria-describedby={error || hint ? `${id}-detail` : undefined}
        {...props}
      />
    </FieldShell>
  );
});

export interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  label: string;
  hint?: string;
  error?: string;
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(function Select(
  { label, hint, error, id: providedId, className = "", children, ...props },
  ref,
) {
  const generatedId = useId();
  const id = providedId ?? generatedId;
  return (
    <FieldShell id={id} label={label} hint={hint} error={error}>
      <select
        ref={ref}
        id={id}
        className={`ui-input ui-select ${className}`}
        aria-invalid={Boolean(error)}
        aria-describedby={error || hint ? `${id}-detail` : undefined}
        {...props}
      >
        {children}
      </select>
    </FieldShell>
  );
});

export interface ChoiceProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
  description?: string;
}

export function Checkbox({ label, description, id: providedId, ...props }: ChoiceProps) {
  const generatedId = useId();
  const id = providedId ?? generatedId;
  return (
    <label className="ui-choice" htmlFor={id}>
      <input id={id} type="checkbox" {...props} />
      <span className="ui-choice-control" aria-hidden="true" />
      <span><strong>{label}</strong>{description ? <small>{description}</small> : null}</span>
    </label>
  );
}

export interface RadioOption {
  value: string;
  label: string;
}

export interface RadioGroupProps {
  label: string;
  name: string;
  options: RadioOption[];
  value?: string;
  onChange?: (value: string) => void;
}

export function RadioGroup({ label, name, options, value, onChange }: RadioGroupProps) {
  return (
    <fieldset className="ui-radio-group">
      <legend className="ui-label">{label}</legend>
      <div>
        {options.map((option) => (
          <label className="ui-choice" key={option.value}>
            <input
              type="radio"
              name={name}
              value={option.value}
              checked={value === option.value}
              onChange={() => onChange?.(option.value)}
            />
            <span className="ui-choice-control" aria-hidden="true" />
            <span><strong>{option.label}</strong></span>
          </label>
        ))}
      </div>
    </fieldset>
  );
}

export interface SwitchProps extends Omit<InputHTMLAttributes<HTMLInputElement>, "type"> {
  label: string;
  description?: string;
}

export function Switch({ label, description, id: providedId, ...props }: SwitchProps) {
  const generatedId = useId();
  const id = providedId ?? generatedId;
  return (
    <label className="ui-switch-row" htmlFor={id}>
      <span><strong>{label}</strong>{description ? <small>{description}</small> : null}</span>
      <input id={id} type="checkbox" role="switch" {...props} />
      <span className="ui-switch" aria-hidden="true"><span /></span>
    </label>
  );
}
