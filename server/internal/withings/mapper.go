package withings

import (
	"math"
	"slices"
	"time"

	"github.com/claude/freereps/internal/models"
)

// SourceName is written to health_metrics.source for every row this package
// produces. Source priority matches named sources by prefix, so this value also
// decides which rows a `Withings` priority entry covers.
const SourceName = "Withings"

// measureType is a Withings meastype code. The full catalogue and the codes
// FreeReps deliberately skips are documented in specs/withings-api.md.
const (
	typeWeight      = 1
	typeFatFreeMass = 5
	typeFatRatio    = 6
	typeFatMass     = 8
	typeDiastolic   = 9
	typeSystolic    = 10
	typeHeartPulse  = 11
	typeMuscleMass  = 76
	typeHydration   = 77
	typeBoneMass    = 88
)

// metricTarget names the FreeReps metric and unit a meastype maps to.
type metricTarget struct {
	name  string
	units string
}

// measureTypeMap is both the mapping and the request list: RequestedTypes is
// derived from it, so a type can never be requested without a target metric.
//
// Type 11 is the pulse the blood pressure cuff records with each reading, not a
// continuous heart rate. It gets its own metric name because writing a single
// seated measurement into `heart_rate` would shift every daily average that
// Oura and the Apple Watch produce.
var measureTypeMap = map[int]metricTarget{
	typeWeight:      {"weight_body_mass", "kg"},
	typeFatFreeMass: {"lean_body_mass", "kg"},
	typeFatRatio:    {"body_fat_percentage", "%"},
	typeFatMass:     {"fat_mass", "kg"},
	typeDiastolic:   {"blood_pressure_diastolic", "mmHg"},
	typeSystolic:    {"blood_pressure_systolic", "mmHg"},
	typeHeartPulse:  {"blood_pressure_heart_rate", "bpm"},
	typeMuscleMass:  {"muscle_mass", "kg"},
	typeHydration:   {"body_water", "kg"},
	typeBoneMass:    {"bone_mass", "kg"},
}

// RequestedTypes returns the meastype codes to ask getmeas for, sorted so the
// request is stable across calls.
func RequestedTypes() []int {
	types := make([]int, 0, len(measureTypeMap))
	for t := range measureTypeMap {
		types = append(types, t)
	}
	slices.Sort(types)
	return types
}

// measureValue decodes the Withings value/unit pair into a single number.
func measureValue(m Measure) float64 {
	return float64(m.Value) * math.Pow10(m.Unit)
}

// MapMeasureGroups converts Withings measurement groups into health metric rows.
//
// Blood pressure is written as two separate qty rows rather than through the
// Systolic/Diastolic columns of HealthMetricRow: `blood_pressure` has no
// allowlist entry, and the dashboard, correlations and the iOS app all work with
// the split metric names.
func MapMeasureGroups(groups []MeasureGroup, userID int) []models.HealthMetricRow {
	var rows []models.HealthMetricRow
	for _, g := range groups {
		// Every measure in a group shares the group's timestamp. Blood pressure
		// monitors produce three readings within a few minutes and the scale
		// reports a dozen values at once; grouping by rounded time instead
		// would pair values across readings.
		t := time.Unix(g.Date, 0).UTC()
		for _, m := range g.Measures {
			target, ok := measureTypeMap[m.Type]
			if !ok {
				// A measure type FreeReps does not map — a device reporting
				// more than was asked for is not an error.
				continue
			}
			qty := measureValue(m)
			rows = append(rows, models.HealthMetricRow{
				Time:       t,
				UserID:     userID,
				MetricName: target.name,
				Source:     SourceName,
				Units:      target.units,
				Qty:        &qty,
			})
		}
	}
	return rows
}
