import { useEffect, useId, useRef, useState } from "preact/hooks";
import { GameRules } from "@/entities/game/game-rules";
import { api, type GameSpec, type RoomView, unwrap } from "@/shared/api/game-api";
import { live } from "@/shared/api/live-connection";
import { ROOM_ACTION, ROOM_CODE } from "@/shared/api/protocol.generated";
import { t, translated } from "@/shared/i18n/i18n";
import {
  INVITE_INPUT_MAX_LENGTH,
  invitation,
  normalizeCode,
  parseInvitation,
  savedName,
  saveName,
} from "@/shared/invitations/invitations";
import { ActionIcon } from "@/shared/ui/action-icon";
import { Button, Field, Heading, Notice } from "@/shared/ui/ui-controls";
import "@/features/entry/room-entry.css";

const ENTRY_MODE = { play: "play", watch: "watch" } as const;

type Props = {
  catalog: GameSpec[];
  game?: GameSpec;
  code?: string;
  watch?: boolean;
  preferredSeat?: number;
  preset?: { size: number; players: number };
  from?: string;
  embedded?: boolean;
};
export function RoomEntry({
  catalog,
  game,
  code = "",
  watch = false,
  preferredSeat,
  preset,
  from = "",
  embedded = false,
}: Props) {
  const [name, setName] = useState(savedName),
    [input, setInput] = useState(code),
    [watching, setWatching] = useState(watch);
  const [size, setSize] = useState(
      preset?.size !== undefined && (!game?.sizes || game.sizes.includes(preset.size))
        ? preset.size
        : game?.sizes?.[0] || 0,
    ),
    [players, setPlayers] = useState(preset?.players || game?.minPlayers || 2);
  const [seat, setSeat] = useState(preferredSeat),
    [view, setView] = useState<RoomView | null>(null);
  const [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [nextRoom, setNextRoom] = useState("");
  const alive = useRef(true),
    submitController = useRef<AbortController | null>(null),
    uid = useId();
  const creating = Boolean(game),
    normalized = normalizeCode(input);
  useEffect(
    () => () => {
      alive.current = false;
      submitController.current?.abort();
    },
    [],
  );
  useEffect(() => {
    if (!from) return;
    return live<RoomView>(
      from,
      (value) => {
        if (alive.current && value?.room.nextRoom) setNextRoom(value.room.nextRoom);
      },
      () => {},
    );
  }, [from]);
  useEffect(() => {
    setView(null);
    if (creating || !normalized) return;
    let current = true;
    const controller = new AbortController();
    unwrap(api.GET("/game-api/state", { params: { query: { room: normalized } }, signal: controller.signal }))
      .then((value) => {
        if (current) {
          setView(value);
          if (seat !== undefined && (!value.room.seats[seat] || value.room.seats[seat].taken))
            setSeat(undefined);
        }
      })
      .catch(() => {});
    return () => {
      current = false;
      controller.abort();
    };
  }, [creating, normalized]);
  const nameId = `${uid}-name`,
    codeId = `${uid}-code`;
  const title = game ? t("createTitle", { game: translated(game, "title") || "" }) : t("joinHeading");
  return (
    <section class={embedded ? "inline-entry" : "entry-page"} aria-label={title}>
      {!embedded && (
        <a href={from ? `/#/room/${from}` : "/#/"} class="back-link">
          <ActionIcon name="back" />
          {t(from ? "backRoom" : "backGames")}
        </a>
      )}
      {embedded ? (
        <div class="join-heading">
          <h2>{title}</h2>
          <p>{t("joinHint")}</p>
        </div>
      ) : (
        <Heading title={title} hint={game ? translated(game, "description") : undefined} />
      )}
      <form
        class={embedded ? "join-form" : "entry-form"}
        onKeyDown={(event) => {
          if (!creating && event.key === "Enter" && event.target instanceof HTMLInputElement) {
            event.preventDefault();
            const button = event.currentTarget.querySelector<HTMLButtonElement>(
              `button[value="${watching ? ENTRY_MODE.watch : ENTRY_MODE.play}"]`,
            );
            if (button && !button.disabled) event.currentTarget.requestSubmit(button);
          }
        }}
        onSubmit={async (event) => {
          event.preventDefault();
          if (busy) return;
          if (!creating && !normalized) {
            setError(t("codeInvalid"));
            document.getElementById(codeId)?.focus();
            return;
          }
          const joinWatching =
            event.submitter instanceof HTMLButtonElement
              ? event.submitter.value === ENTRY_MODE.watch
              : watching;
          setBusy(true);
          setError("");
          try {
            const controller = new AbortController();
            submitController.current = controller;
            const value = await unwrap(
              api.POST("/game-api/action", {
                signal: controller.signal,
                body: game
                  ? { action: ROOM_ACTION.create, from, name, game: game.id, size, players, seat }
                  : {
                      action: ROOM_ACTION.enter,
                      name,
                      room: normalized,
                      watch: joinWatching,
                      seat: joinWatching ? undefined : seat,
                    },
              }),
            );
            if (!alive.current) return;
            saveName(name);
            location.hash = `/room/${value.room.id}`;
          } catch (failure) {
            if (!alive.current) return;
            setError(failure instanceof Error ? failure.message : t("error503"));
            if (from)
              try {
                const previous = await unwrap(
                  api.GET("/game-api/state", { params: { query: { room: from } } }),
                );
                if (alive.current && previous.room.nextRoom) setNextRoom(previous.room.nextRoom);
              } catch {
                /* Keep original error. */
              }
          } finally {
            if (alive.current) setBusy(false);
          }
        }}
      >
        <div class={embedded ? "join-inputs" : "entry-fields"}>
          {!creating && (
            <Field label={t("code")} id={codeId}>
              <input
                id={codeId}
                name="room-code"
                class="code-input"
                value={input}
                required
                disabled={busy}
                maxLength={INVITE_INPUT_MAX_LENGTH}
                autoComplete="off"
                autoCapitalize="characters"
                spellcheck={false}
                placeholder={t("codePlaceholder")}
                aria-invalid={Boolean(error && !normalized)}
                onInput={(event) => {
                  const value = event.currentTarget.value;
                  setInput(value);
                  setError("");
                  const pasted = parseInvitation(value);
                  setSeat(pasted.seat);
                  setWatching(pasted.watch || false);
                }}
              />
            </Field>
          )}
          <Field label={t("yourName")} id={nameId}>
            <input
              id={nameId}
              name="nickname"
              class="name-input"
              value={name}
              required
              disabled={busy}
              maxLength={ROOM_CODE.maxNameRunes}
              autoComplete="nickname"
              placeholder={t("namePlaceholder")}
              onInput={(event) => setName(event.currentTarget.value)}
              onBlur={() => saveName(name)}
            />
          </Field>
        </div>
        {!creating && view && (
          <p
            class="join-preview"
            title={translated(
              catalog.find((item) => item.id === view.room.game),
              "title",
            )}
          >
            {translated(
              catalog.find((item) => item.id === view.room.game),
              "title",
            )}
          </p>
        )}
        {game?.sizes && (
          <fieldset class="option-field">
            <legend>{t("boardSize")}</legend>
            <div class="choice-group size-choices">
              {game.sizes.map((value) => (
                <label class="choice" key={value}>
                  <input
                    type="radio"
                    name={`${uid}-size`}
                    value={value}
                    checked={size === value}
                    onChange={() => setSize(value)}
                  />
                  <span class="choice-content">
                    {value} × {value}
                  </span>
                </label>
              ))}
            </div>
          </fieldset>
        )}
        {game && game.maxPlayers > game.minPlayers && (
          <fieldset class="option-field">
            <legend>{t("playerCount")}</legend>
            <div class="choice-group">
              {Array.from(
                { length: game.maxPlayers - game.minPlayers + 1 },
                (_, index) => index + game.minPlayers,
              ).map((value) => (
                <label class="choice" key={value}>
                  <input
                    type="radio"
                    name={`${uid}-players`}
                    value={value}
                    checked={players === value}
                    onChange={() => setPlayers(value)}
                  />
                  <span class="choice-content">
                    <ActionIcon name="users" />
                    {value}
                  </span>
                </label>
              ))}
            </div>
          </fieldset>
        )}
        {game && (
          <>
            <p class="field-help">{t(from ? "newGameHint" : "settingsHint")}</p>
            <GameRules game={game} />
          </>
        )}
        {error && <Notice error>{error}</Notice>}
        {nextRoom && (
          <Notice>
            <a class="text-link" href={invitation(nextRoom, { seat: preferredSeat })}>
              {t("continue")} · {nextRoom}
            </a>
          </Notice>
        )}
        <div class="form-actions">
          {creating ? (
            <Button type="submit" class="primary" disabled={busy}>
              {t(busy ? "creating" : "create")}
              <ActionIcon name="plus" />
            </Button>
          ) : (
            [false, true].map((value) => (
              <Button
                key={String(value)}
                type="submit"
                name="entry-mode"
                value={value ? ENTRY_MODE.watch : ENTRY_MODE.play}
                class={watching === value ? "primary" : ""}
                disabled={busy || !normalized}
              >
                <ActionIcon name={value ? "eye" : "users"} />
                {t(busy ? "entering" : value ? "watch" : "play")}
              </Button>
            ))
          )}
          {!embedded && creating && (
            <a class="button" href={from ? `/#/room/${from}` : "/#/"}>
              {t("cancel")}
            </a>
          )}
        </div>
      </form>
    </section>
  );
}
