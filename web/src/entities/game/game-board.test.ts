import { expect, test } from "bun:test";
import { replayState } from "@/entities/game/game-board";
import type { GameState } from "@/shared/api/game-api";
import { GAME_STATE } from "@/shared/api/protocol.generated";

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
