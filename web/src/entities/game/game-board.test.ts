import { expect, test } from "bun:test";
import { replayState } from "@/entities/game/game-board";
import type { GameState } from "@/shared/api/game-api";
import { GAME_STATE, NAVAL_CELL } from "@/shared/api/protocol.generated";

test("naval replay retains the second player's perspective", () => {
  const state: GameState = {
    width: 8,
    height: 8,
    phase: GAME_STATE.finished,
    turn: 0,
    winner: 0,
    own: Array(64).fill(0),
    enemy: Array(64).fill(0),
    moves: [
      { player: 0, cell: 0, hit: true },
      { player: 1, cell: 1, hit: false },
    ],
  };
  const past = replayState("battleship", state, 1, 1);
  expect(past.own?.[0]).toBe(2);
  expect(past.enemy?.[0]).toBe(0);
  expect(state.own?.[0]).toBe(0);
});
test("naval replay applies sunk ships and water only when revealed by that move", () => {
  const state: GameState = {
    width: 10,
    height: 10,
    phase: GAME_STATE.finished,
    turn: 1,
    winner: 0,
    own: Array(100).fill(0),
    enemy: Array(100).fill(0),
    moves: [
      { player: 0, cell: 0, hit: true },
      { player: 0, cell: 1, hit: true, sunk: [0, 1], water: [2, 10, 11, 12] },
      { player: 0, cell: 99, hit: false },
    ],
  };
  const before = replayState("battleship", state, 1);
  expect(before.enemy?.[0]).toBe(NAVAL_CELL.hit);
  expect(before.enemy?.[1]).toBe(NAVAL_CELL.water);
  expect(before.enemy?.[2]).toBe(NAVAL_CELL.water);
  const sunk = replayState("battleship", state, 2);
  expect(sunk.enemy?.[0]).toBe(NAVAL_CELL.sunk);
  expect(sunk.enemy?.[1]).toBe(NAVAL_CELL.sunk);
  expect(sunk.enemy?.[2]).toBe(NAVAL_CELL.clear);
  expect(replayState("battleship", state, 2, 1).own ?? []).toEqual(sunk.enemy ?? []);
  expect(state.enemy?.[0]).toBe(NAVAL_CELL.water);
});
test("box replay uses recorded captures rather than reimplementing the rules", () => {
  const state: GameState = {
    width: 4,
    height: 4,
    phase: GAME_STATE.finished,
    turn: 0,
    winner: 3,
    board: Array(40).fill(0),
    scores: [0, 0, 0, 1],
    boxes: Array(16).fill(0),
    moves: [
      { player: 0, cell: 0 },
      { player: 1, cell: 4 },
      { player: 2, cell: 20 },
      { player: 3, cell: 21, boxes: [0] },
      { player: 3, cell: 1 },
    ],
  };
  const past = replayState("dotsboxes", state, 4);
  expect(past.boxes?.[0]).toBe(4);
  expect(past.scores?.[3]).toBe(1);
});

test("nested replay preserves board zero, recorded closures and free choice", () => {
  const moves = [
    { player: 0, cell: 0, nextBoard: 0 },
    { player: 1, cell: 1, nextBoard: 1 },
    { player: 0, cell: 3, nextBoard: 0 },
    { player: 1, cell: 2, nextBoard: 2 },
    { player: 0, cell: 6, nextBoard: 0 },
    { player: 1, cell: 10, nextBoard: 4 },
    { player: 0, cell: 30, nextBoard: 0 },
    { player: 1, cell: 19, regionResult: 2, nextBoard: 7 },
    { player: 0, cell: 57 },
    { player: 1, cell: 4, nextBoard: 1 },
  ];
  const state: GameState = {
    width: 9,
    height: 9,
    regionSize: 3,
    phase: GAME_STATE.finished,
    turn: 1,
    winner: 0,
    board: Array(81).fill(0),
    regions: [2, 0, 0, 0, 0, 0, 0, 0, 0],
    moves,
  };
  expect(replayState("ultimate", state, 1).activeBoard).toBe(0);
  const closed = replayState("ultimate", state, 8);
  expect(closed.regions?.[0]).toBe(2);
  expect(closed.activeBoard).toBe(7);
  expect(closed.board?.[19]).toBe(2);
  expect(replayState("ultimate", state, 9).activeBoard).toBeUndefined();
  expect(replayState("ultimate", state, 0).regions).toEqual(Array(9).fill(0));
  if (!closed.regions) throw new Error("Missing recorded region closure");
  closed.regions[0] = 1;
  expect(state.regions?.[0]).toBe(2);
});
