/**
 * Names a metric's source for display.
 *
 * An empty source is not an unknown one: HealthKit writes through Health Auto
 * Export without setting the field, and the source-priority rules match it as
 * the empty string. Showing "—" there would claim the origin is unrecorded when
 * it is in fact the phone.
 */
export function sourceLabel(source: string | null | undefined): string {
  if (source == null) return "—";
  if (source === "") return "Apple Health";
  return source;
}

/** The same, spelled out where there is room for the mechanism. */
export function sourceLabelLong(source: string | null | undefined): string {
  if (source === "") return "Apple Health (HealthKit)";
  return sourceLabel(source);
}
