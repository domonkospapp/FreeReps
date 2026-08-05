import type { ReactNode } from "react";

/** The shared row: label column, value, both on one baseline. */
export function Row({
  label,
  labelWidth = 180,
  children,
}: {
  label: ReactNode;
  labelWidth?: number;
  children: ReactNode;
}) {
  return (
    <div
      style={{
        display: "flex",
        alignItems: "baseline",
        gap: 24,
        padding: "15px 0",
        borderBottom: "1px solid var(--color-neutral-300)",
      }}
    >
      <span className="kick" style={{ width: labelWidth, flex: "none" }}>
        {label}
      </span>
      <div style={{ flex: 1, minWidth: 0 }}>{children}</div>
    </div>
  );
}

/** Tab heading, explanatory paragraph, then content under a 2px rule. */
export function TabHeader({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  return (
    <>
      <h2 style={{ fontSize: 22 }}>{title}</h2>
      <p
        style={{
          font: "400 13px/1.55 var(--font-body)",
          color: "var(--color-neutral-700)",
          maxWidth: "62ch",
          margin: "10px 0 0",
        }}
      >
        {children}
      </p>
      <div
        style={{
          borderTop: "2px solid var(--color-text)",
          marginTop: 20,
        }}
      />
    </>
  );
}

export const MONO: React.CSSProperties = {
  fontFamily:
    "ui-monospace, SFMono-Regular, Menlo, Consolas, 'Liberation Mono', monospace",
  fontSize: 12.5,
};

/** A square checkbox — never the rounded native one. */
export function SquareCheckbox({
  checked,
  onChange,
  label,
}: {
  checked: boolean;
  onChange: () => void;
  label: string;
}) {
  return (
    <button
      type="button"
      role="checkbox"
      aria-checked={checked}
      aria-label={label}
      onClick={onChange}
      style={{
        width: 15,
        height: 15,
        flex: "none",
        padding: 0,
        cursor: "pointer",
        borderRadius: 0,
        border: checked
          ? "2px solid var(--color-accent)"
          : "1px solid var(--color-divider)",
        background: checked ? "var(--color-accent)" : "transparent",
      }}
    />
  );
}

/** A square switch — square, never a pill. Used by the phone layout. */
export function SquareSwitch({
  checked,
  onChange,
  label,
}: {
  checked: boolean;
  onChange: () => void;
  label: string;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      onClick={onChange}
      style={{
        width: 44,
        height: 26,
        flex: "none",
        padding: 2,
        cursor: "pointer",
        borderRadius: 0,
        border: "1px solid var(--color-neutral-400)",
        background: checked ? "var(--color-accent)" : "transparent",
        display: "flex",
        alignItems: "center",
        justifyContent: checked ? "flex-end" : "flex-start",
      }}
    >
      <span
        style={{
          width: 20,
          height: 20,
          display: "block",
          background: checked ? "var(--color-bg)" : "var(--color-neutral-400)",
        }}
      />
    </button>
  );
}
