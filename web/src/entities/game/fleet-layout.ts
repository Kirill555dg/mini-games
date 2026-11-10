import type { GameState } from "@/shared/api/game-api";

export type FleetPlacement = NonNullable<GameState["placements"]>[number];
export type FleetGeometry = {
  width: number;
  height: number;
  lengths: readonly number[];
  gap: number;
};
export const FLEET_INTERACTION = { dragThreshold: 6, undoLimit: 64 } as const;
const RANDOM_SEARCH_BUDGET = 20_000;

export function copyFleet(placements: readonly FleetPlacement[]): FleetPlacement[] {
  return placements.map((placement) => ({ ...placement }));
}

export function fleetFingerprint(placements: readonly FleetPlacement[]): string {
  return [...placements]
    .sort((a, b) => a.ship - b.ship)
    .map(({ ship, cell, vertical }) => `${ship}:${cell}:${Number(vertical)}`)
    .join(";");
}

export function placementCells(placement: FleetPlacement, geometry: FleetGeometry): number[] {
  const { width, height, lengths } = geometry;
  const length = lengths[placement.ship];
  if (
    !Number.isInteger(placement.ship) ||
    !Number.isInteger(placement.cell) ||
    !length ||
    placement.cell < 0 ||
    placement.cell >= width * height
  )
    return [];
  const row = Math.floor(placement.cell / width);
  const column = placement.cell % width;
  if (placement.vertical ? row + length > height : column + length > width) return [];
  return Array.from({ length }, (_, index) => placement.cell + index * (placement.vertical ? width : 1));
}

export function canPlaceShip(
  candidate: FleetPlacement,
  placements: readonly FleetPlacement[],
  geometry: FleetGeometry,
): boolean {
  const candidateCells = placementCells(candidate, geometry);
  if (!candidateCells.length) return false;
  const others = placements.filter((placement) => placement.ship !== candidate.ship);
  for (const other of others) {
    for (const occupied of placementCells(other, geometry)) {
      for (const cell of candidateCells) {
        if (
          Math.abs(Math.floor(cell / geometry.width) - Math.floor(occupied / geometry.width)) <=
            geometry.gap &&
          Math.abs((cell % geometry.width) - (occupied % geometry.width)) <= geometry.gap
        )
          return false;
      }
    }
  }
  return true;
}

export function replaceShip(
  placements: readonly FleetPlacement[],
  candidate: FleetPlacement,
): FleetPlacement[] {
  return [...placements.filter((placement) => placement.ship !== candidate.ship), { ...candidate }].sort(
    (a, b) => a.ship - b.ship,
  );
}

export function completeFleet(placements: readonly FleetPlacement[], geometry: FleetGeometry): boolean {
  return (
    placements.length === geometry.lengths.length &&
    new Set(placements.map((placement) => placement.ship)).size === geometry.lengths.length &&
    placements.every((placement) => canPlaceShip(placement, placements, geometry))
  );
}

export function fleetCellName(cell: number, width: number): string {
  return `${String.fromCharCode("A".charCodeAt(0) + (cell % width))}${Math.floor(cell / width) + 1}`;
}

export function randomFleet(
  geometry: FleetGeometry,
  random: () => number = Math.random,
): FleetPlacement[] | null {
  const ships = geometry.lengths
    .map((length, ship) => ({ length, ship }))
    .sort((a, b) => b.length - a.length);
  let remaining = RANDOM_SEARCH_BUDGET;
  const candidates = ships.map(({ ship, length }) => {
    const options: FleetPlacement[] = [];
    for (let cell = 0; cell < geometry.width * geometry.height; cell++) {
      for (const vertical of length === 1 ? [false] : [false, true]) {
        const candidate = { ship, cell, vertical };
        if (placementCells(candidate, geometry).length) options.push(candidate);
      }
    }
    for (let index = options.length - 1; index > 0; index--) {
      const swap = Math.floor(random() * (index + 1));
      [options[index], options[swap]] = [options[swap], options[index]];
    }
    return options;
  });
  function search(index: number, placed: FleetPlacement[]): FleetPlacement[] | null {
    if (index === ships.length) return placed;
    for (const candidate of candidates[index]) {
      if (--remaining < 0) return null;
      if (!canPlaceShip(candidate, placed, geometry)) continue;
      const result = search(index + 1, [...placed, candidate]);
      if (result) return result;
    }
    return null;
  }
  return search(0, [])?.sort((a, b) => a.ship - b.ship) || null;
}
