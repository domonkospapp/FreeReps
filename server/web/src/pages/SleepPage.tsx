import { useQuery } from "@tanstack/react-query";
import { useMemo, type ReactNode } from "react";
import { useSearchParams } from "react-router-dom";
import { fetchSleep, type SleepSession, type SleepStage } from "../api";
import PageHeader from "../components/PageHeader";
import RangeControl from "../components/RangeControl";
import Hypnogram, { hourTicks } from "../components/sleep/Hypnogram";
import NightsChart from "../components/sleep/NightsChart";
import StageComposition, {
  type StageTotals,
} from "../components/sleep/StageComposition";
import { useIsDesktop } from "../hooks/useMediaQuery";
import {
  formatClock,
  formatDayMonth,
  formatHoursMinutes,
} from "../utils/format";
import { stageColor } from "../utils/stageColors";
import { queryMessage, queryState } from "../utils/queryState";

const RANGES = ["7d", "30d", "90d"] as const;
type Range = (typeof RANGES)[number];

const RANGE_DAYS: Record<Range, number> = { "7d": 7, "30d": 30, "90d": 90 };

export default function SleepPage() {
  const isDesktop = useIsDesktop();
  const [params, setParams] = useSearchParams();
  const range = (params.get("range") as Range) ?? "30d";

  const days = RANGE_DAYS[range];
  const end = new Date();
  const start = new Date(end.getTime() - days * 86400000);
  const endISO = end.toISOString().split("T")[0];
  const startISO = start.toISOString().split("T")[0];

  const query = useQuery({
    queryKey: ["sleep", startISO, endISO],
    queryFn: () => fetchSleep(startISO, endISO),
  });
  const state = queryState(query);
  const message = queryMessage(state, query.error);

  const sessions = query.data?.sessions ?? [];
  const stages = query.data?.stages ?? [];

  // The API returns newest first; the last night is the summary's subject.
  const last = sessions.length > 0 ? sessions[0] : null;

  const lastNightStages = useMemo(
    () => (last ? stagesForSession(stages, last) : []),
    [stages, last],
  );

  const totals = useMemo(() => stageTotals(lastNightStages), [lastNightStages]);

  const setRange = (next: Range) => {
    const p = new URLSearchParams(params);
    p.set("range", next);
    setParams(p, { replace: true });
  };

  const averageHours =
    sessions.length > 0
      ? sessions.reduce((a, s) => a + s.TotalSleep, 0) / sessions.length
      : null;

  const awakenings = lastNightStages.filter((s) => s.Stage === "Awake").length;

  return (
    <>
      <PageHeader
        kicker={
          last
            ? `Last night — ${new Date(last.Date).toLocaleDateString("en-GB", {
                weekday: "long",
                day: "numeric",
                month: "long",
              })}`
            : "No sleep recorded"
        }
        title="Sleep"
        actions={
          <RangeControl
            options={RANGES}
            value={range}
            onChange={setRange}
            name="sleep-range"
          />
        }
      />

      {message ? (
        <p
          className="page-x"
          style={{ color: "var(--color-neutral-600)", fontSize: 13 }}
        >
          {message}
        </p>
      ) : state === "loading" ? (
        <div
          className="page-x"
          style={{ borderTop: "2px solid var(--color-text)", paddingTop: 24 }}
        >
          <span className="skel" style={{ width: 180, height: 44 }} />
        </div>
      ) : !last ? (
        <p
          className="page-x"
          style={{ color: "var(--color-neutral-600)", fontSize: 13 }}
        >
          No sleep sessions in this window.
        </p>
      ) : isDesktop ? (
        <DesktopSleep
          session={last}
          stages={lastNightStages}
          totals={totals}
          sessions={sessions}
          allStages={stages}
          averageHours={averageHours}
          awakenings={awakenings}
        />
      ) : (
        <MobileSleep
          session={last}
          stages={lastNightStages}
          totals={totals}
          sessions={sessions}
        />
      )}
    </>
  );
}

function DesktopSleep({
  session,
  stages,
  totals,
  sessions,
  allStages,
  averageHours,
  awakenings,
}: {
  session: SleepSession;
  stages: SleepStage[];
  totals: StageTotals;
  sessions: SleepSession[];
  allStages: SleepStage[];
  averageHours: number | null;
  awakenings: number;
}) {
  const efficiency =
    session.InBed > 0 ? (session.Asleep / session.InBed) * 100 : null;
  const ticks = hourTicks(stages);

  return (
    <>
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(5, 1fr)",
          borderTop: "2px solid var(--color-text)",
          borderBottom: "2px solid var(--color-text)",
        }}
      >
        <HeroCell
          label="Total sleep"
          value={formatHoursMinutes(session.TotalSleep)}
          meta={
            averageHours != null
              ? `${formatHoursMinutes(averageHours)} on average`
              : ""
          }
        />
        <HeroCell
          label="Time in bed"
          value={formatHoursMinutes(session.InBed)}
          meta={`${formatClock(session.InBedStart)} → ${formatClock(session.InBedEnd)}`}
        />
        <HeroCell
          label="Efficiency"
          value={efficiency != null ? `${efficiency.toFixed(0)}%` : "—"}
          meta={`${formatHoursMinutes(totals.Awake)} awake`}
        />
        <HeroCell
          label="Deep"
          value={formatHoursMinutes(totals.Deep)}
          meta={pctOf(totals.Deep, session.TotalSleep)}
        />
        <HeroCell
          label="REM"
          value={formatHoursMinutes(totals.REM)}
          meta={pctOf(totals.REM, session.TotalSleep)}
        />
      </div>

      <Section title="Stage composition">
        <StageComposition totals={totals} />
      </Section>

      <Section
        title="Hypnogram"
        aside={`${formatClock(session.SleepStart)} → ${formatClock(session.SleepEnd)} · ${awakenings} awakening${awakenings === 1 ? "" : "s"}`}
      >
        <Hypnogram stages={stages} />
        <div style={{ position: "relative", height: 20, marginLeft: 52 }}>
          {ticks.map((t) => (
            <span
              key={t.pct}
              className="num"
              style={{
                position: "absolute",
                left: `${t.pct}%`,
                transform: "translateX(-50%)",
                font: "400 11px var(--font-body)",
                color: "var(--color-neutral-600)",
              }}
            >
              {t.label}
            </span>
          ))}
        </div>
      </Section>

      <Section
        title={`Last ${sessions.length} nights`}
        aside={
          averageHours != null ? (
            <>
              {formatDayMonth(new Date(sessions[sessions.length - 1].Date))} –{" "}
              {formatDayMonth(new Date(sessions[0].Date))} · average{" "}
              <span style={{ fontWeight: 700, color: "var(--color-text)" }}>
                {formatHoursMinutes(averageHours)}
              </span>
            </>
          ) : null
        }
      >
        <NightsChart sessions={sessions} stages={allStages} />
      </Section>
    </>
  );
}

function MobileSleep({
  session,
  stages,
  totals,
  sessions,
}: {
  session: SleepSession;
  stages: SleepStage[];
  totals: StageTotals;
  sessions: SleepSession[];
}) {
  const efficiency =
    session.InBed > 0 ? (session.Asleep / session.InBed) * 100 : null;
  const latency =
    (new Date(session.SleepStart).getTime() -
      new Date(session.InBedStart).getTime()) /
    60000;

  return (
    <>
      <div className="page-x" style={{ paddingBottom: 16 }}>
        <div style={{ display: "flex", alignItems: "baseline", gap: 12 }}>
          <span
            className="num"
            style={{
              font: "800 44px/1 var(--font-heading)",
              letterSpacing: "-0.035em",
            }}
          >
            {formatHoursMinutes(session.TotalSleep)}
          </span>
          <span
            style={{
              font: "500 13px var(--font-body)",
              color: "var(--color-neutral-600)",
            }}
          >
            in bed {formatHoursMinutes(session.InBed)}
            {efficiency != null ? ` · ${efficiency.toFixed(0)}% eff.` : ""}
          </span>
        </div>
      </div>

      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(3, 1fr)",
          borderTop: "2px solid var(--color-text)",
          borderBottom: "2px solid var(--color-text)",
        }}
      >
        <MobileStat label="Deep" value={formatHoursMinutes(totals.Deep)} />
        <MobileStat
          label="Efficiency"
          value={efficiency != null ? `${efficiency.toFixed(0)}%` : "—"}
        />
        <MobileStat
          label="Latency"
          value={latency > 0 ? `${Math.round(latency)}m` : "—"}
        />
      </div>

      <div className="kick page-x" style={{ paddingTop: 16, paddingBottom: 6 }}>
        Hypnogram · {formatClock(session.SleepStart)} →{" "}
        {formatClock(session.SleepEnd)}
      </div>
      <div className="page-x" style={{ paddingBottom: 14 }}>
        <Hypnogram stages={stages} compact />
      </div>

      <div className="page-x" style={{ paddingBottom: 16 }}>
        <StageComposition totals={totals} compact />
      </div>

      <div
        className="kick page-x"
        style={{
          borderTop: "2px solid var(--color-text)",
          paddingTop: 12,
          paddingBottom: 6,
        }}
      >
        Last {Math.min(sessions.length, 10)} nights
      </div>
      <div>
        {sessions.slice(0, 10).map((s) => (
          <NightRow key={s.Date} session={s} />
        ))}
      </div>
    </>
  );
}

function NightRow({ session }: { session: SleepSession }) {
  const segments = (
    [
      ["Deep", session.Deep],
      ["REM", session.REM],
      ["Core", session.Core],
    ] as const
  ).filter(([, v]) => v > 0);
  const total = segments.reduce((a, [, v]) => a + v, 0) || 1;

  return (
    <div className="row" style={{ paddingTop: 10, paddingBottom: 10 }}>
      <span style={{ font: "500 13px var(--font-body)", width: 44, flex: "none" }}>
        {new Date(session.Date).toLocaleDateString("en-GB", {
          weekday: "short",
        })}
      </span>
      <div style={{ flex: 1, display: "flex", height: 12 }}>
        {segments.map(([stage, value]) => (
          <div
            key={stage}
            style={{ flex: value / total, background: stageColor(stage) }}
          />
        ))}
      </div>
      <span
        className="num"
        style={{
          font: "600 13px var(--font-body)",
          width: 46,
          textAlign: "right",
          flex: "none",
        }}
      >
        {formatHoursMinutes(session.TotalSleep)}
      </span>
    </div>
  );
}

function Section({
  title,
  aside,
  children,
}: {
  title: string;
  aside?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div style={{ borderTop: "2px solid var(--color-text)", marginTop: 26 }}>
      <div
        className="flex items-baseline justify-between gap-5 page-x"
        style={{ paddingTop: 20, paddingBottom: 16 }}
      >
        <h2 style={{ fontSize: 19, fontWeight: 700 }}>{title}</h2>
        {aside ? (
          <span
            style={{
              font: "400 12px var(--font-body)",
              color: "var(--color-neutral-600)",
            }}
          >
            {aside}
          </span>
        ) : null}
      </div>
      <div className="page-x" style={{ paddingBottom: 30 }}>
        {children}
      </div>
    </div>
  );
}

function HeroCell({
  label,
  value,
  meta,
}: {
  label: string;
  value: string;
  meta: string;
}) {
  return (
    <div
      className="page-x"
      style={{
        paddingTop: 24,
        paddingBottom: 22,
        borderRight: "1px solid var(--color-divider)",
      }}
    >
      <div className="kick">{label}</div>
      <div
        className="num"
        style={{
          font: "800 44px/1 var(--font-heading)",
          letterSpacing: "-0.035em",
          marginTop: 14,
        }}
      >
        {value}
      </div>
      <div
        style={{
          font: "400 12px var(--font-body)",
          color: "var(--color-neutral-600)",
          marginTop: 12,
        }}
      >
        {meta}
      </div>
    </div>
  );
}

function MobileStat({ label, value }: { label: string; value: string }) {
  return (
    <div
      className="page-x"
      style={{
        paddingTop: 12,
        paddingBottom: 12,
        borderRight: "1px solid var(--color-divider)",
      }}
    >
      <div className="kick">{label}</div>
      <div
        className="num"
        style={{
          font: "700 22px/1 var(--font-heading)",
          letterSpacing: "-0.02em",
          marginTop: 8,
        }}
      >
        {value}
      </div>
    </div>
  );
}

function pctOf(part: number, whole: number): string {
  if (whole <= 0) return "";
  return `${((part / whole) * 100).toFixed(0)}% of sleep`;
}

function stagesForSession(
  stages: SleepStage[],
  session: SleepSession,
): SleepStage[] {
  const from = new Date(session.SleepStart).getTime();
  const to = new Date(session.SleepEnd).getTime();
  return stages.filter((s) => {
    const t = new Date(s.StartTime).getTime();
    return t >= from && t < to;
  });
}

function stageTotals(stages: SleepStage[]): StageTotals {
  const totals: StageTotals = { Deep: 0, Core: 0, REM: 0, Awake: 0 };
  for (const s of stages) {
    if (s.Stage in totals) {
      totals[s.Stage as keyof StageTotals] += s.DurationHr;
    }
  }
  return totals;
}
