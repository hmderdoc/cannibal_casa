# Cannibal Casa - design

A BBS door (Go, DOOR32.SYS, portable binaries) where the house is a machine
for processing the neighbors. You don't play a hero walking around; you build
and tune a factory and watch it run on a roguelike-style map, in color.

## Decisions (locked)

1. **One clock.** Game time advances at `time_scale` game minutes per real
   minute (default 60), whether or not anyone is connected. Being away is
   just elapsed time the simulation catches up on. There is no separate
   offline rate: what bounds the house while you're gone is the house.
2. **Resource-driven.** Every flow has a real constraint: intake needs a free
   cell and a lure, holding costs feed per head per day, processing needs a
   worker and an idle station, storage has capacity, disposal has throughput,
   appliances draw power. A well-designed house reaches a steady state; a
   badly designed one jams, gets loud, and gets raided while you're out.
3. **Death is inheritance.** A raid or starvation ends this cannibal; a
   relative inherits the house with its fixtures, half the cash, and empty
   cells.
4. **Content line.** Adults only, every one of them annoying. Raunchy humor
   is fine; gore is implied through cooking talk, never described.
5. **Multiplayer plumbing from the start**, fiction still open. Every player
   has a house on a shared street with a feed, casseroles and anonymous tips.
   Local per BBS through the data directory, or across boards through
   `cannibalcasa-server`. What the street *means* is not yet decided.
6. **Name.** "Cannibal Casa" everywhere. Never "Casa de Cannibal".
7. **Color is required.** Semantic palette, 16-color default, 256 on request.

## The machine

- **Sidewalk.** Passers-by walk past every few minutes (more by day, joggers
  at night). At the gate a lure may pull one onto the porch; the welcome mat
  fires when a cell is free. Success drops them down the chute; failure
  sometimes screams (Heat). Drivers leave a car at the curb.
- **Cells.** Livestock eats ($4/day, $2 with the gruel dispenser) and fattens
  toward marbling 5. Hungry livestock screams, stops fattening, tries the
  door, and dies after a day at zero.
- **Workers.** The cannibal and any Renfields pick jobs by priority: eat,
  sleep, feed, plate finished dishes, dispose of bags, butcher (per policy),
  cook (food to a shelf target, then sellable recipes), stock the stand,
  brainwash. Each job is a list of walk-here/work-this-long/then-do steps and
  survives a save mid-step.
- **Stations** cook one recipe at a time for its duration (10 minutes for
  tartare, 14 hours for pulled pork); the dish waits until someone plates it.
- **Evidence.** Butchering leaves bones, clothes and a phone on the back step.
  Disposal machines each take one bag for their duration (chipper 5 minutes
  and loud, compost a day and silent, lye 12 hours). Plots hold a few. The
  stockpot turns bones into broth you can sell.
- **Heat** accrues hourly from loose bags, cars, hungry screaming, running
  smokers; cools hourly, faster with a fence and curtains and when nothing is
  visible. Missing-person reports land one-off after each victim's notice
  period. Crossing a tier sends a visitor to the front door who scores what's
  visible (cars, bags without a fence, screams without soundproofing). 100 is
  the raid.
- **Cash.** The curb stand sells one marked item an hour, nine to six. The
  garden, an Airbnb listing and Etsy soap pay daily. Utilities drain hourly.

## The door

Cursor over the map; Enter looks at whatever is there (a worker's job, a
passer-by's odds, a machine's status with an on/off switch, a build offer on
an empty slot). Space pauses while you're at the controls; 1/2/3 run 1x/2x/4x.
Menus pause the clock; reading panels don't. B builds, P tunes policy, L is
the roster, I the inventory, N the street. On login a digest shows what the
house did while you were gone.

## Architecture

```
cmd/cannibalcasa/          door client (term I/O, input, render, screens)
cmd/cannibalcasa-server/   neighborhood server (token-gated, owns the worlds)
cmd/balancesim/            headless runs under a few build orders
internal/sim/              the simulation: data, map, state, jobs, tick, commands
internal/world/            neighborhood store (players, feed, mail) + local backend
internal/proto/            JSON-lines wire protocol
internal/ui/               cell framebuffer, palette, CP437, widgets
```

The door calls `Sync(speed)` about once a second; the backend advances the
world by the wall-clock elapsed at scale times speed and returns it. Commands
(build, toggle, policy, rename, inherit) are the only way the door changes the
world. Local and remote backends run identical code.
