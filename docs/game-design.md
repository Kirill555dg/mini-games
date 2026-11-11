# Game selection and rules

Each game should offer a distinct kind of decision, remain finite, and make legal actions and public results clear. A solved theoretical result alone does not decide whether a game is worth playing. None of these observations substitutes for playtesting with people.

## Classic tic-tac-toe

Larger square boards with a longer target line have simple drawing strategies. Their solved values describe perfect play, rather than the expected outcome between friends.

| Board and target | Perfect-play result |
| --- | --- |
| 3x3, three in a row | Draw |
| 4x4, four in a row | Draw |
| 4x4, three in a row | First-player win |
| 5x5 or 6x6, five in a row | Draw |
| 7x7 or 8x8, five in a row | Draw |

Sources: Table 3 in [Games solved: Now and in the future](https://web.mit.edu/6.034/wwwbob/GamesSolved.pdf#page=15), [The advantage of the initiative](https://doi.org/10.1016/S0020-0255(99)00095-X), and [On solving the 7,7,5-game and the 8,8,5-game](https://doi.org/10.1016/j.tcs.2020.02.023).

Classic games use 3x3 only. Earlier board sizes and match-history formats have no compatibility guarantee and are discarded when retired. Shortening a 4x4 game's target to three would introduce a forced first-player win. We instead offer Nine boards as a separate game with an additional tactical constraint. Availability records are independent of disposable match history and retain their rolling month.

## Nine boards

Nine 3x3 boards form a larger 3x3 board. A cell's position inside its local board determines which local board the opponent must play next. Three marks claim a local board, and three claimed boards in a row win the match.

A won or full local board closes permanently. A move directed to a closed board allows free choice among open boards. A drawn local board belongs to neither player. Closing every board without a macro winning line is a draw.

These rules follow the corrections in [Ben Orlin's original explanation](https://mathwithbaddrawings.com/ultimate-tic-tac-toe-original-post/). The [43-move first-player proof](https://arxiv.org/abs/2006.02353) permits continued play in already won boards, so its result does not apply to this implementation. We do not claim a proven balance or a guaranteed absence of draws for the closed-board variant.

## Implementation contract

- The game ID is `ultimate`; classic records retain `tictactoe`.
- The Go engine owns legal destinations, board closures, turn and outcomes. It is independent of transport and room membership.
- `regionSize` owns the local geometry. Cell and region indices are zero-based. `activeBoard` is optional: zero means the first board, and absence means free choice.
- Every move records `regionResult` when it closes a board and `nextBoard` for the following forced destination. History projects these recorded decisions instead of implementing victory rules again in the browser.
- Won boards and drawn boards are distinct from temporarily unavailable open boards. The board shows the forced destination to players and spectators alike.
- At most 81 marks can be placed. There is no search engine, timer or AI load in production. Snapshots and replay pointers are detached copies.

## Four in a row

Keep the standard 7x6 board and four-in-a-row target. Gravity makes support cells, intersecting threats and move order matter, rather than merely enlarging tic-tac-toe. A game takes at most 42 placements. [Allis's original thesis](https://tromp.github.io/c4/connect4_thesis.pdf) proves a first-player win with perfect play on this board. The game is not theoretically equal; switch the starting side between games in a series.

## Sea battle

Keep 10x10 with ten ships and twenty occupied cells, separation including diagonals, and another shot after a hit. These traditional rules also appear in the [NSTU problem statement, problem 1](https://olymp.nstu.ru/contests/downloadproblems/36). The 8x8 fleet is this service's shorter alternative, not a separately proven balanced game.

Hidden placement, area search, locating a damaged ship and deciding when to resume searching give it a different role. Individual sinking must be explicit: only after every segment is hit does the engine reveal that ship's cells and the surrounding water proved empty by separation. Actual misses and inferred water have different marks. Living ships and drafts remain private. Move records carry those reveals, so history never needs access to a hidden fleet or a second copy of the deduction rules.

## Dots and boxes

Keep 4x4 boxes (40 edges), with two to four players. The player can choose any free edge even when a box can be captured; closing one or two boxes awards another turn. This preserves decisions about chains, loops and giving up points to retain control, described in [Allcock's endgame analysis](https://web.ma.utexas.edu/users/allcock/research/dots.pdf).

The 3–4-player option is a social variant: players can help a rival and may wait during another player's capture run. Two-player strategy results do not establish balance for this variant. Ties remain possible; more modes or extra rules are not justified merely to eliminate them.
