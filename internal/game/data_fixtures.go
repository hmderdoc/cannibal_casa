package game

// Fixture is an upgrade you can own. It occupies a slot on one level of the
// house (see house.go for the floor plans and where each slot sits).
type Fixture struct {
	ID      string
	Name    string
	Level   Level
	Slot    string // slot letter on that level's floor plan
	Cost    int
	Prereq  string // another fixture id, or ""
	Glyph   rune
	Color   string
	Starter bool // owned from day one
	Cell    bool // counts as a livestock cell
	Plot    int  // burial capacity it adds
	Desc    string
}

var Fixtures = []Fixture{
	// --- Ground: kitchen ---------------------------------------------------
	{ID: "stove", Name: "Stove", Level: Ground, Slot: "A", Cost: 0, Glyph: '#', Color: "fix_kitchen", Starter: true,
		Desc: "Four burners and an oven. Everything starts here."},
	{ID: "fridge", Name: "Fridge", Level: Ground, Slot: "B", Cost: 0, Glyph: '%', Color: "fix_cold", Starter: true,
		Desc: "Holds 6 cuts. Fresh for a few days, then it isn't."},
	{ID: "fridge_big", Name: "Sub-Zero Fridge", Level: Ground, Slot: "C", Cost: 150, Prereq: "fridge", Glyph: '%', Color: "fix_cold",
		Desc: "+6 cuts of cold storage. Stainless. The realtor said it adds value."},
	{ID: "freezer", Name: "Freezer", Level: Ground, Slot: "D", Cost: 120, Prereq: "fridge", Glyph: '*', Color: "fix_cold",
		Desc: "8 frozen cuts. Frozen meat never spoils but loses a star in the pan."},
	{ID: "grinder", Name: "Meat Grinder", Level: Ground, Slot: "E", Cost: 80, Prereq: "stove", Glyph: 'g', Color: "fix_kitchen",
		Desc: "Turns scraps into pie filling. Unlocks Mrs. Lovett's pies."},
	{ID: "smoker", Name: "Offset Smoker", Level: Ground, Slot: "F", Cost: 200, Prereq: "stove", Glyph: 'S', Color: "fix_kitchen",
		Desc: "Low and slow. Unlocks Long Pig Pulled Pork; the whole street will smell it."},
	{ID: "sous_vide", Name: "Sous Vide Circulator", Level: Ground, Slot: "G", Cost: 180, Prereq: "stove", Glyph: '~', Color: "fix_kitchen",
		Desc: "Edge to edge pink. Unlocks the five-star loin."},
	{ID: "dehydrator", Name: "Dehydrator", Level: Ground, Slot: "H", Cost: 90, Prereq: "stove", Glyph: 'd', Color: "fix_kitchen",
		Desc: "Jerky keeps for a month. Unlocks Trail Mix (heavy on the trail)."},
	{ID: "blender", Name: "Blender", Level: Ground, Slot: "I", Cost: 40, Glyph: 'b', Color: "fix_kitchen",
		Desc: "Unlocks the Soylent smoothie. It's people. It's always people."},
	{ID: "stockpot", Name: "Stockpot", Level: Ground, Slot: "J", Cost: 60, Prereq: "stove", Glyph: 'u', Color: "fix_kitchen",
		Desc: "Bone broth: destroys bones AND feeds you. Wellness moms pay for the jars."},
	{ID: "microwave", Name: "Microwave", Level: Ground, Slot: "K", Cost: 50, Glyph: 'w', Color: "fix_kitchen",
		Desc: "Thirty seconds on high ends a phone's location history."},
	{ID: "knife_set", Name: "Japanese Knife Set", Level: Ground, Slot: "L", Cost: 70, Glyph: '!', Color: "fix_kitchen",
		Desc: "Cleaner butchery: one extra prime cut per victim."},
	{ID: "wine_rack", Name: "Wine Rack", Level: Ground, Slot: "N", Cost: 60, Glyph: 'Y', Color: "fix_living",
		Desc: "A nice Chianti. Unlocks the liver with fava beans. Dinner guests linger."},
	// --- Ground: living room ---------------------------------------------
	{ID: "dining_set", Name: "Dining Set", Level: Ground, Slot: "O", Cost: 100, Glyph: 'T', Color: "fix_living",
		Desc: "Host dinner parties. Nothing lowers Heat like feeding the neighbors their neighbors."},
	{ID: "curtains", Name: "Blackout Curtains", Level: Ground, Slot: "P", Cost: 60, Glyph: '|', Color: "fix_living",
		Desc: "Inspections see less. The nosy neighbor sees nothing."},
	{ID: "big_tv", Name: "85-inch TV", Level: Ground, Slot: "Q", Cost: 80, Glyph: 'V', Color: "fix_living",
		Desc: "Renfields stay loyal when there's something on."},
	// --- Ground: the closet ----------------------------------------------
	{ID: "chair", Name: "The Chair", Level: Ground, Slot: "R", Cost: 250, Glyph: 'h', Color: "fix_closet",
		Desc: "Eyelid clamps, Beethoven, a slideshow. Brainwash livestock into Renfields."},
	{ID: "vhs", Name: "Motivational VHS Library", Level: Ground, Slot: "U", Cost: 120, Prereq: "chair", Glyph: '=', Color: "fix_closet",
		Desc: "Faster brainwashing. Tapes 1 through 12 of 'You Belong Here'."},
	// --- Ground: garage ----------------------------------------------------
	{ID: "van", Name: "Windowless Van", Level: Ground, Slot: "W", Cost: 300, Glyph: 'v', Color: "fix_garage",
		Desc: "'FREE WIFI' painted on the side. Hunts succeed more; victims' cars get chopped for free."},
	{ID: "chest_freezer", Name: "Chest Freezer", Level: Ground, Slot: "X", Cost: 220, Prereq: "freezer", Glyph: '*', Color: "fix_cold",
		Desc: "+16 frozen cuts. Every suburban garage has one. Yours is honest about it."},
	{ID: "chipper", Name: "Wood Chipper", Level: Ground, Slot: "Y", Cost: 350, Glyph: '@', Color: "fix_garage",
		Desc: "Disposes of anything instantly. Loud. Very Fargo."},
	{ID: "chemistry", Name: "Chemistry Set", Level: Ground, Slot: "Z", Cost: 90, Glyph: 'q', Color: "fix_garage",
		Desc: "Chloroform, mostly. Hunts succeed more. Say your own name."},
	{ID: "lye_tub", Name: "Plastic Tub of Lye", Level: Ground, Slot: "a", Cost: 150, Glyph: 'l', Color: "fix_garage",
		Desc: "Dissolves evidence. Plastic, not ceramic. You've seen the show."},
	// --- Ground: porch lures -----------------------------------------------
	{ID: "lure_couch", Name: "FREE COUCH", Level: Ground, Slot: "1", Cost: 30, Glyph: 'c', Color: "fix_lure",
		Desc: "Curb-side. Somebody always stops to look."},
	{ID: "lure_wifi", Name: "Free Wi-Fi Sign", Level: Ground, Slot: "2", Cost: 60, Glyph: 'w', Color: "fix_lure",
		Desc: "Password: password. Attracts tech bros and huns."},
	{ID: "lure_garage_sale", Name: "Garage Sale Sign", Level: Ground, Slot: "3", Cost: 50, Glyph: '$', Color: "fix_lure",
		Desc: "Everything must go. Especially the customers."},
	{ID: "lure_openhouse", Name: "OPEN HOUSE Sign", Level: Ground, Slot: "4", Cost: 80, Glyph: 'O', Color: "fix_lure",
		Desc: "Balloons on the mailbox. Realtors, salesmen and Karens roll right in."},
	{ID: "lure_airbnb", Name: "Airbnb Listing", Level: Ground, Slot: "5", Cost: 200, Glyph: 'A', Color: "fix_lure",
		Desc: "Guests check in. Five stars, would eat again. Influencers love the pool photos."},
	{ID: "doorbell_cam", Name: "Doorbell Camera", Level: Ground, Slot: "6", Cost: 90, Glyph: 'o', Color: "fix_lure",
		Desc: "One extra phase of warning before an inspection knocks."},
	// --- Basement ------------------------------------------------------------
	{ID: "cell_1", Name: "Cell 1", Level: Basement, Slot: "A", Cost: 0, Glyph: 'n', Color: "fix_cell", Starter: true, Cell: true,
		Desc: "A cot, a bucket, a motivational poster."},
	{ID: "cell_2", Name: "Cell 2", Level: Basement, Slot: "B", Cost: 100, Prereq: "cell_1", Glyph: 'n', Color: "fix_cell", Cell: true,
		Desc: "Dug over a weekend. The neighbors assumed a wine cellar."},
	{ID: "cell_3", Name: "Cell 3", Level: Basement, Slot: "C", Cost: 200, Prereq: "cell_2", Glyph: 'n', Color: "fix_cell", Cell: true,
		Desc: "Now it's a wing."},
	{ID: "cell_4", Name: "Cell 4", Level: Basement, Slot: "D", Cost: 350, Prereq: "cell_3", Glyph: 'n', Color: "fix_cell", Cell: true,
		Desc: "Structural engineer said no. Structural engineer is in cell 4."},
	{ID: "cell_5", Name: "Cell 5", Level: Basement, Slot: "E", Cost: 500, Prereq: "cell_4", Glyph: 'n', Color: "fix_cell", Cell: true,
		Desc: "The HOA would call it an unpermitted ADU."},
	{ID: "soundproof", Name: "Soundproofing", Level: Basement, Slot: "F", Cost: 180, Prereq: "cell_1", Glyph: '#', Color: "fix_cell",
		Desc: "Egg-crate foam floor to ceiling. Livestock stop radiating Heat and escape less."},
	{ID: "buffet", Name: "All-You-Can-Eat Trough", Level: Basement, Slot: "G", Cost: 150, Prereq: "cell_1", Glyph: 'U', Color: "fix_cell",
		Desc: "Livestock fatten faster. Marbling is a lifestyle."},
	{ID: "gruel", Name: "Gruel Dispenser", Level: Basement, Slot: "H", Cost: 120, Prereq: "cell_1", Glyph: 'G', Color: "fix_cell",
		Desc: "Feeding livestock costs half. Tastes like a hospital."},
	// --- Backyard --------------------------------------------------------------
	{ID: "plot_1", Name: "Flower Bed", Level: Backyard, Slot: "A", Cost: 0, Glyph: '"', Color: "fix_yard", Starter: true, Plot: 2,
		Desc: "Petunias. Room for two underneath."},
	{ID: "plot_2", Name: "Rose Garden", Level: Backyard, Slot: "B", Cost: 60, Prereq: "plot_1", Glyph: '"', Color: "fix_yard", Plot: 3,
		Desc: "The roses have never looked better. Three more underneath."},
	{ID: "plot_3", Name: "Koi Pond", Level: Backyard, Slot: "C", Cost: 90, Prereq: "plot_2", Glyph: 'o', Color: "fix_cold", Plot: 3,
		Desc: "The koi are enormous. Three more underneath."},
	{ID: "compost", Name: "Compost Bin", Level: Backyard, Slot: "D", Cost: 80, Glyph: 'C', Color: "fix_yard",
		Desc: "Bones go in, black gold comes out. Needed for the garden."},
	{ID: "garden", Name: "Heirloom Tomato Garden", Level: Backyard, Slot: "E", Cost: 120, Prereq: "compost", Glyph: 't', Color: "fix_yard",
		Desc: "Suspiciously fertile. Sells at the market every day."},
	{ID: "fence", Name: "Eight-Foot Privacy Fence", Level: Backyard, Slot: "F", Cost: 200, Glyph: '|', Color: "fix_yard",
		Desc: "The neighbor's dog stops digging. Heat cools faster."},
	{ID: "grill", Name: "Kamado Grill", Level: Backyard, Slot: "G", Cost: 110, Glyph: 'Q', Color: "fix_kitchen",
		Desc: "Ceramic. Unlocks Cul-de-Sac Ribs, the dinner party favorite."},
	{ID: "shed", Name: "Tool Shed", Level: Backyard, Slot: "H", Cost: 140, Glyph: 'D', Color: "fix_yard",
		Desc: "Shovels, tarps, zip ties. Burying is quieter; escapes are rarer."},
	{ID: "pool", Name: "Pool", Level: Backyard, Slot: "I", Cost: 400, Prereq: "fence", Glyph: '~', Color: "fix_cold",
		Desc: "Pool party! Influencers arrive uninvited. Also a very large lye tub if you think about it."},
	{ID: "gnome", Name: "Garden Gnome", Level: Backyard, Slot: "J", Cost: 25, Glyph: 'g', Color: "fix_yard",
		Desc: "He's seen things. Slightly lucky."},
	// --- Attic ---------------------------------------------------------------------
	{ID: "cameras", Name: "Surveillance Rig", Level: Attic, Slot: "A", Cost: 250, Glyph: 'o', Color: "fix_attic",
		Desc: "Watch the house from your phone. Missions are much safer."},
	{ID: "drying_rack", Name: "Drying Racks", Level: Attic, Slot: "B", Cost: 100, Glyph: '=', Color: "fix_attic",
		Desc: "Air-cured charcuterie. Needed for the Delicatessen board."},
	{ID: "telescope", Name: "Telescope", Level: Attic, Slot: "C", Cost: 90, Glyph: 'T', Color: "fix_attic",
		Desc: "For astronomy. Spot joggers before they spot you; night hunts succeed more."},
	{ID: "cot", Name: "Renfield Cot", Level: Attic, Slot: "D", Cost: 80, Glyph: 'n', Color: "fix_attic",
		Desc: "A second Renfield needs somewhere to sleep."},
	{ID: "soap_molds", Name: "Soap Molds", Level: Attic, Slot: "E", Cost: 70, Prereq: "stockpot", Glyph: 'm', Color: "fix_attic",
		Desc: "Paper Street Soap Co. Turns scraps into money on Etsy."},
}

var fixtureByID = func() map[string]*Fixture {
	m := map[string]*Fixture{}
	for i := range Fixtures {
		m[Fixtures[i].ID] = &Fixtures[i]
	}
	return m
}()

// FixtureByID returns nil for an unknown id.
func FixtureByID(id string) *Fixture { return fixtureByID[id] }

// RenfieldCap is how many Renfields the house can keep before they wander.
func (s *State) RenfieldCap() int {
	n := 1
	if s.Has("cot") {
		n++
	}
	return n
}
