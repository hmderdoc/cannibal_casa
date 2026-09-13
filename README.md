# Cannibal Casa

A BBS door where you play a cannibal in the suburbs. Stay fed, stay unnoticed,
upgrade the house. Roguelike-style text graphics in full color, multiplayer
from day one: every player lives on the same cul-de-sac.

Runs on any board that serves a `DOOR32.SYS` dropfile (Synchronet, Mystic,
ENiGMA½, Talisman, WWIV, ...) on Linux, Windows, macOS or FreeBSD. One static
binary, no runtime dependencies.

## The game

- **Hunt** at night. The street rolls three passers-by: a solar salesman, a
  tech bro, a pair of missionaries. Each archetype has marbling (meal quality),
  fight (capture odds), and how long until someone notices they're gone.
- **Livestock** waits in basement cells, fattening if fed. Butcher for cuts;
  cuts spoil in days unless frozen.
- **Cook** at the stove, smoker, sous vide, grinder, dehydrator, grill. Recipes
  unlock with the gear: liver with fava beans and a nice Chianti, Mrs.
  Lovett's meat pies, Long Pig Pulled Pork, bone broth, Paper Street soap.
- **Evidence**: every meal leaves bones, clothes and a phone that keeps
  pinging. Bury it, compost it, chip it, dissolve it, or simmer it into broth.
- **Heat** climbs with loose evidence, noisy livestock and missing-person
  reports, and cools with fences, curtains and dinner parties where the
  neighbors eat the neighbors. Cross a tier and someone comes to the door:
  the nosy neighbor, the HOA, Channel 6, a true-crime podcast, the FBI.
- **Renfields**: The Chair brainwashes a captive into a helper who watches
  the house while you're on a mission (the market, a Tinder date, a hunting
  trip downtown, a Dexter-style clean-up job).
- **Death is inheritance.** Get raided or starve and a relative inherits the
  house: fixtures and recipes stay, cash halves, the freezer comes empty.
- **The street**: a shared Nextdoor feed, a neighbor list, casseroles you can
  send, and anonymous tips to the HOA about the house across the road.

Pacing: a caller may advance a few in-game days per real day (default 3),
banked up to 9 for catch-up. Within a day, actions are free.

## Install (Synchronet)

```sh
cd /sbbs/xtrn/cannibal_casa
go build ./cmd/cannibalcasa            # or drop a release binary here
jsexec install-xtrn ../xtrn/cannibal_casa
```

The installer registers the door (DOOR32, native, multiuser) and seeds
`cannibalcasa.ini` from `cannibalcasa.example.ini`. Other BBS packages: run
`cannibalcasa -dropfile <path to DOOR32.SYS>` as a native door.

Data lives in `$SBBSDATA/cannibalcasa` (per-player saves, the feed, mail), or
`./data` next to the binary when the BBS doesn't export `SBBSDATA`.

## Multiplayer

Without a server, every node on the BBS shares one neighborhood through the
data directory. To share a street across boards, run the server somewhere:

```sh
go build ./cmd/cannibalcasa-server
./cannibalcasa-server -addr :7710 -token CHANGE_ME -data /sbbs/data/cannibalcasa
```

(`cannibalcasa-server.service` is a systemd unit to adapt.) Then point each
door at it in `cannibalcasa.ini`:

```ini
[server]
server  = your.host
port    = 7710
token   = CHANGE_ME
bbsname = my.bbs.example
```

The server is authoritative: doors send actions, the server applies the
rules. If the server is unreachable the door falls back to the local street
and says so.

## Keys

Arrows or WASD move; walking onto a fixture uses it, walking onto an empty
slot offers to build there. `Tab` changes floor. `T` passes the time. `B`
build, `C` cook, `E` pantry, `F` fridge, `L` livestock, `X` evidence, `M`
missions, `H` hunt, `P` dinner party, `N` the street, `O` options (256-color
mode, rename), `?` help, `Q` quit.

## Development

```sh
go test ./...
go run ./cmd/balancesim -runs 200 -days 60   # headless economy tuning
go run ./cmd/balancesim -runs 1 -trace sweeney
./cannibalcasa -snapshot -size 80x24          # one frame to stdout, UTF-8
./cannibalcasa -local -utf8                   # play in a local terminal
```

Layout: `internal/game` is the pure rules (no I/O), `internal/world` the
neighborhood store and local backend, `internal/proto` the JSON-lines wire
format, `internal/ui` the color framebuffer and CP437 encoder, and the two
`cmd/` binaries. See `docs/DESIGN.md` for the design decisions.
