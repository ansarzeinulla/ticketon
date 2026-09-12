"use client";

import { useId, type InputHTMLAttributes, type ReactNode, type SelectHTMLAttributes } from "react";

const controlClasses =
  "w-full rounded-lg border border-border-subtle bg-surface px-3 py-2 text-sm " +
  "text-foreground placeholder:text-foreground-muted/70 " +
  "disabled:cursor-not-allowed disabled:opacity-60";

const invalidClasses = "border-danger";

interface FieldShellProps {
  label: string;
  htmlFor: string;
  hint?: ReactNode;
  error?: string;
  required?: boolean;
  children: ReactNode;
}

function FieldShell({ label, htmlFor, hint, error, required, children }: FieldShellProps) {
  return (
    <div className="space-y-1.5">
      <label htmlFor={htmlFor} className="block text-sm font-medium text-foreground">
        {label}
        {required && <span className="ml-1 text-danger">*</span>}
      </label>
      {children}
      {error ? (
        // The id has to be here, not only on the control's aria-errormessage:
        // an attribute pointing at an element that does not exist is silently
        // ignored, so a screen reader would announce the field as invalid
        // without ever saying why.
        <p id={`${htmlFor}-error`} className="text-xs text-danger" role="alert">
          {error}
        </p>
      ) : hint ? (
        <p className="text-xs text-foreground-muted">{hint}</p>
      ) : null}
    </div>
  );
}

interface TextFieldProps extends Omit<InputHTMLAttributes<HTMLInputElement>, "id"> {
  label: string;
  hint?: ReactNode;
  error?: string;
}

export function TextField({ label, hint, error, className, ...props }: TextFieldProps) {
  const id = useId();
  return (
    <FieldShell label={label} htmlFor={id} hint={hint} error={error} required={props.required}>
      <input
        id={id}
        {...props}
        aria-invalid={error ? true : undefined}
        aria-errormessage={error ? `${id}-error` : undefined}
        className={`${controlClasses} ${error ? invalidClasses : ""} ${className ?? ""}`}
      />
    </FieldShell>
  );
}

interface SelectFieldProps extends Omit<SelectHTMLAttributes<HTMLSelectElement>, "id"> {
  label: string;
  hint?: ReactNode;
  error?: string;
  options: readonly string[];
}

export function SelectField({
  label,
  hint,
  error,
  options,
  className,
  ...props
}: SelectFieldProps) {
  const id = useId();
  return (
    <FieldShell label={label} htmlFor={id} hint={hint} error={error} required={props.required}>
      <select
        id={id}
        {...props}
        aria-invalid={error ? true : undefined}
        aria-errormessage={error ? `${id}-error` : undefined}
        className={`${controlClasses} ${error ? invalidClasses : ""} ${className ?? ""}`}
      >
        {options.map((option) => (
          <option key={option} value={option}>
            {option}
          </option>
        ))}
      </select>
    </FieldShell>
  );
}

interface TextAreaFieldProps {
  label: string;
  name: string;
  value: string;
  onChange: (value: string) => void;
  rows?: number;
  hint?: ReactNode;
  error?: string;
  placeholder?: string;
  disabled?: boolean;
}

export function TextAreaField({
  label,
  name,
  value,
  onChange,
  rows = 3,
  hint,
  error,
  placeholder,
  disabled,
}: TextAreaFieldProps) {
  const id = useId();
  return (
    <FieldShell label={label} htmlFor={id} hint={hint} error={error}>
      <textarea
        id={id}
        name={name}
        rows={rows}
        value={value}
        placeholder={placeholder}
        disabled={disabled}
        onChange={(event) => onChange(event.target.value)}
        aria-invalid={error ? true : undefined}
        aria-errormessage={error ? `${id}-error` : undefined}
        className={`${controlClasses} resize-y ${error ? invalidClasses : ""}`}
      />
    </FieldShell>
  );
}
