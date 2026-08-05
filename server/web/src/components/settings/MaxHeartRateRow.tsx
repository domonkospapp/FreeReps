import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { fetchMaxHeartRate, saveMaxHeartRate } from "../../api";
import { formatNumber } from "../../utils/format";
import { Row } from "./parts";

/**
 * The training zones derive from this figure. Left unset it is measured from
 * the workout history, which makes the bands move retroactively after one hard
 * session — a configured maximum is a physiological constant and holds still.
 */
export default function MaxHeartRateRow() {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: ["max-heart-rate"],
    queryFn: fetchMaxHeartRate,
  });

  const [draft, setDraft] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    if (loaded || !query.data) return;
    setDraft(
      query.data.origin === "configured" ? String(Math.round(query.data.bpm)) : "",
    );
    setLoaded(true);
  }, [loaded, query.data]);

  async function commit(bpm: number) {
    setSaving(true);
    setError(null);
    try {
      await saveMaxHeartRate(bpm);
      await queryClient.invalidateQueries({ queryKey: ["max-heart-rate"] });
      // The zone bars read the same figure.
      queryClient.invalidateQueries({ queryKey: ["workout-zones"] });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(false);
    }
  }

  const data = query.data;
  const configured = data?.origin === "configured";

  return (
    <Row label="Max heart rate">
      <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
        <input
          className="input num"
          style={{ width: 110 }}
          type="number"
          min={100}
          max={250}
          inputMode="numeric"
          value={draft}
          placeholder={data ? String(Math.round(data.observed)) : ""}
          onChange={(e) => setDraft(e.target.value)}
          aria-label="Maximum heart rate in bpm"
        />
        <span
          style={{
            font: "400 12px var(--font-body)",
            color: "var(--color-neutral-600)",
          }}
        >
          bpm
        </span>
        <button
          type="button"
          className="btn btn-secondary"
          style={{ fontSize: 12 }}
          disabled={saving || draft === ""}
          onClick={() => commit(Number(draft))}
        >
          {saving ? "Saving…" : "Save"}
        </button>
        {configured ? (
          <button
            type="button"
            className="btn btn-ghost"
            style={{ fontSize: 12 }}
            disabled={saving}
            onClick={() => {
              setDraft("");
              commit(0);
            }}
          >
            Use measured
          </button>
        ) : null}
      </div>

      <p
        style={{
          font: "400 12px/1.5 var(--font-body)",
          color: "var(--color-neutral-600)",
          margin: "8px 0 0",
          maxWidth: "56ch",
        }}
      >
        {error ? (
          <span style={{ color: "var(--color-accent-700)" }}>{error}</span>
        ) : data ? (
          configured ? (
            <>
              Zones derive from your own figure. The history measures{" "}
              <span className="num">{formatNumber(data.observed, 0)}</span> bpm.
            </>
          ) : (
            <>
              Unset, so zones derive from the highest rate your workouts
              recorded — currently{" "}
              <span className="num">{formatNumber(data.observed, 0)}</span> bpm.
              That figure rises after a hard session and shifts every band with
              it.
            </>
          )
        ) : (
          "Loading…"
        )}
      </p>
    </Row>
  );
}
