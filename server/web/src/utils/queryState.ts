import type { UseQueryResult } from "@tanstack/react-query";

/**
 * What a screen should render for a query that has not produced data.
 *
 * "offline" is its own state rather than part of "loading": when the browser
 * reports no connection, React Query pauses the fetch, which leaves isLoading
 * false and error null. Folding that into the empty state would have the page
 * claim there is nothing to show when the truth is that it never asked — a real
 * risk here, since the server is reachable over Tailscale only.
 */
export type QueryState = "loading" | "offline" | "error" | "ready";

export function queryState(query: UseQueryResult<unknown>): QueryState {
  if (query.data !== undefined) return "ready";
  if (query.error) return "error";
  if (query.fetchStatus === "paused") return "offline";
  if (query.isPending || query.isFetching) return "loading";
  return "ready";
}

/** The sentence a screen shows in place of its content. */
export function queryMessage(
  state: QueryState,
  error: unknown,
): string | null {
  switch (state) {
    case "offline":
      return "No connection to the server. Check that you are on the tailnet.";
    case "error":
      return `The data could not be loaded. ${(error as Error)?.message ?? ""}`.trim();
    default:
      return null;
  }
}
