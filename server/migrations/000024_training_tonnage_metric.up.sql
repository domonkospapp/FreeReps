-- Register the derived training volume series.
--
-- The metric is computed from workout_sets by RebuildTrainingMetrics and written
-- into health_metrics, because GetCorrelation reads that table exclusively.
-- Without this row the data would be stored but invisible: getAvailableMetrics
-- inner-joins the allowlist, so the dashboard and the correlation picker would
-- never offer it.
--
-- is_cumulative marks it as a total rather than a level — a week's tonnage is
-- the sum of its days, not their average.
INSERT INTO metric_allowlist (metric_name, category, display_label, display_unit, is_cumulative)
VALUES ('strength_tonnage', 'training', 'Strength Tonnage', 'kg', TRUE)
ON CONFLICT (metric_name) DO UPDATE SET
    category = EXCLUDED.category,
    display_label = EXCLUDED.display_label,
    display_unit = EXCLUDED.display_unit,
    is_cumulative = EXCLUDED.is_cumulative;
