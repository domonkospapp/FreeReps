interface Props {
  values: (number | null)[];
  width: number;
  height: number;
  stroke: string;
  strokeWidth: number;
  /** CSS width; defaults to filling the container. */
  cssWidth?: number | string;
}

/**
 * One polyline, points computed from the array. No library, no layout pass.
 * The viewBox carries the geometry, so the same element scales with its
 * container on any breakpoint.
 */
export default function Sparkline({
  values,
  width,
  height,
  stroke,
  strokeWidth,
  cssWidth = "100%",
}: Props) {
  const points = polylinePoints(values, width, height);

  return (
    <svg
      viewBox={`0 0 ${width} ${height}`}
      preserveAspectRatio="none"
      style={{ width: cssWidth, height, display: "block" }}
      aria-hidden
    >
      {points ? (
        <polyline
          points={points}
          fill="none"
          stroke={stroke}
          strokeWidth={strokeWidth}
          vectorEffect="non-scaling-stroke"
        />
      ) : null}
    </svg>
  );
}

/** Returns null when fewer than two points carry a value. */
export function polylinePoints(
  values: (number | null)[],
  width: number,
  height: number,
): string | null {
  const present = values.filter((v): v is number => v != null);
  if (present.length < 2) return null;

  const min = Math.min(...present);
  const max = Math.max(...present);
  const span = max - min || 1;
  const step = values.length > 1 ? width / (values.length - 1) : 0;

  return values
    .map((v, i) =>
      v == null
        ? null
        : `${(i * step).toFixed(1)},${(height - 2 - ((v - min) / span) * (height - 4)).toFixed(1)}`,
    )
    .filter((p): p is string => p != null)
    .join(" ");
}
