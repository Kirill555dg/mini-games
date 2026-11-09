import { describe, expect, test } from "bun:test";
import { normalizeCode, parseInvitation } from "@/shared/invitations/invitations";

describe("room invitations", () => {
  test("normalizes human-entered codes", () => {
    expect(normalizeCode(" ab1-2cd ")).toBe("AB12CD");
    expect(normalizeCode("AB12C")).toBe("");
    expect(normalizeCode("AB12C!")).toBe("");
  });
  test("pasted spectator links preserve their mode and never reserve a seat", () => {
    expect(parseInvitation("https://example.test/#/join/ab12cd?mode=watch&seat=1")).toEqual({
      code: "AB12CD",
      watch: true,
      seat: undefined,
    });
  });
  test("side hints are parsed without turning them into authorization", () => {
    expect(parseInvitation("https://example.test/#/join/AB12CD?seat=0")).toEqual({
      code: "AB12CD",
      watch: false,
      seat: 0,
    });
    expect(parseInvitation("https://example.test/#/join/AB12CD?seat=-1").seat).toBeUndefined();
    expect(parseInvitation("https://example.test/not-an-invite").code).toBe("");
  });
});
