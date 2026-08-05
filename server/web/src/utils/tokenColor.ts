/**
 * Resolves a design token to a concrete colour.
 *
 * Canvas-based charts cannot read CSS variables, so anything drawn with uPlot
 * has to look the value up at render time — and re-read it when the theme
 * changes, which is why callers key their memo on the theme preference.
 */
export function tokenColor(name: string, fallback = "#000000"): string {
  if (typeof window === "undefined") return fallback;
  const value = getComputedStyle(document.documentElement)
    .getPropertyValue(name)
    .trim();
  return value || fallback;
}

/** The same colour at a given alpha, for chart fills. */
export function tokenColorAlpha(name: string, alpha: number): string {
  return `color-mix(in srgb, ${tokenColor(name)} ${Math.round(alpha * 100)}%, transparent)`;
}
