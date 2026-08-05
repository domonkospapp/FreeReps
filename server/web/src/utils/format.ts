/** U+2212, the real minus sign. A hyphen is too short beside tabular digits. */
export const MINUS = "−";

/** U+2009, a thin space. The thousands separator in every number on screen. */
const THIN_SPACE = " ";

/**
 * Formats a number with a thin space between thousands and the real minus sign
 * for negatives.
 */
export function formatNumber(value: number, digits?: number): string {
  const fixed =
    digits != null
      ? Math.abs(value).toFixed(digits)
      : String(roundForDisplay(Math.abs(value)));
  const [whole, fraction] = fixed.split(".");
  const grouped = whole.replace(/\B(?=(\d{3})+(?!\d))/g, THIN_SPACE);
  const body = fraction ? `${grouped}.${fraction}` : grouped;
  return value < 0 ? `${MINUS}${body}` : body;
}

/** Large values read better whole; small ones need the decimal. */
function roundForDisplay(v: number): number {
  if (v >= 100) return Math.round(v);
  if (v >= 10) return Math.round(v * 10) / 10;
  return Math.round(v * 100) / 100;
}

/** Signed value with the real minus sign, e.g. "+4.1" / "−1.2". */
export function formatDelta(value: number, digits?: number): string {
  const body = formatNumber(Math.abs(value), digits);
  return value < 0 ? `${MINUS}${body}` : `+${body}`;
}

/** Signed percentage, e.g. "+6%" / "−6%". */
export function formatDeltaPercent(fraction: number): string {
  return `${fraction < 0 ? MINUS : "+"}${Math.abs(fraction * 100).toFixed(0)}%`;
}

/** Hours as "7:24". */
export function formatHoursMinutes(hours: number): string {
  const total = Math.round(hours * 60);
  return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, "0")}`;
}

/** Minutes as "54m" or "1h 04m". */
export function formatDuration(seconds: number): string {
  const total = Math.round(seconds / 60);
  const h = Math.floor(total / 60);
  const m = total % 60;
  return h > 0 ? `${h}h ${String(m).padStart(2, "0")}m` : `${m}m`;
}

/** Clock time as "23:24". */
export function formatClock(iso: string): string {
  const d = new Date(iso);
  return `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
}

export function formatTimeAgo(iso: string): string {
  const diffMin = Math.floor((Date.now() - new Date(iso).getTime()) / 60000);
  if (diffMin < 60) return `${Math.max(diffMin, 0)}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  const diffDay = Math.floor(diffHr / 24);
  return diffDay === 0 ? "today" : `${diffDay}d ago`;
}

/** "Monday, 16 March 2026" — the dashboard kicker. */
export function formatFullDate(d: Date): string {
  return d.toLocaleDateString("en-GB", {
    weekday: "long",
    day: "numeric",
    month: "long",
    year: "numeric",
  });
}

/** "Mon, 16 Mar" — the table group header. */
export function formatShortDate(d: Date): string {
  return d.toLocaleDateString("en-GB", {
    weekday: "short",
    day: "numeric",
    month: "short",
  });
}

/** "16 Dec 2025" — range endpoints. */
export function formatDateWithYear(d: Date): string {
  return d.toLocaleDateString("en-GB", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

/** "16 Dec" — chart endpoints. */
export function formatDayMonth(d: Date): string {
  return d.toLocaleDateString("en-GB", { day: "numeric", month: "short" });
}
