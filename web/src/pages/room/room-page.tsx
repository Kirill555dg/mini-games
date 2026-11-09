import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "preact/hooks";
import { BOARD_LAYOUT } from "@/entities/game/board-layout";
import { GameBoard, gameTitle, replayState } from "@/entities/game/game-board";
import { api, type Command, type GameSpec, type RoomView, unwrap } from "@/shared/api/game-api";
import { live } from "@/shared/api/live-connection";
import { GAME_STATE, ROOM_ACTION } from "@/shared/api/protocol.generated";
import { sideName, t, translated } from "@/shared/i18n/i18n";
import { invitation } from "@/shared/invitations/invitations";
import { Button, ConfirmLeave, Notice } from "@/shared/ui/ui-controls";
import { MoveReview } from "@/widgets/replay/move-review";

export function RoomPage({
  id,
  catalog,
  connection,
}: {
  id: string;
  catalog: GameSpec[];
  connection: (online: boolean) => void;
}) {
  const [current, setCurrent] = useState<RoomView>(),
    [closed, setClosed] = useState(false),
    [online, setOnline] = useState(false),
    [busy, setBusy] = useState(false);
  const [message, setMessage] = useState(""),
    [error, setError] = useState(""),
    [cursor, setCursor] = useState<number | null>(null),
    [vertical, setVertical] = useState(false),
    [ownField, setOwnField] = useState(true),
    [confirm, setConfirm] = useState(false);
  const inFlight = useRef(false),
    alive = useRef(true),
    stage = useRef<HTMLDivElement>(null),
    controls = useRef<HTMLDivElement>(null);
  const accept = useCallback((value: RoomView | null) => {
    if (!value) {
      setClosed(true);
      return;
    }
    setCurrent((previous) =>
      !previous || value.instance !== previous.instance || value.revision >= previous.revision
        ? value
        : previous,
    );
  }, []);
  useEffect(() => {
    alive.current = true;
    const stop = live<RoomView>(id, accept, (value) => {
      setOnline(value);
      connection(value);
    });
    return () => {
      alive.current = false;
      stop();
    };
  }, [id, accept, connection]);
  useEffect(() => {
    setCursor(null);
    setOwnField(current?.state.phase === "setup");
  }, [current?.instance, current?.round]);
  useEffect(() => {
    if (current?.state.phase === GAME_STATE.playing) setOwnField(false);
  }, [current?.state.phase]);
  async function act(action: Command["action"], values: Partial<Command> = {}) {
    if (inFlight.current) return false;
    inFlight.current = true;
    setBusy(true);
    setError("");
    setMessage("");
    try {
      const value = await unwrap(
        api.POST("/game-api/action", { body: { action, room: id, round: current?.round, ...values } }),
      );
      if (alive.current) accept(value);
      return true;
    } catch (failure) {
      if (alive.current) setError(failure instanceof Error ? failure.message : t("error503"));
      try {
        const value = await unwrap(api.GET("/game-api/state", { params: { query: { room: id } } }));
        if (alive.current) accept(value);
      } catch {
        /* The live connection reports room closure or network failure. */
      }
      return false;
    } finally {
      inFlight.current = false;
      if (alive.current) setBusy(false);
    }
  }
  const fit = useCallback(() => {
    if (!stage.current || !current) return;
    const ratio = current.room.game === "connectfour" ? current.state.width / current.state.height : 1;
    const available = Math.max(
      BOARD_LAYOUT.minHeight,
      (window.visualViewport?.height || innerHeight) -
        stage.current.getBoundingClientRect().top -
        (controls.current?.offsetHeight || 0) -
        BOARD_LAYOUT.footerReserve,
    );
    stage.current.style.setProperty(
      "--board-size",
      `${Math.floor(Math.min(BOARD_LAYOUT.maxWidth, stage.current.clientWidth, available * ratio))}px`,
    );
  }, [current]);
  useLayoutEffect(fit);
  useEffect(() => {
    const resize = () => requestAnimationFrame(fit);
    window.addEventListener("resize", resize);
    window.visualViewport?.addEventListener("resize", resize);
    return () => {
      window.removeEventListener("resize", resize);
      window.visualViewport?.removeEventListener("resize", resize);
    };
  }, [fit]);
  async function leave() {
    setConfirm(false);
    const left = !current?.name || (await act(ROOM_ACTION.leave));
    if (alive.current && left) location.hash = "/";
  }
  async function share(watch = false) {
    const url = invitation(id, { watch });
    try {
      await navigator.clipboard.writeText(url);
      setMessage(t("copied"));
    } catch {
      setMessage(t("copyFallback", { code: id, url }));
    }
  }
  if (closed)
    return (
      <section class="room-page">
        <Notice>{t("roomClosed")}</Notice>
        {current?.room.nextRoom && (
          <a
            class="button primary"
            href={invitation(current.room.nextRoom, { watch: current.role < 0, seat: current.role })}
          >
            {t("continue")}
          </a>
        )}{" "}
        <a class="button" href="/#/">
          {t("backGames")}
        </a>
      </section>
    );
  if (!current)
    return (
      <section class="room-page">
        <Notice>{t("loading")}</Notice>
        <Button
          onClick={() => {
            location.hash = "/";
          }}
        >
          {t("backGames")}
        </Button>
      </section>
    );
  const { room, state, role } = current,
    player = role >= 0,
    spec = catalog.find((game) => game.id === room.game);
  const total = state.moves?.length || 0,
    index = cursor === null ? total : Math.min(cursor, total),
    reviewing = index < total;
  let text: string;
  if (reviewing) text = t("reviewMove", { n: index });
  else if (room.finished)
    text =
      current.reason && state.winner === GAME_STATE.draw
        ? t("interrupted")
        : state.winner === GAME_STATE.draw
          ? t("draw")
          : state.winner === role
            ? t("victory")
            : t("winner", { name: room.seats[state.winner]?.name || t("player", { n: state.winner + 1 }) });
  else if (!room.started)
    text = room.seats.every((seat) => seat.taken)
      ? t("waitingReady")
      : t("waitingPlayers", { n: room.seats.filter((seat) => seat.taken).length, total: room.seats.length });
  else if (state.phase === "setup")
    text = player ? t(state.ready?.[role] ? "fleetReady" : "placeFleet") : t("fleetsSetup");
  else text = t(role === state.turn ? "yourTurn" : "turn", { name: room.seats[state.turn]?.name || "" });
  const canMove =
    online &&
    !busy &&
    player &&
    room.started &&
    !room.finished &&
    !reviewing &&
    (state.phase === "setup" || state.turn === role);
  const showOwn = state.phase === "setup" || ownField;
  const goal = state.winLength
    ? t("lineGoal", { width: state.width, height: state.height, n: state.winLength })
    : t(room.game === "dotsboxes" ? "dotsGoal" : "navalGoal");
  return (
    <section class="room-page">
      <div class="room-heading">
        <div>
          <h1>{gameTitle(catalog, room.game)}</h1>
          <div class="room-meta">
            {t("room")} <span class="room-code">{id}</span>
            <span class="room-goal">{goal}</span>
          </div>
        </div>
        <div class="actions">
          <Button class="small" onClick={() => void share()}>
            {t("invite")}
          </Button>
          <Button
            class="small danger"
            disabled={busy}
            onClick={() => {
              if (player && room.started && !room.finished) setConfirm(true);
              else void leave();
            }}
          >
            {t("leave")}
          </Button>
        </div>
      </div>
      <p
        class={`game-status ${online && player && role === state.turn && room.started && !room.finished ? "your-turn" : ""}`}
        role="status"
      >
        {text}
        {!online && ` · ${t("reconnecting")}`}
      </p>
      {error && <Notice error>{error}</Notice>}
      {message && <Notice>{message}</Notice>}
      {room.nextRoom && (
        <Notice>
          <a
            class="text-link"
            href={invitation(room.nextRoom, { watch: !player, seat: player ? role : undefined })}
          >
            {t("continue")} · {room.nextRoom}
          </a>
        </Notice>
      )}
      <div class="room-layout">
        <div class="play-area">
          <div ref={stage} class="game-stage">
            <GameBoard
              game={room.game}
              state={replayState(room.game, state, index, role)}
              actor={role}
              enabled={canMove}
              own={room.game === "battleship" && showOwn}
              vertical={vertical}
              onMove={(move) => void act(ROOM_ACTION.move, move)}
            />
          </div>
          <div ref={controls} class="game-controls">
            {!room.started && !room.finished && player && (
              <div class="preparation">
                <FieldSide
                  spec={spec}
                  role={role}
                  seats={room.seats.map((seat) => seat.name)}
                  busy={busy || !online}
                  change={(seat) => void act(ROOM_ACTION.seat, { seat })}
                />
                <Button
                  class={room.seats[role].ready ? "" : "primary"}
                  disabled={busy || !online || Boolean(room.swap)}
                  onClick={() => void act(ROOM_ACTION.ready)}
                >
                  {t(room.seats[role].ready ? "notReady" : "ready")}
                </Button>
              </div>
            )}
            {room.swap && (
              <Notice>
                {t("swapRequest", { from: sideName(spec, room.swap.from), to: sideName(spec, room.swap.to) })}
                {player && (role === room.swap.from || role === room.swap.to) && (
                  <div class="actions">
                    {role === room.swap.to && (
                      <Button
                        class="small primary"
                        disabled={busy}
                        onClick={() => void act(ROOM_ACTION.acceptSwap)}
                      >
                        {t("acceptSwap")}
                      </Button>
                    )}
                    <Button class="small" disabled={busy} onClick={() => void act(ROOM_ACTION.cancelSwap)}>
                      {t("declineSwap")}
                    </Button>
                  </div>
                )}
              </Notice>
            )}
            {room.game === "battleship" &&
              room.started &&
              (state.phase === "setup" && player && !state.ready?.[role] ? (
                <>
                  <p class="fleet-help">
                    {state.fleet?.length
                      ? t("nextShip", { n: state.fleet[0], fleet: state.fleet.join(", ") })
                      : t("fleetPlaced")}
                  </p>
                  <div class="actions fleet-actions">
                    <Button class="small" disabled={busy} onClick={() => setVertical(!vertical)}>
                      {t(vertical ? "vertical" : "horizontal")}
                    </Button>
                    <Button
                      class="small"
                      disabled={busy || !online}
                      onClick={() => void act(ROOM_ACTION.move, { kind: "shuffle" })}
                    >
                      {t("shuffle")}
                    </Button>
                    <Button
                      class="small"
                      disabled={busy || !online}
                      onClick={() => void act(ROOM_ACTION.move, { kind: "clear" })}
                    >
                      {t("clear")}
                    </Button>
                    <Button
                      class="small primary"
                      disabled={busy || !online || Boolean(state.fleet?.length)}
                      onClick={() => void act(ROOM_ACTION.move, { kind: "ready" })}
                    >
                      {t("ready")}
                    </Button>
                  </div>
                </>
              ) : state.phase !== "setup" ? (
                <fieldset class="field-switch" aria-label={t("board")}>
                  {[false, true].map((own) => (
                    <Button
                      key={String(own)}
                      class={`small ${ownField === own ? "selected" : ""}`}
                      aria-pressed={ownField === own}
                      onClick={() => setOwnField(own)}
                    >
                      {player
                        ? t(own ? "yourField" : "enemyField")
                        : t("shotsBy", { name: room.seats[own ? 1 : 0].name })}
                    </Button>
                  ))}
                </fieldset>
              ) : null)}
            {(room.finished || (total > 0 && room.game !== "battleship")) && (
              <MoveReview
                index={index}
                total={total}
                change={(next) => setCursor(next === total ? null : next)}
              />
            )}
            {room.finished && player && (
              <a class="button primary rematch" href={`/#/new/${room.game}?from=${id}`}>
                {t("newGame")}
              </a>
            )}
            {!player && !current.name && (
              <a class="button primary" href={invitation(id)}>
                {t("continue")}
              </a>
            )}
            {!player &&
              current.name &&
              !room.started &&
              !room.finished &&
              room.seats.some((seat) => !seat.taken) && (
                <Button
                  class="primary"
                  disabled={busy || !online}
                  onClick={() => void act(ROOM_ACTION.enter, { name: current.name })}
                >
                  {t("play")}
                </Button>
              )}
          </div>
        </div>
        <aside class="room-sidebar" aria-label={t("playerCount")}>
          <div class="seat-list">
            {room.seats.map((seat, seatIndex) => (
              <div
                class={`seat ${seatIndex === role ? "is-you" : ""} player-${seatIndex + 1}`}
                key={seatIndex}
              >
                <span class="seat-number">{seatIndex + 1}</span>
                <div>
                  <strong>{seat.name || t("free")}</strong>
                  <span class="seat-presence">
                    {sideName(spec, seatIndex)} ·{" "}
                    {t(
                      !seat.taken
                        ? "waitingPlayer"
                        : state.phase === "setup" && room.started
                          ? state.ready?.[seatIndex]
                            ? "fleetPrepared"
                            : "placingFleet"
                          : !room.started && seat.ready
                            ? "readyLabel"
                            : seatIndex === role
                              ? "you"
                              : seat.online
                                ? "online"
                                : "returnGrace",
                    )}
                  </span>
                </div>
                {state.scores?.length ? <strong class="seat-score">{state.scores[seatIndex]}</strong> : null}
              </div>
            ))}
          </div>
          <p class="spectator-count muted">{t("spectators", { n: room.spectators })}</p>
          <details class="room-rules">
            <summary>{t("rules")}</summary>
            <p>{translated(spec, "rules")}</p>
          </details>
          <Button class="small spectator-invite" onClick={() => void share(true)}>
            {t("inviteWatch")}
          </Button>
        </aside>
      </div>
      {confirm && <ConfirmLeave code={id} cancel={() => setConfirm(false)} confirm={() => void leave()} />}
    </section>
  );
}
function FieldSide({
  spec,
  role,
  seats,
  busy,
  change,
}: {
  spec?: GameSpec;
  role: number;
  seats: string[];
  busy: boolean;
  change: (seat: number) => void;
}) {
  return (
    <label class="side-choice">
      {t("yourSide")}
      <select
        value={role}
        aria-label={t("chooseSeat")}
        disabled={busy}
        onChange={(event) => change(Number(event.currentTarget.value))}
      >
        {seats.map((name, index) => (
          <option value={index} key={index}>
            {sideName(spec, index)}
            {index !== role && name ? ` · ${name}` : ""}
          </option>
        ))}
      </select>
    </label>
  );
}
