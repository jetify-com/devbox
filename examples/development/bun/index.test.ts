import { expect, test } from "bun:test";
import figlet from "figlet";

test("2 + 2", () => {
  expect(2 + 2).toBe(4);
});

test("figlet renders a banner", () => {
  const banner = figlet.textSync("Bun!");
  expect(banner.split("\n").length).toBeGreaterThan(1);
});
