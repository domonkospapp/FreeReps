import type { FrontPageMetric } from "../../api";
import {
  formatDelta,
  formatDeltaPercent,
  formatNumber,
} from "../../utils/format";

const CATEGORY_LABEL: Record<string, string> = {
  cardiovascular: "Cardiovascular",
  sleep: "Sleep",
  activity: "Activity",
  body: "Body",
  fitness: "Fitness",
  oura: "Oura",
  respiratory: "Respiratory",
  hearing: "Hearing",
  nutrition: "Nutrition",
  lab: "Lab",
  other: "Other",
};

const CATEGORY_ORDER = [
  "cardiovascular",
  "sleep",
  "activity",
  "body",
  "fitness",
  "oura",
  "respiratory",
  "hearing",
  "nutrition",
  "lab",
  "other",
];

/** The latest reading with the display multiplier applied. */
export function displayValue(m: FrontPageMetric): string {
  if (m.latest == null) return "—";
  return formatNumber(m.latest * m.multiplier);
}

/**
 * Absolute for most metrics, relative for cumulative ones — a step count
 * difference of 400 says less than "−6%".
 */
export function displayDelta(m: FrontPageMetric): string {
  if (m.is_cumulative) {
    if (m.delta_7d_pct == null) return "—";
    return Math.abs(m.delta_7d_pct * 100) < 0.5
      ? "flat"
      : formatDeltaPercent(m.delta_7d_pct);
  }
  if (m.delta_7d == null) return "—";
  const shown = formatDelta(m.delta_7d * m.multiplier);
  // "+0" reads as movement where there is none.
  return /^[+−]0(\.0+)?$/.test(shown) ? "flat" : shown;
}

export function displayRange(m: FrontPageMetric): string {
  if (m.range_low == null || m.range_high == null) return "—";
  return `${formatNumber(m.range_low * m.multiplier)} – ${formatNumber(
    m.range_high * m.multiplier,
  )}`;
}

export interface MetricGroupSection {
  category: string;
  label: string;
  metrics: FrontPageMetric[];
}

/** Groups metrics into the table's category bands, in a fixed reading order. */
export function groupByCategory(
  metrics: FrontPageMetric[],
): MetricGroupSection[] {
  const byCategory = new Map<string, FrontPageMetric[]>();
  for (const m of metrics) {
    const list = byCategory.get(m.category) ?? [];
    list.push(m);
    byCategory.set(m.category, list);
  }
  const known = CATEGORY_ORDER.filter((c) => byCategory.has(c));
  const rest = [...byCategory.keys()].filter((c) => !CATEGORY_ORDER.includes(c));
  return [...known, ...rest].map((category) => ({
    category,
    label: CATEGORY_LABEL[category] ?? category,
    metrics: byCategory.get(category)!,
  }));
}

export function categoryLabel(category: string): string {
  return CATEGORY_LABEL[category] ?? category;
}
