/**
 * The neutral ramp carries depth and intensity; the accent is reserved for the
 * one thing that should pop out — being awake, and the top heart rate zone.
 * One mapping, shared by the desktop and phone layouts.
 */

export const STAGE_COLOR: Record<string, string> = {
  Deep: "var(--color-neutral-900)",
  Core: "var(--color-neutral-700)",
  REM: "var(--color-neutral-400)",
  Awake: "var(--color-accent)",
};

/** Top to bottom in the hypnogram: shallowest first. */
export const STAGE_LANES = ["Awake", "REM", "Core", "Deep"] as const;

/** Darkest = deepest, so the composition bar reads as a depth ramp. */
export const STAGE_COMPOSITION_ORDER = ["Deep", "Core", "REM", "Awake"] as const;

export function stageColor(stage: string): string {
  return STAGE_COLOR[stage] ?? "var(--color-neutral-500)";
}

export const ZONE_COLORS = [
  "var(--color-neutral-300)",
  "var(--color-neutral-500)",
  "var(--color-neutral-700)",
  "var(--color-neutral-900)",
  "var(--color-accent)",
];

/**
 * Zone boundaries as fractions of max heart rate. The screen states the bpm
 * bands computed from the user's own maximum rather than hardcoding them.
 */
export const ZONE_BOUNDS = [0.6, 0.7, 0.8, 0.9];

export function zoneBands(maxHR: number): string[] {
  const edges = ZONE_BOUNDS.map((f) => Math.round(maxHR * f));
  return [
    `< ${edges[0]}`,
    `${edges[0]}–${edges[1]}`,
    `${edges[1]}–${edges[2]}`,
    `${edges[2]}–${edges[3]}`,
    `> ${edges[3]}`,
  ];
}

/** Which zone a heart rate lands in, 0-indexed. */
export function zoneOf(bpm: number, maxHR: number): number {
  const f = bpm / maxHR;
  for (let i = 0; i < ZONE_BOUNDS.length; i++) {
    if (f < ZONE_BOUNDS[i]) return i;
  }
  return ZONE_BOUNDS.length;
}
