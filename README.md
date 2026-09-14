# Cannibal Casa

A BBS door where you run a cannibal's house in the suburbs as a machine:
lure the neighbors in, keep them in cells, butcher, cook, sell, and dispose of
what's left before the street notices. A continuous simulation on a
roguelike-style map, in full color, that keeps running when you log off.

Runs on any board that serves a `DOOR32.SYS` dropfile (Synchronet, Mystic,
ENiGMA½, Talisman, WWIV, ...) on Linux, Windows, macOS or FreeBSD. One static
binary, no runtime dependencies.

## The game

The house is a machine. Passers-by walk the sidewalk; a lure pulls one onto
the porch and the welcome mat drops them into a free cell. Livestock eats and
fattens. You and your Renfields butcher, carry cuts to the fridge, cook at the
stove, smoker or sous vide, plate to the shelf, eat, sleep, stock the curb
stand, and haul the bags out back to the chipper, the compost, the lye tub or
a flower bed. Heat reads the jams: bags on the step, cars at the curb, hungry
screaming from the basement. Cross a tier and someone comes to the door.

It runs on one clock, at `time_scale` game minutes per real minute, whether
you're watching or not. Log off for a day and the house runs two game months
on your policy. Come back to full freezers or to sirens.

You don't walk around. You move a cursor, look at things, build machines on
any tile (and move them: workers walk, so layout is throughput), switch them
on and off, and set policy: butcher at what marbling, keep how many meals on
the shelf, which recipes go to the stand, whether The Chair turns guests into
workers. Then you watch it run, in color, on a roguelike map.

Money comes from the wallets of the people who drop through the mat, from
the curb stand, and from going out: a shift at the mall, a market run,
hunting, scavenging, or raiding another house on the street, CPU or player.
Renfields take roles (hand, cook, guard, shill, gimp), and one that eats
enough of what the house serves becomes a fellow cannibal who hunts on their
own.

Death is inheritance: a raid or starvation ends this cannibal, and a relative
takes the house with its fixtures, half the cash, and empty cells.

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

The server owns the worlds and runs the simulation; doors sync the clock and
send commands. If the server is unreachable the door falls back to the local street
and says so.

## Keys

Arrows or WASD move the cursor; `Enter` looks at whatever is under it (a
worker's job, a passer-by's odds, a machine's status, switch and Move, or
the build list on empty floor). `Tab` changes floor. `Space` pauses while
you're at the controls; `1` `2` `3` run at 1x, 2x, 4x. `B` build then place
with the cursor, `G` go out, `P` policy, `L` roster and roles, `I`
inventory, `N` the street, `O` options, `?` the guide, `Q` quit.

## Development

```sh
go test ./...
go run ./cmd/balancesim -runs 30 -days 60    # headless runs under a few build orders
go run ./cmd/balancesim -runs 1 -days 10 -trace factory
go run ./cmd/balancesim -runs 4 -deaths      # what killed each house
./cannibalcasa -snapshot -size 80x24 -snapshot-days 5   # one frame to stdout, UTF-8
./cannibalcasa -local -utf8                   # play in a local terminal
```

Layout: `internal/sim` is the simulation (no I/O), `internal/world` the
neighborhood store and local backend, `internal/proto` the JSON-lines wire
format, `internal/ui` the color framebuffer and CP437 encoder, and the two
`cmd/` binaries. See `docs/DESIGN.md` for the design decisions.
