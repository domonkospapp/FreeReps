package withings

import (
	"math"
	"testing"
	"time"
)

// TestMeasureValueExponent verifies the value/unit decoding.
//
// Withings sends 76.543 kg as value 76543 with unit -3. Storing the raw value
// would put a five-digit weight into the series, and no validation downstream
// would reject it.
func TestMeasureValueExponent(t *testing.T) {
	tests := []struct {
		name  string
		input Measure
		want  float64
	}{
		{"weight in grams", Measure{Value: 76543, Type: typeWeight, Unit: -3}, 76.543},
		{"fat ratio in hundredths", Measure{Value: 1823, Type: typeFatRatio, Unit: -2}, 18.23},
		{"systolic without exponent", Measure{Value: 118, Type: typeSystolic, Unit: 0}, 118},
		{"positive exponent", Measure{Value: 7, Type: typeWeight, Unit: 1}, 70},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := measureValue(tt.input)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("measureValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMapMeasureGroupsScale verifies that one scale step produces one row per
// mapped measure type, all carrying the group's timestamp.
func TestMapMeasureGroupsScale(t *testing.T) {
	groups := []MeasureGroup{{
		GroupID:  1,
		Date:     1754377200,
		Category: 1,
		Measures: []Measure{
			{Value: 76543, Type: typeWeight, Unit: -3},
			{Value: 1823, Type: typeFatRatio, Unit: -2},
			{Value: 62580, Type: typeFatFreeMass, Unit: -3},
			{Value: 13963, Type: typeFatMass, Unit: -3},
			{Value: 59420, Type: typeMuscleMass, Unit: -3},
			{Value: 3160, Type: typeBoneMass, Unit: -3},
			{Value: 43110, Type: typeHydration, Unit: -3},
		},
	}}

	rows := MapMeasureGroups(groups, 7)
	if len(rows) != 7 {
		t.Fatalf("got %d rows, want 7", len(rows))
	}

	want := map[string]float64{
		"weight_body_mass":    76.543,
		"body_fat_percentage": 18.23,
		"lean_body_mass":      62.58,
		"fat_mass":            13.963,
		"muscle_mass":         59.42,
		"bone_mass":           3.16,
		"body_water":          43.11,
	}
	seen := map[string]bool{}
	for _, r := range rows {
		w, ok := want[r.MetricName]
		if !ok {
			t.Errorf("unexpected metric %q", r.MetricName)
			continue
		}
		seen[r.MetricName] = true
		if r.Qty == nil {
			t.Fatalf("%s: qty is nil", r.MetricName)
		}
		if math.Abs(*r.Qty-w) > 1e-9 {
			t.Errorf("%s = %v, want %v", r.MetricName, *r.Qty, w)
		}
		if !r.Time.Equal(time.Unix(1754377200, 0).UTC()) {
			t.Errorf("%s: time = %v, want the group timestamp", r.MetricName, r.Time)
		}
		if r.UserID != 7 {
			t.Errorf("%s: user_id = %d, want 7", r.MetricName, r.UserID)
		}
	}
	for name := range want {
		if !seen[name] {
			t.Errorf("metric %q missing from the mapped rows", name)
		}
	}
}

// TestMapMeasureGroupsBloodPressure verifies that one cuff reading produces
// three separate qty rows sharing a timestamp.
//
// Two things this guards: blood pressure is written as split qty metrics rather
// than through the Systolic/Diastolic columns, because `blood_pressure` has no
// allowlist entry and would be rejected at ingest; and the cuff's pulse goes to
// its own metric instead of `heart_rate`, where a single seated reading would
// shift the daily average of the continuous heart rate from other sources.
func TestMapMeasureGroupsBloodPressure(t *testing.T) {
	groups := []MeasureGroup{{
		GroupID:  2,
		Date:     1754380000,
		Category: 1,
		Measures: []Measure{
			{Value: 78, Type: typeDiastolic, Unit: 0},
			{Value: 118, Type: typeSystolic, Unit: 0},
			{Value: 61, Type: typeHeartPulse, Unit: 0},
		},
	}}

	rows := MapMeasureGroups(groups, 1)
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}

	want := map[string]float64{
		"blood_pressure_diastolic":  78,
		"blood_pressure_systolic":   118,
		"blood_pressure_heart_rate": 61,
	}
	for _, r := range rows {
		w, ok := want[r.MetricName]
		if !ok {
			t.Fatalf("unexpected metric %q", r.MetricName)
		}
		if r.Qty == nil || *r.Qty != w {
			t.Errorf("%s = %v, want %v", r.MetricName, r.Qty, w)
		}
		if r.Systolic != nil || r.Diastolic != nil {
			t.Errorf("%s: systolic/diastolic columns filled, want qty only", r.MetricName)
		}
		if !r.Time.Equal(time.Unix(1754380000, 0).UTC()) {
			t.Errorf("%s: time = %v, want the group timestamp", r.MetricName, r.Time)
		}
	}
}

// TestMapMeasureGroupsSeparatesReadings verifies that two readings a few minutes
// apart keep their own timestamps. A blood pressure monitor produces three
// readings in a row; pairing them by rounded time would mix systolic from one
// reading with diastolic from another.
func TestMapMeasureGroupsSeparatesReadings(t *testing.T) {
	groups := []MeasureGroup{
		{GroupID: 1, Date: 1754380000, Measures: []Measure{{Value: 118, Type: typeSystolic}}},
		{GroupID: 2, Date: 1754380120, Measures: []Measure{{Value: 122, Type: typeSystolic}}},
	}

	rows := MapMeasureGroups(groups, 1)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[0].Time.Equal(rows[1].Time) {
		t.Error("both readings share a timestamp; they were taken 2 minutes apart")
	}
}

// TestMapMeasureGroupsSkipsUnknownTypes verifies that an unmapped meastype is
// dropped rather than failing the batch. A device reporting more than was asked
// for is not an error, and rejecting the group would lose the weight along with
// the unknown value.
func TestMapMeasureGroupsSkipsUnknownTypes(t *testing.T) {
	groups := []MeasureGroup{{
		GroupID: 3,
		Date:    1754380000,
		Measures: []Measure{
			{Value: 76543, Type: typeWeight, Unit: -3},
			{Value: 1750, Type: 4, Unit: -3},  // height, not requested
			{Value: 950, Type: 91, Unit: -2},  // pulse wave velocity, not mapped
			{Value: 1, Type: 999999, Unit: 0}, // nothing at all
		},
	}}

	rows := MapMeasureGroups(groups, 1)
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if rows[0].MetricName != "weight_body_mass" {
		t.Errorf("metric = %q, want weight_body_mass", rows[0].MetricName)
	}
}

// TestMapMeasureGroupsSetsWithingsSource verifies every row carries the source
// name. Source priority selects by this value, so a row without it lands in the
// lowest priority bucket and loses to Apple Health.
func TestMapMeasureGroupsSetsWithingsSource(t *testing.T) {
	groups := []MeasureGroup{{
		Date: 1754380000,
		Measures: []Measure{
			{Value: 76543, Type: typeWeight, Unit: -3},
			{Value: 118, Type: typeSystolic},
			{Value: 3160, Type: typeBoneMass, Unit: -3},
		},
	}}

	rows := MapMeasureGroups(groups, 1)
	if len(rows) == 0 {
		t.Fatal("no rows mapped")
	}
	for _, r := range rows {
		if r.Source != SourceName {
			t.Errorf("%s: source = %q, want %q", r.MetricName, r.Source, SourceName)
		}
		if r.Units == "" {
			t.Errorf("%s: units empty", r.MetricName)
		}
	}
}

// TestRequestedTypesCoversTheMap verifies the request list is derived from the
// mapping. A type requested without a target metric would be fetched and
// silently dropped; a mapped type left out of the request would never arrive.
func TestRequestedTypesCoversTheMap(t *testing.T) {
	types := RequestedTypes()
	if len(types) != len(measureTypeMap) {
		t.Fatalf("RequestedTypes() has %d entries, measureTypeMap has %d", len(types), len(measureTypeMap))
	}
	for i := 1; i < len(types); i++ {
		if types[i] <= types[i-1] {
			t.Fatalf("RequestedTypes() not sorted: %v", types)
		}
	}
	for _, tp := range types {
		if _, ok := measureTypeMap[tp]; !ok {
			t.Errorf("type %d requested but not mapped", tp)
		}
	}
}
