import { useCallback, useEffect, useState } from "react";
import {
  fetchHevyStatus,
  saveHevyCredentials,
  triggerHevySync,
  disconnectHevy,
  type HevyStatus,
} from "../../api";

/** Today as YYYY-MM-DD, the default ingest cutoff. */
function today(): string {
  return new Date().toISOString().slice(0, 10);
}

export default function HevyTab() {
  const [status, setStatus] = useState<HevyStatus | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [syncing, setSyncing] = useState(false);
  const [apiKey, setApiKey] = useState("");
  const [syncFrom, setSyncFrom] = useState(today());
  const [saving, setSaving] = useState(false);

  const load = useCallback(() => {
    setError(null);
    fetchHevyStatus()
      .then((s) => {
        setStatus(s);
        if (s.sync_from) setSyncFrom(s.sync_from);
      })
      .catch((e) => setError(e.message));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  if (error && !status) {
    return (
      <div>
        <p className="text-red-400 mb-4">{error}</p>
        <button onClick={load} className="text-sm text-cyan-400 hover:underline">
          Retry
        </button>
      </div>
    );
  }
  if (!status) {
    return <p className="text-zinc-500">Loading...</p>;
  }

  async function handleSave() {
    setSaving(true);
    setError(null);
    try {
      await saveHevyCredentials(apiKey, syncFrom);
      setApiKey("");
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to save API key");
    } finally {
      setSaving(false);
    }
  }

  async function handleSync() {
    setSyncing(true);
    setError(null);
    try {
      await triggerHevySync();
      setTimeout(load, 3000);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Sync failed");
    } finally {
      setSyncing(false);
    }
  }

  async function handleDisconnect() {
    if (!confirm("Disconnect Hevy? The API key is removed. Sets already imported stay in the database.")) {
      return;
    }
    try {
      await disconnectHevy();
      setApiKey("");
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Disconnect failed");
    }
  }

  // Step 1: No API key stored yet.
  if (!status.configured) {
    return (
      <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-6">
        <h3 className="text-zinc-200 font-medium mb-2">Connect Hevy</h3>
        <p className="text-zinc-400 text-sm mb-4">
          Create an API key at{" "}
          <a
            href="https://hevy.com/settings?developer"
            target="_blank"
            rel="noopener noreferrer"
            className="text-cyan-400 hover:underline"
          >
            hevy.com/settings
          </a>
          . API access requires an active Hevy Pro subscription.
        </p>
        {error && <p className="text-red-400 text-sm mb-4">{error}</p>}
        <div className="space-y-3 mb-4">
          <div>
            <label className="block text-xs text-zinc-500 uppercase tracking-wide mb-1">
              API Key
            </label>
            <input
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              className="w-full bg-zinc-800 border border-zinc-700 rounded px-3 py-2 text-sm text-zinc-200 font-mono focus:outline-none focus:border-cyan-600"
              placeholder="Hevy API key"
            />
          </div>
          <div>
            <label className="block text-xs text-zinc-500 uppercase tracking-wide mb-1">
              Import From
            </label>
            <input
              type="date"
              value={syncFrom}
              onChange={(e) => setSyncFrom(e.target.value)}
              className="w-full bg-zinc-800 border border-zinc-700 rounded px-3 py-2 text-sm text-zinc-200 font-mono focus:outline-none focus:border-cyan-600"
            />
            <p className="text-xs text-zinc-500 mt-1">
              Workouts that started before this date are ignored. Keep it at the switchover
              date so an Alpha Progression history later uploaded to Hevy cannot be counted
              a second time.
            </p>
          </div>
        </div>
        <button
          onClick={handleSave}
          disabled={saving || !apiKey}
          className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white rounded-lg font-medium transition-colors"
        >
          {saving ? "Verifying..." : "Save API Key"}
        </button>
      </div>
    );
  }

  // Step 2: Connected.
  return (
    <div>
      {error && <p className="text-red-400 text-sm mb-4">{error}</p>}
      <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4 mb-4">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <span className="w-2 h-2 bg-emerald-400 rounded-full" />
            <span className="text-zinc-200 font-medium">Connected</span>
          </div>
          <div className="flex gap-2">
            <button
              onClick={handleSync}
              disabled={syncing}
              className="px-3 py-1.5 text-sm bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white rounded-md transition-colors"
            >
              {syncing ? "Syncing..." : "Sync Now"}
            </button>
            <button
              onClick={handleDisconnect}
              className="px-3 py-1.5 text-sm bg-zinc-700 hover:bg-zinc-600 text-zinc-300 rounded-md transition-colors"
            >
              Disconnect
            </button>
          </div>
        </div>
        <p className="text-xs text-zinc-500">Importing workouts from {status.sync_from}</p>
        {status.last_sync && (
          <p className="text-xs text-zinc-500">
            Last sync: {new Date(status.last_sync).toLocaleString()}
          </p>
        )}
      </div>

      <div className="bg-zinc-900 border border-zinc-800 rounded-lg p-4">
        <h3 className="text-sm font-medium text-zinc-300 mb-2">Replace API Key</h3>
        <div className="flex gap-2">
          <input
            type="password"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            className="flex-1 bg-zinc-800 border border-zinc-700 rounded px-3 py-2 text-sm text-zinc-200 font-mono focus:outline-none focus:border-cyan-600"
            placeholder="New Hevy API key"
          />
          <button
            onClick={handleSave}
            disabled={saving || !apiKey}
            className="px-4 py-2 text-sm bg-zinc-700 hover:bg-zinc-600 disabled:opacity-50 text-zinc-200 rounded-lg transition-colors"
          >
            {saving ? "Verifying..." : "Save"}
          </button>
        </div>
      </div>
    </div>
  );
}
