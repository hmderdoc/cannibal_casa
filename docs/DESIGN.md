# Cannibal Casa - design

A BBS door (Go, DOOR32.SYS, portable binaries) where you play a cannibal
living in the suburbs. Feed yourself, expand the house, hide the evidence,
keep the HOA off your porch. Roguelike-style text graphics, always in color.

## Decisions (locked)

1. **Pacing.** A caller may advance a few in-game days per real day (config
   `days_per_day`, default 3); unspent days bank up to `day_bank_max` so a
   caller who skips a weekend can catch up. Within a day, actions are free.
2. **Death is inheritance.** When a cannibal is caught (Heat hits 100) or
   starves, a relative inherits the house. Fixtures and recipes persist, cash
   is halved, livestock and evidence are lost, Heat resets low. The dynasty's
   past cannibals live on in the hall of fame.
3. **Content line.** Adults only, every one of them annoying. Raunchy and
   risque humor is fine; gore is implied through puns and cooking talk, never
   described. Never children, never sexual violence.
4. **Multiplayer from the start.** Every player lives on the same cul-de-sac.
   A neighborhood server shares one street across boards; without a server the
   door runs a local neighborhood shared by every node on the BBS. Same code
   path either way (a `Backend` interface).
5. **Name.** "Cannibal Casa" everywhere. Never "Casa de Cannibal".
6. **Color is required.** Every cell carries fg/bg. A semantic palette
   (`meat_fresh`, `heat_high`, `cash`, ...) resolves to 16-color by default
   and 256-color on request. Sysops can retint via `[colors]` later.

## Core loop

Each in-game day has four phases: Morning, Afternoon, Evening, Night. Day
phases are for cooking, building, selling, disposal. Night is for hunting.
Ending Night rolls the day: hunger drops, meat decays, livestock eat, Renfield
loyalty drifts, evidence radiates Heat, the neighborhood rolls events.

Meters:

| Meter    | Range | Drives                                                   |
|----------|-------|----------------------------------------------------------|
| Hunger   | 0-100 | 0 = starving. Meals restore; fresh cuts restore more.    |
| Heat     | 0-100 | Suspicion. Antagonist ladder by tier. 100 = the raid.    |
| Cash     | $     | Upgrades. Earned by selling dishes, candles, tomatoes.   |
| Stars    | count | Score. Earned by serving quality dishes.                 |

## The house

Four levels, each a fixed floor plan with fixture slots:

- **Ground.** Kitchen (stove, fridge, freezer, grinder, smoker, sous vide,
  dehydrator), Living room (dinner parties), the Closet (brainwashing chair),
  Garage (van, chest freezer, wood chipper), Porch (lures).
- **Basement.** Cells = livestock capacity. Dig to expand. Soundproofing.
- **Backyard.** Burial plots, compost, garden beds, shed, pool, tall fence.
- **Attic.** Surveillance (mission safety), drying racks (jerky).

## Victims

Archetypes with marbling (meal quality), fight (capture difficulty),
notice (days until someone misses them) and heat-when-missing. Solar salesman,
HOA inspector, MLM hun, influencer, crossfit guy, missionaries (pairs), tech
bro, vegan, jogger, delivery driver, true-crime podcaster (late game).

## Recipes

Gated by fixtures and cut freshness. Liver with fava beans and a nice Chianti,
long-pig pulled pork, meat pies (the sellable), Farmer Vincent fritters,
Soylent smoothie, jerky, bone broth (food AND evidence disposal), tallow
candles (Etsy money), charcuterie board (prestige).

## Evidence

Meals leave bones, clothes, phones, and cars. Each undisposed item radiates
Heat daily. Disposal: bury (dog risk), chipper (noise), lye bath (floor risk),
bone broth (elegant), compost (tomatoes!). Phones ping until microwaved.

## Missions

Leave the house for a few phases: farmers market, city hunt, Tinder date,
burglary, vigilante job. While away, escape and dig rolls run against
Renfield loyalty and security fixtures.

## Antagonist ladder (Heat tiers)

Nosy neighbor -> HOA -> local news -> true-crime podcast -> detective -> FBI.
Crossing a tier schedules an inspection: one phase of warning, then a visible
evidence check.

## Neighborhood (multiplayer)

- Shared **Nextdoor feed** of every player's notable events.
- **Neighbor list** with cannibal name, generation, stars, days, heat tier.
- **Casserole**: send a dish to a neighbor (feeds them, costs you a dish).
- **Anonymous tip**: raise a neighbor's Heat; may backfire onto you.

## Architecture

```
cmd/cannibalcasa/          door client (term I/O, input, render, screens)
cmd/cannibalcasa-server/   neighborhood server (authoritative, token-gated)
cmd/balancesim/            headless economy sim for tuning
internal/game/             pure rules: state, data tables, actions, day tick
internal/world/            neighborhood store (players, feed, mail) + local backend
internal/proto/            JSON-lines wire protocol
internal/ui/               cell framebuffer, palette, CP437, widgets
```

The door never runs rules on its own in remote mode: it sends Actions, the
server applies `game.Apply` and returns state. Local mode calls the same
function in-process against a file store in the data directory.
