import { expect, test } from "bun:test";
import {
  canPlaceShip,
  completeFleet,
  copyFleet,
  type FleetGeometry,
  fleetFingerprint,
  placementCells,
  randomFleet,
  replaceShip,
} from "@/entities/game/fleet-layout";

const classic: FleetGeometry = { width: 10, height: 10, lengths: [4, 3, 3, 2, 2, 2, 1, 1, 1, 1], gap: 1 };

test("ship footprints respect edges and rectangular boards", () => {
  expect(placementCells({ ship: 0, cell: 8, vertical: false }, classic)).toEqual([]);
  expect(placementCells({ ship: 0, cell: 69, vertical: true }, classic)).toEqual([69, 79, 89, 99]);
  expect(placementCells({ ship: 0, cell: 79, vertical: true }, classic)).toEqual([]);
  expect(placementCells({ ship: -1, cell: 0, vertical: false }, classic)).toEqual([]);
  expect(placementCells({ ship: 0, cell: 0.5, vertical: false }, classic)).toEqual([]);
  const rectangle = { width: 8, height: 6, lengths: [3], gap: 1 };
  expect(placementCells({ ship: 0, cell: 31, vertical: true }, rectangle)).toEqual([31, 39, 47]);
  expect(placementCells({ ship: 0, cell: 39, vertical: true }, rectangle)).toEqual([]);
});

test("preview enforces server fleet gap, including diagonal contact", () => {
  const placed = [{ ship: 0, cell: 0, vertical: false }];
  expect(canPlaceShip({ ship: 1, cell: 14, vertical: true }, placed, classic)).toBe(false);
  expect(canPlaceShip({ ship: 1, cell: 15, vertical: true }, placed, classic)).toBe(true);
  expect(canPlaceShip({ ship: 1, cell: 10, vertical: false }, placed, { ...classic, gap: 0 })).toBe(true);
  expect(canPlaceShip({ ship: 1, cell: 0, vertical: false }, placed, { ...classic, gap: 0 })).toBe(false);
});

test("moving an existing ship ignores its previous footprint without mutating saved state", () => {
  const original = [{ ship: 0, cell: 0, vertical: false }];
  const candidate = { ship: 0, cell: 1, vertical: false };
  expect(canPlaceShip(candidate, original, classic)).toBe(true);
  const next = replaceShip(original, candidate);
  next[0].cell = 2;
  expect(original[0].cell).toBe(0);
  const copy = copyFleet(original);
  copy[0].vertical = true;
  expect(original[0].vertical).toBe(false);
});

test("ready requires every unique ship and a valid full layout", () => {
  const small = { width: 4, height: 4, lengths: [2, 1], gap: 1 };
  const fleet = [
    { ship: 0, cell: 0, vertical: false },
    { ship: 1, cell: 15, vertical: false },
  ];
  expect(completeFleet(fleet, small)).toBe(true);
  expect(completeFleet(fleet.slice(0, 1), small)).toBe(false);
  expect(completeFleet([fleet[0], fleet[0]], small)).toBe(false);
  expect(completeFleet([fleet[0], { ...fleet[1], cell: 5 }], small)).toBe(false);
  expect(fleetFingerprint(fleet)).toBe(fleetFingerprint([...fleet].reverse()));
});

test("random setup places a complete classic fleet with authoritative dimensions and spacing", () => {
  let seed = 17;
  const random = () => {
    seed = (seed * 16807) % 2147483647;
    return (seed - 1) / 2147483646;
  };
  for (let iteration = 0; iteration < 12; iteration++) {
    const fleet = randomFleet(classic, random);
    expect(fleet).not.toBeNull();
    expect(completeFleet(fleet || [], classic)).toBe(true);
  }
  expect(randomFleet({ width: 1, height: 1, lengths: [2], gap: 1 }, random)).toBeNull();
});
