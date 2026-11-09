// Client transport policy. Values deliberately outlast server-side request deadlines.
export const CONNECTION_POLICY = {
  requestTimeoutMs: 7_000,
  pollIntervalMs: 3_000,
  staleAfterMs: 22_000,
  reconnectAfterMs: 40_000,
  statusRefreshMs: 30_000,
} as const;
