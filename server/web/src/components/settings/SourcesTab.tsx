import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import {
  fetchImportLogs,
  fetchSourcePriority,
  saveSourcePriority,
} from "../../api";
import { formatTimeAgo } from "../../utils/format";
import { TabHeader } from "./parts";

const DEFAULT_CATEGORY = "_default";

function sourceLabel(s: string): string {
  return s === "" ? "Apple Health (HealthKit)" : s;
}

/**
 * Surfaces the source-priority logic that already runs in the ingest path but
 * had no UI: when two devices report the same metric, the source higher in the
 * list wins.
 */
export default function SourcesTab() {
  const queryClient = useQueryClient();
  const config = useQuery({
    queryKey: ["source-priority"],
    queryFn: fetchSourcePriority,
  });
  const logs = useQuery({
    queryKey: ["import-logs", 50],
    queryFn: () => fetchImportLogs(50),
  });

  const [order, setOrder] = useState<string[]>([]);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!config.data) return;
    const rule = config.data.rules.find((r) => r.category === DEFAULT_CATEGORY);
    setOrder(rule?.sources ?? config.data.default ?? config.data.sources);
  }, [config.data]);

  const lastSyncBySource = new Map<string, string>();
  for (const log of logs.data ?? []) {
    if (log.source && !lastSyncBySource.has(log.source)) {
      lastSyncBySource.set(log.source, log.created_at);
    }
  }

  function move(index: number, direction: -1 | 1) {
    const target = index + direction;
    if (target < 0 || target >= order.length) return;
    const next = [...order];
    [next[index], next[target]] = [next[target], next[index]];
    setOrder(next);
  }

  async function save() {
    setSaving(true);
    setError(null);
    try {
      await saveSourcePriority(DEFAULT_CATEGORY, order);
      queryClient.invalidateQueries({ queryKey: ["source-priority"] });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(false);
    }
  }

  const saved =
    config.data?.rules.find((r) => r.category === DEFAULT_CATEGORY)?.sources ??
    config.data?.default ??
    [];
  const changed = order.join("|") !== saved.join("|");

  return (
    <>
      <TabHeader title="Sources">
        When two devices report the same metric, the source higher in this list
        wins. Move an entry up to prefer it.
      </TabHeader>

      <div style={{ paddingTop: 4 }}>
        {order.map((src, i) => {
          const lastSync = lastSyncBySource.get(src);
          return (
            <div
              key={src || "(healthkit)"}
              style={{
                display: "flex",
                alignItems: "center",
                gap: 16,
                padding: "14px 0",
                borderBottom: "1px solid var(--color-neutral-300)",
              }}
            >
              <span className="kick num" style={{ width: 20, flex: "none" }}>
                {i + 1}
              </span>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ font: "600 14px var(--font-body)" }}>
                  {sourceLabel(src)}
                </div>
                <div
                  style={{
                    font: "400 12px var(--font-body)",
                    color: "var(--color-neutral-600)",
                    marginTop: 2,
                  }}
                >
                  {i === 0 ? "Wins on conflict" : `Used when ranks 1–${i} are absent`}
                </div>
              </div>
              <span
                className={`tag ${lastSync ? "tag-accent" : "tag-neutral"}`}
                style={{ flex: "none" }}
              >
                {lastSync ? "Connected" : "Idle"}
              </span>
              <span
                style={{
                  width: 130,
                  flex: "none",
                  textAlign: "right",
                  font: "400 12px var(--font-body)",
                  color: "var(--color-neutral-600)",
                }}
              >
                {lastSync ? formatTimeAgo(lastSync) : "no ingest yet"}
              </span>
              <span style={{ display: "flex", gap: 4, flex: "none" }}>
                <button
                  type="button"
                  className="btn btn-secondary"
                  style={{ fontSize: 11, padding: "4px 8px" }}
                  disabled={i === 0}
                  onClick={() => move(i, -1)}
                  aria-label={`Move ${sourceLabel(src)} up`}
                >
                  ↑
                </button>
                <button
                  type="button"
                  className="btn btn-secondary"
                  style={{ fontSize: 11, padding: "4px 8px" }}
                  disabled={i === order.length - 1}
                  onClick={() => move(i, 1)}
                  aria-label={`Move ${sourceLabel(src)} down`}
                >
                  ↓
                </button>
              </span>
            </div>
          );
        })}
      </div>

      {error ? (
        <p style={{ color: "var(--color-accent-700)", fontSize: 13, marginTop: 14 }}>
          {error}
        </p>
      ) : null}

      <div style={{ paddingTop: 20 }}>
        <button
          type="button"
          className="btn btn-primary"
          onClick={save}
          disabled={saving || !changed}
        >
          {saving ? "Saving…" : "Save order"}
        </button>
      </div>
    </>
  );
}
