import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "preact/hooks";
import { BOARD_LAYOUT } from "@/entities/game/board-layout";
import { GameBoard, gameTitle } from "@/entities/game/game-board";
import { GameRules } from "@/entities/game/game-rules";
import { api, type Command, type GameSpec, type RoomView, unwrap } from "@/shared/api/game-api";
import { live } from "@/shared/api/live-connection";
import { GAME_STATE, ROOM_ACTION } from "@/shared/api/protocol.generated";
import { sideName, t } from "@/shared/i18n/i18n";
import { invitation } from "@/shared/invitations/invitations";
import { ActionIcon } from "@/shared/ui/action-icon";
import { CopyControl } from "@/shared/ui/copy-control";
import { Button, ConfirmLeave, Mark, Notice } from "@/shared/ui/ui-controls";
import { NavalGame } from "@/widgets/naval/naval-game";

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
  const [error, setError] = useState(""),
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
  async function act(action: Command["action"], values: Partial<Command> = {}) {
    if (inFlight.current) return false;
    inFlight.current = true;
    setBusy(true);
    setError("");
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
        /* Live connection reports recovery. */
      }
      return false;
    } finally {
      inFlight.current = false;
      if (alive.current) setBusy(false);
    }
  }
  const fit = useCallback(() => {
    if (!stage.current || !current || current.room.game === "battleship") return;
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
  function requestLeave() {
    if (current && current.role >= 0 && current.room.started && !current.room.finished) setConfirm(true);
    else void leave();
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
        )}
        <a class="button" href="/#/">
          <ActionIcon name="back" />
          {t("backGames")}
        </a>
      </section>
    );
  if (!current)
    return (
      <section class="room-page">
        <Button
          onClick={() => {
            location.hash = "/";
          }}
        >
          <ActionIcon name="back" />
          {t("backGames")}
        </Button>
        <Notice>{t("loading")}</Notice>
      </section>
    );
  const { room, state, role } = current,
    player = role >= 0,
    spec = catalog.find((game) => game.id === room.game),
    naval = room.game === "battleship";
  let text: string;
  if (room.finished)
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
  else if (state.phase === GAME_STATE.setup)
    text = player ? t(state.ready?.[role] ? "fleetReady" : "placeFleet") : t("fleetsSetup");
  else text = t(role === state.turn ? "yourTurn" : "turn", { name: room.seats[state.turn]?.name || "" });
  const canMove =
    online &&
    !busy &&
    player &&
    room.started &&
    !room.finished &&
    (state.phase === GAME_STATE.setup || state.turn === role);
  const goal = state.winLength
    ? t("lineGoal", { width: state.width, height: state.height, n: state.winLength })
    : t(room.game === "dotsboxes" ? "dotsGoal" : "navalGoal");
  return (
    <section class={`room-page ${naval ? "naval-room" : ""}`}>
      <div class="room-topline">
        <Button class="back-link" disabled={busy} onClick={requestLeave}>
          <ActionIcon name="back" />
          {t("backGames")}
        </Button>
        <GameRules game={spec} />
      </div>
      <div class="room-heading">
        <div>
          <h1>{gameTitle(catalog, room.game)}</h1>
          <p class="room-goal">{goal}</p>
        </div>
        <div class="room-invitation">
          <CopyControl value={id} label={t("copyCode")} className="room-code-copy">
            <span class="room-code" translate={false}>
              {id}
            </span>
          </CopyControl>
          <CopyControl value={invitation(id)} label={t("copyLink")}>
            {t("invite")}
          </CopyControl>
        </div>
      </div>
      <p
        class={`game-status ${online && player && role === state.turn && room.started && !room.finished ? "your-turn" : ""} ${room.finished ? "finished" : ""}`}
        role="status"
      >
        <span class="turn-dot" />
        {text}
        {!online && ` · ${t("reconnecting")}`}
      </p>
      {error && <Notice error>{error}</Notice>}
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
          {!room.started && !room.finished && player && (
            <div class="preparation">
              <div class="preparation-heading">
                <h2>{t("roomPreparation")}</h2>
                <p>{t("readyHint")}</p>
              </div>
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
                <ActionIcon name="check" />
                {t(room.seats[role].ready ? "notReady" : "ready")}
              </Button>
            </div>
          )}
          {room.swap && (
            <Notice>
              {t("swapRequest", { from: sideName(spec, room.swap.from), to: sideName(spec, room.swap.to) })}
              {player && (role === room.swap.from || role === room.swap.to) && (
                <span class="actions">
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
                </span>
              )}
            </Notice>
          )}
          <div ref={stage} class="game-stage">
            {naval ? (
              room.started || room.finished ? (
                <NavalGame
                  key={`${id}:${current.round}:${role}`}
                  state={state}
                  actor={role}
                  enabled={canMove}
                  playerNames={room.seats.map((seat) => seat.name)}
                  onAction={(values) => act(ROOM_ACTION.move, values)}
                />
              ) : (
                <div class="prestart-preview">
                  <img src="/battleship.svg" width="160" height="132" alt="" />
                  <p>{t("fleetReadyHint")}</p>
                </div>
              )
            ) : (
              <GameBoard
                game={room.game}
                state={state}
                actor={role}
                enabled={canMove}
                onMove={(move) => void act(ROOM_ACTION.move, move)}
              />
            )}
          </div>
          <div ref={controls} class="game-controls">
            {room.finished && (
              <div class="finished-actions">
                {player && (
                  <a class="button primary" href={`/#/new/${room.game}?from=${id}`}>
                    <ActionIcon name="plus" />
                    {t("newGame")}
                  </a>
                )}
                <a class="button" href={`/#/history?code=${id}`}>
                  <ActionIcon name="clock" />
                  {t("history")}
                </a>
              </div>
            )}
            {!player && !current.name && (
              <a class="button primary" href={invitation(id, { watch: true })}>
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
            {room.seats.map((seat, index) => (
              <div
                class={`seat ${index === role ? "is-you" : ""} ${state.turn === index && room.started && !room.finished ? "is-turn" : ""} player-${index + 1}`}
                key={index}
              >
                <span class="seat-number">
                  {room.game === "tictactoe" ? <Mark player={index + 1} /> : index + 1}
                </span>
                <div>
                  <strong>
                    {seat.name || t("free")}
                    {index === role && <small>{t("you")}</small>}
                  </strong>
                  <span class="seat-presence">
                    {sideName(spec, index)} ·{" "}
                    {t(
                      !seat.taken
                        ? "waitingPlayer"
                        : state.phase === GAME_STATE.setup && room.started
                          ? state.ready?.[index]
                            ? "fleetPrepared"
                            : "placingFleet"
                          : !room.started && seat.ready
                            ? "readyLabel"
                            : seat.online
                              ? "online"
                              : "returnGrace",
                    )}
                  </span>
                </div>
                {state.scores?.length ? <strong class="seat-score">{state.scores[index]}</strong> : null}
              </div>
            ))}
          </div>
          <p class="spectator-count muted">
            <ActionIcon name="eye" />
            {t("spectators", { n: room.spectators })}
          </p>
          <CopyControl
            value={invitation(id, { watch: true })}
            label={t("inviteWatch")}
            className="spectator-invite"
          />
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
    <fieldset class="side-choice">
      <legend>{t("yourSide")}</legend>
      <div class="side-buttons">
        {seats.map((name, index) => (
          <Button
            key={index}
            class={`side-option player-${index + 1}`}
            aria-pressed={role === index}
            disabled={busy || index === role}
            onClick={() => change(index)}
          >
            <span class="side-dot" />
            {sideName(spec, index)}
            {index !== role && name && <small>{name}</small>}
          </Button>
        ))}
      </div>
    </fieldset>
  );
}
