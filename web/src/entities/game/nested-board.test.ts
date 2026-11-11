import { expect, test } from "bun:test";
import { cellsInRegion, regionForCell } from "@/entities/game/nested-board";

test("nested board coordinate groups do not wrap at region boundaries", () => {
  expect(cellsInRegion(0, 9, 3)).toEqual([0, 1, 2, 9, 10, 11, 18, 19, 20]);
  expect(cellsInRegion(8, 9, 3)).toEqual([60, 61, 62, 69, 70, 71, 78, 79, 80]);
  expect(regionForCell(2, 9, 3)).toBe(0);
  expect(regionForCell(3, 9, 3)).toBe(1);
  expect(regionForCell(48, 9, 3)).toBe(4);
  expect(regionForCell(80, 9, 3)).toBe(8);
});
