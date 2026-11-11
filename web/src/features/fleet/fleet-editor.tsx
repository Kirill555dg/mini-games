import type { CSSProperties, TargetedKeyboardEvent, TargetedMouseEvent, TargetedPointerEvent } from "preact";
import { useEffect, useRef, useState } from "preact/hooks";
import {
  canPlaceShip,
  completeFleet,
  copyFleet,
  FLEET_INTERACTION,
  type FleetGeometry,
  type FleetPlacement,
  fleetCellName,
  fleetFingerprint,
  placementCells,
  randomFleet,
  replaceShip,
} from "@/entities/game/fleet-layout";
import type { Command, GameState } from "@/shared/api/game-api";
import { t } from "@/shared/i18n/i18n";
import { ActionIcon } from "@/shared/ui/action-icon";
import { Button } from "@/shared/ui/ui-controls";
import "@/features/fleet/fleet-editor.css";

type Props = {
  state: GameState;
  enabled: boolean;
  onAction: (values: Partial<Command>) => Promise<boolean>;
};
type Ghost = { placement: FleetPlacement; column: number; row: number; valid: boolean };
type Drag = {
  ship: number;
  id: number;
  startX: number;
  startY: number;
  lastX: number;
  lastY: number;
  offsetColumn: number;
  offsetRow: number;
  vertical: boolean;
  started: boolean;
};
type SaveState = "idle" | "saving" | "saved" | "error";

export function FleetEditor({ state, enabled, onAction }: Props) {
  const geometry: FleetGeometry = {
    width: state.width,
    height: state.height,
    lengths: state.fleetLengths || [],
    gap: state.fleetGap ?? 0,
  };
  const [draft, setDraft] = useState(() => copyFleet(state.placements || []));
  const [selected, setSelected] = useState<number | null>(() => {
    const next = geometry.lengths.findIndex((_, ship) => !draft.some((item) => item.ship === ship));
    return next >= 0 ? next : geometry.lengths.length ? 0 : null;
  });
  const [vertical, setVertical] = useState(
    () => draft.find((item) => item.ship === selected)?.vertical ?? false,
  );
  const [ghost, setGhost] = useState<Ghost | null>(null);
  const [dragging, setDragging] = useState(false);
  const [saveState, setSaveState] = useState<SaveState>("idle");
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState("");
  const [undoCount, setUndoCount] = useState(0);
  const [focusedCell, setFocusedCell] = useState(0);
  const board = useRef<HTMLDivElement>(null);
  const drag = useRef<Drag | null>(null);
  const ghostRef = useRef<Ghost | null>(null);
  const draftRef = useRef(draft);
  const history = useRef<FleetPlacement[][]>([]);
  const pending = useRef(false);
  const retryDeploy = useRef(false);
  const dirty = useRef(false);
  const alive = useRef(true);
  const suppressClick = useRef(false);
  const serverFingerprint = fleetFingerprint(state.placements || []);
  const lastServer = useRef(serverFingerprint);
  const editable = enabled && !saving;

  useEffect(() => {
    alive.current = true;
    const warn = (event: BeforeUnloadEvent) => {
      if (dirty.current) event.preventDefault();
    };
    window.addEventListener("beforeunload", warn);
    return () => {
      alive.current = false;
      window.removeEventListener("beforeunload", warn);
    };
  }, []);

  useEffect(() => {
    if (serverFingerprint === lastServer.current) return;
    lastServer.current = serverFingerprint;
    if (serverFingerprint === fleetFingerprint(draftRef.current)) {
      dirty.current = false;
      setSaveState("saved");
    } else if (!dirty.current && !pending.current && !drag.current) {
      const next = copyFleet(state.placements || []);
      draftRef.current = next;
      setDraft(next);
    }
  }, [serverFingerprint, state.placements]);

  useEffect(() => {
    if (enabled || !drag.current) return;
    drag.current = null;
    ghostRef.current = null;
    setGhost(null);
    setDragging(false);
  }, [enabled]);

  function showGhost(next: Ghost | null) {
    ghostRef.current = next;
    setGhost(next);
  }
  function orientationFor(ship: number) {
    return draftRef.current.find((item) => item.ship === ship)?.vertical ?? vertical;
  }
  function cancelDrag() {
    drag.current = null;
    setDragging(false);
    showGhost(null);
  }
  function selectShip(ship: number, keyboard = false) {
    setSelected(ship);
    const current = draftRef.current.find((item) => item.ship === ship);
    if (current) setVertical(current.vertical);
    setMessage("");
    showGhost(null);
    if (keyboard) {
      const cell = current?.cell ?? focusedCell;
      setFocusedCell(cell);
      board.current?.querySelector<HTMLButtonElement>(`[data-fleet-cell="${cell}"]`)?.focus();
    }
  }
  async function persist(next: FleetPlacement[], deploy = false) {
    if (pending.current || !enabled) return;
    pending.current = true;
    setSaving(true);
    retryDeploy.current = deploy;
    dirty.current = true;
    setSaveState("saving");
    setMessage("");
    const fingerprint = fleetFingerprint(next);
    let success = false;
    try {
      success = await onAction({ kind: deploy ? "deploy" : "fleet", placements: copyFleet(next) });
    } catch {
      // A transport failure must keep the editable draft and offer retry.
    } finally {
      pending.current = false;
      if (alive.current) setSaving(false);
      if (alive.current && fingerprint === fleetFingerprint(draftRef.current)) {
        const saved = success || (!deploy && lastServer.current === fingerprint);
        dirty.current = !saved;
        setSaveState(saved ? "saved" : "error");
      }
    }
  }
  function changeFleet(next: FleetPlacement[], remember = true) {
    if (!editable || pending.current) return;
    if (fleetFingerprint(next) === fleetFingerprint(draftRef.current)) return;
    if (remember) {
      history.current = [...history.current, copyFleet(draftRef.current)].slice(-FLEET_INTERACTION.undoLimit);
      setUndoCount(history.current.length);
    }
    draftRef.current = next;
    setDraft(next);
    showGhost(null);
    void persist(next);
  }
  function place(candidate: FleetPlacement) {
    if (!editable) return;
    if (!canPlaceShip(candidate, draftRef.current, geometry)) {
      setMessage(t("fleetInvalid"));
      return;
    }
    changeFleet(replaceShip(draftRef.current, candidate));
    setSelected(candidate.ship);
    setVertical(candidate.vertical);
  }
  function rotate() {
    if (!editable || selected === null) return;
    if (drag.current) {
      drag.current.vertical = !drag.current.vertical;
      drag.current.offsetColumn = 0;
      drag.current.offsetRow = 0;
      setVertical(drag.current.vertical);
      previewPoint(drag.current.lastX, drag.current.lastY, drag.current);
      return;
    }
    const current = draftRef.current.find((item) => item.ship === selected);
    const nextVertical = !(current?.vertical ?? vertical);
    if (current) {
      const candidate = { ...current, vertical: nextVertical };
      if (!canPlaceShip(candidate, draftRef.current, geometry)) {
        setMessage(t("fleetInvalid"));
        return;
      }
      changeFleet(replaceShip(draftRef.current, candidate));
    }
    setVertical(nextVertical);
    if (ghost) {
      const placement = { ...ghost.placement, vertical: nextVertical };
      showGhost({ ...ghost, placement, valid: canPlaceShip(placement, draftRef.current, geometry) });
    }
  }
  function remove() {
    if (selected === null || !editable) return;
    changeFleet(draftRef.current.filter((item) => item.ship !== selected));
  }
  function undo() {
    if (!editable || pending.current) return;
    const previous = history.current.pop();
    if (!previous) return;
    setUndoCount(history.current.length);
    changeFleet(previous, false);
  }
  function shuffle() {
    if (!editable) return;
    const next = randomFleet(geometry);
    if (next) changeFleet(next);
    else setMessage(t("fleetInvalid"));
  }
  function previewPoint(clientX: number, clientY: number, active?: Drag) {
    if (!board.current || (selected === null && !active)) return;
    const ship = active?.ship ?? selected;
    if (ship === null) return;
    const rect = board.current.getBoundingClientRect();
    const column =
      Math.floor(((clientX - rect.left) / rect.width) * geometry.width) - (active?.offsetColumn || 0);
    const row = Math.floor(((clientY - rect.top) / rect.height) * geometry.height) - (active?.offsetRow || 0);
    const placement = {
      ship,
      cell: row * geometry.width + column,
      vertical: active?.vertical ?? orientationFor(ship),
    };
    const inside = column >= 0 && row >= 0 && column < geometry.width && row < geometry.height;
    showGhost({
      placement,
      column,
      row,
      valid: inside && canPlaceShip(placement, draftRef.current, geometry),
    });
  }
  function startDrag(event: TargetedPointerEvent<HTMLButtonElement>, ship: number, placed: boolean) {
    if (!editable || event.button !== 0) return;
    suppressClick.current = false;
    selectShip(ship);
    const current = draftRef.current.find((item) => item.ship === ship);
    const orientation = current?.vertical ?? vertical;
    const rect = event.currentTarget.getBoundingClientRect();
    const length = geometry.lengths[ship];
    drag.current = {
      ship,
      id: event.pointerId,
      startX: event.clientX,
      startY: event.clientY,
      lastX: event.clientX,
      lastY: event.clientY,
      offsetColumn:
        placed && current && !orientation
          ? Math.min(length - 1, Math.floor(((event.clientX - rect.left) / rect.width) * length))
          : 0,
      offsetRow:
        placed && current && orientation
          ? Math.min(length - 1, Math.floor(((event.clientY - rect.top) / rect.height) * length))
          : 0,
      vertical: orientation,
      started: false,
    };
    event.currentTarget.setPointerCapture(event.pointerId);
  }
  function moveDrag(event: TargetedPointerEvent<HTMLButtonElement>) {
    const active = drag.current;
    if (!active || active.id !== event.pointerId) return;
    active.lastX = event.clientX;
    active.lastY = event.clientY;
    if (!active.started) {
      if (
        Math.hypot(event.clientX - active.startX, event.clientY - active.startY) <
        FLEET_INTERACTION.dragThreshold
      )
        return;
      active.started = true;
      setDragging(true);
    }
    event.preventDefault();
    previewPoint(event.clientX, event.clientY, active);
  }
  function endDrag(event: TargetedPointerEvent<HTMLButtonElement>) {
    const active = drag.current;
    if (!active || active.id !== event.pointerId) return;
    if (active.started) {
      suppressClick.current = true;
      const target = ghostRef.current;
      if (target?.valid) place(target.placement);
      else setMessage(t("fleetInvalid"));
    }
    cancelDrag();
    if (event.currentTarget.hasPointerCapture(event.pointerId))
      event.currentTarget.releasePointerCapture(event.pointerId);
  }
  function shipClick(event: TargetedMouseEvent<HTMLButtonElement>, ship: number) {
    if (suppressClick.current && event.detail !== 0) {
      suppressClick.current = false;
      return;
    }
    selectShip(ship, event.detail === 0);
  }
  function vesselStyle(placement: FleetPlacement, column?: number, row?: number): CSSProperties {
    return {
      "--column": column ?? placement.cell % geometry.width,
      "--row": row ?? Math.floor(placement.cell / geometry.width),
      "--span-x": placement.vertical ? 1 : geometry.lengths[placement.ship],
      "--span-y": placement.vertical ? geometry.lengths[placement.ship] : 1,
    };
  }
  function moveFocus(event: TargetedKeyboardEvent<HTMLButtonElement>, cell: number) {
    const column = cell % geometry.width;
    const targets: Record<string, number> = {
      ArrowLeft: column ? cell - 1 : cell,
      ArrowRight: column < geometry.width - 1 ? cell + 1 : cell,
      ArrowUp: Math.max(column, cell - geometry.width),
      ArrowDown: Math.min((geometry.height - 1) * geometry.width + column, cell + geometry.width),
      Home: cell - column,
      End: cell - column + geometry.width - 1,
    };
    const next = targets[event.key];
    if (next === undefined) return;
    event.preventDefault();
    setFocusedCell(next);
    board.current?.querySelector<HTMLButtonElement>(`[data-fleet-cell="${next}"]`)?.focus();
  }

  const occupied = new Map(
    draft.flatMap((item) => placementCells(item, geometry).map((cell) => [cell, item.ship] as const)),
  );
  const ready = completeFleet(draft, geometry);
  const selectedPlaced = draft.find((item) => item.ship === selected);
  const saveKey =
    saveState === "error"
      ? retryDeploy.current
        ? "fleetDeployFailed"
        : "fleetUnsaved"
      : saveState === "saving"
        ? "fleetSaving"
        : saveState === "saved"
          ? "fleetSaved"
          : "fleetReadyHint";
  return (
    <section
      class={`fleet-editor ${dragging ? "is-dragging" : ""}`}
      aria-label={t("fleetBoard")}
      onKeyDown={(event) => {
        if (event.key.toLowerCase() === "r" && !event.metaKey && !event.ctrlKey && !event.altKey) {
          event.preventDefault();
          rotate();
        } else if (event.key === "Escape") cancelDrag();
        else if (event.key === "Delete" || event.key === "Backspace") {
          event.preventDefault();
          remove();
        }
      }}
    >
      <div class="fleet-chart" style={{ "--columns": geometry.width, "--rows": geometry.height }}>
        <div class="fleet-columns" aria-hidden="true">
          {Array.from({ length: geometry.width }, (_, column) => (
            <span key={column}>{String.fromCharCode("A".charCodeAt(0) + column)}</span>
          ))}
        </div>
        <div class="fleet-rows" aria-hidden="true">
          {Array.from({ length: geometry.height }, (_, row) => (
            <span key={row}>{row + 1}</span>
          ))}
        </div>
        <div
          class="fleet-grid"
          ref={board}
          onPointerMove={(event) => {
            if (!drag.current && editable && event.pointerType !== "touch")
              previewPoint(event.clientX, event.clientY);
          }}
          onPointerLeave={() => {
            if (!drag.current) showGhost(null);
          }}
        >
          <div class="fleet-cells">
            {Array.from({ length: geometry.width * geometry.height }, (_, cell) => (
              <button
                type="button"
                class="fleet-cell"
                key={cell}
                data-fleet-cell={cell}
                tabIndex={cell === focusedCell ? 0 : -1}
                aria-label={`${fleetCellName(cell, geometry.width)}: ${t(occupied.has(cell) ? "ship" : "emptyCell")}`}
                aria-disabled={!editable}
                onFocus={() => setFocusedCell(cell)}
                onKeyDown={(event) => moveFocus(event, cell)}
                onClick={() => {
                  if (selected !== null) place({ ship: selected, cell, vertical: orientationFor(selected) });
                }}
              />
            ))}
          </div>
          {draft.map((item) => (
            <button
              type="button"
              key={item.ship}
              class={`fleet-vessel ${item.vertical ? "is-vertical" : ""} ${selected === item.ship ? "is-selected" : ""} ${dragging && drag.current?.ship === item.ship ? "is-lifted" : ""}`}
              style={vesselStyle(item)}
              data-fleet-ship={item.ship}
              aria-label={`${t("fleetShipLabel", { n: item.ship + 1, length: geometry.lengths[item.ship] })}, ${fleetCellName(item.cell, geometry.width)}`}
              aria-pressed={selected === item.ship}
              disabled={!editable}
              onPointerDown={(event) => startDrag(event, item.ship, true)}
              onPointerMove={moveDrag}
              onPointerUp={endDrag}
              onPointerCancel={cancelDrag}
              onLostPointerCapture={() => {
                if (drag.current) cancelDrag();
              }}
              onClick={(event) => shipClick(event, item.ship)}
              onContextMenu={(event) => event.preventDefault()}
            >
              {Array.from({ length: geometry.lengths[item.ship] }, (_, cell) => (
                <span class="fleet-porthole" key={cell} />
              ))}
            </button>
          ))}
          {ghost && (
            <span
              class={`fleet-footprint ${ghost.valid ? "is-valid" : "is-invalid"}`}
              style={vesselStyle(ghost.placement, ghost.column, ghost.row)}
              aria-hidden="true"
            />
          )}
        </div>
      </div>
      <div class="fleet-dock">
        <div class="fleet-dock-heading">
          <strong>{t("fleetDock")}</strong>
          <span>{t("fleetProgress", { n: draft.length, total: geometry.lengths.length })}</span>
        </div>
        <fieldset class="fleet-tokens" aria-label={t("chooseShip")}>
          {geometry.lengths.map((length, ship) => {
            const placed = draft.some((item) => item.ship === ship);
            return (
              <button
                type="button"
                key={ship}
                class={`fleet-token ${selected === ship ? "is-selected" : ""} ${placed ? "is-placed" : ""}`}
                aria-label={t("fleetShipLabel", { n: ship + 1, length })}
                aria-pressed={selected === ship}
                disabled={!editable}
                data-dock-ship={ship}
                onPointerDown={(event) => startDrag(event, ship, false)}
                onPointerMove={moveDrag}
                onPointerUp={endDrag}
                onPointerCancel={cancelDrag}
                onLostPointerCapture={() => {
                  if (drag.current) cancelDrag();
                }}
                onClick={(event) => shipClick(event, ship)}
                onContextMenu={(event) => event.preventDefault()}
              >
                <span class="fleet-miniature" style={{ "--length": length }} aria-hidden="true">
                  {Array.from({ length }, (_, cell) => (
                    <span key={cell} />
                  ))}
                </span>
                {placed ? (
                  <ActionIcon name="check" />
                ) : (
                  <span class="fleet-token-length" aria-hidden="true">
                    {length}
                  </span>
                )}
              </button>
            );
          })}
        </fieldset>
        <p class="fleet-instruction">{t("fleetDragHint")}</p>
        <div class="fleet-edit-actions">
          <Button
            class="small"
            disabled={!editable || selected === null}
            onClick={rotate}
            title={t("fleetRotateHint")}
          >
            <ActionIcon name="rotate" />
            {t("rotateShip")}
          </Button>
          <Button class="small" disabled={!editable || !selectedPlaced} onClick={remove}>
            <ActionIcon name="trash" />
            {t("removeShip")}
          </Button>
          <Button class="small" disabled={!editable || !undoCount} onClick={undo}>
            <ActionIcon name="back" />
            {t("undo")}
          </Button>
        </div>
        <div class="fleet-finish-actions">
          <Button class="small" disabled={!editable} onClick={shuffle}>
            <ActionIcon name="shuffle" />
            {t("shuffle")}
          </Button>
          <Button class="small" disabled={!editable || !draft.length} onClick={() => changeFleet([])}>
            {t("clear")}
          </Button>
          <Button
            class="primary"
            disabled={!editable || !ready}
            onClick={() => void persist(draftRef.current, true)}
          >
            <ActionIcon name="check" />
            {t("deployFleet")}
          </Button>
        </div>
        <p
          class={`fleet-save-state ${saveState === "error" || message ? "is-error" : ""}`}
          role="status"
          aria-live="polite"
        >
          {message || t(saveKey)}
          {saveState === "error" && (
            <button
              type="button"
              class="text-link"
              disabled={!editable}
              onClick={() => void persist(draftRef.current, retryDeploy.current)}
            >
              {t("retry")}
            </button>
          )}
        </p>
        {ghost && (
          <p class={`fleet-placement-hint ${ghost.valid ? "is-valid" : "is-invalid"}`}>
            {t(ghost.valid ? "fleetValid" : "fleetInvalid")}
          </p>
        )}
      </div>
    </section>
  );
}
