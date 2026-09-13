// Package sim is Cannibal Casa's continuous simulation: one clock in game
// minutes, entities that walk the house, machines with throughput, containers
// with capacity, and Heat as the pressure gauge that reads jams. No terminal
// or network code lives here; the door renders what the sim says and the
// balance simulator runs it headless. The same formula runs whether anyone is
// watching: an absence is just elapsed wall-clock time to catch up on.
package sim

// Victim is an archetype of suburban adult who will not be missed (much).
type Victim struct {
	ID          string
	Name        string
	Glyph       rune
	Color       string
	Marbling    int      // 1..5 at capture
	Fight       int      // 0..5, resists the trap
	NoticeDays  int      // days until reported missing
	HeatMissing float64  // heat when reported (negative: the world is relieved)
	Weight      int      // sidewalk traffic weight
	Lures       []string // fixtures that pull this archetype to the porch
	Pair        bool     // arrives in twos
	Night       bool     // only out after dark
	Drives      bool     // leaves a car at the curb
	Flavor      string
}

var Victims = []Victim{
	{ID: "solar", Name: "Door-to-Door Solar Salesman", Glyph: 's', Color: "victim_easy", Marbling: 3, Fight: 1, NoticeDays: 14, HeatMissing: 5, Weight: 12,
		Lures: []string{"lure_openhouse"}, Drives: true,
		Flavor: "Nobody has ever missed one. His company thinks he quit. His mother thinks he's in Tucson."},
	{ID: "mlm", Name: "MLM Hun", Glyph: 'm', Color: "victim_easy", Marbling: 3, Fight: 1, NoticeDays: 10, HeatMissing: 6, Weight: 10,
		Lures:  []string{"lure_garage_sale", "lure_wifi"},
		Flavor: "Hey hun!! Quick question, would you like to join my downline? No. Would you like to join my dinner?"},
	{ID: "techbro", Name: "Tech Bro", Glyph: 't', Color: "victim_prime", Marbling: 5, Fight: 2, NoticeDays: 5, HeatMissing: 15, Weight: 8,
		Lures: []string{"lure_wifi", "lure_airbnb"}, Drives: true,
		Flavor: "Keto, intermittent fasting, grass-fed everything. Beautifully marbled. Insufferable, even now."},
	{ID: "vegan", Name: "Militant Vegan", Glyph: 'v', Color: "victim_prime", Marbling: 4, Fight: 1, NoticeDays: 6, HeatMissing: 10, Weight: 8,
		Lures:  []string{"lure_garage_sale", "garden"},
		Flavor: "Grass-fed, free-range, no antibiotics. Premium. The irony is the seasoning."},
	{ID: "crossfit", Name: "CrossFit Guy", Glyph: 'c', Color: "victim_hard", Marbling: 1, Fight: 5, NoticeDays: 3, HeatMissing: 12, Weight: 7,
		Lures:  []string{"lure_couch"},
		Flavor: "Lean. Fights back. Will tell you about his WOD from inside the cell."},
	{ID: "jogger", Name: "3 A.M. Jogger", Glyph: 'j', Color: "victim_hard", Marbling: 2, Fight: 3, NoticeDays: 4, HeatMissing: 10, Weight: 9, Night: true,
		Lures:  []string{"telescope"},
		Flavor: "Reflective vest, headphones in, zero situational awareness."},
	{ID: "delivery", Name: "Package Delivery Driver", Glyph: 'd', Color: "victim_mid", Marbling: 3, Fight: 2, NoticeDays: 3, HeatMissing: 14, Weight: 8,
		Lures: []string{"lure_airbnb", "lure_openhouse"}, Drives: true,
		Flavor: "Fast, tired, on a schedule. The packages will keep coming. The packages will pile up."},
	{ID: "missionary", Name: "Missionary", Glyph: 'M', Color: "victim_mid", Marbling: 3, Fight: 1, NoticeDays: 9, HeatMissing: 12, Weight: 6, Pair: true,
		Lures:  []string{"lure_couch"},
		Flavor: "They travel in pairs. Two for one. They knocked. You answered. Everyone's day changed."},
	{ID: "influencer", Name: "Lifestyle Influencer", Glyph: 'i', Color: "victim_mid", Marbling: 3, Fight: 1, NoticeDays: 7, HeatMissing: 22, Weight: 6,
		Lures: []string{"lure_airbnb", "pool"}, Drives: true,
		Flavor: "Her followers assumed a digital detox. On day seven they assumed the worst and were correct."},
	{ID: "karen", Name: "Wants To Speak To The Manager", Glyph: 'K', Color: "victim_hard", Marbling: 3, Fight: 3, NoticeDays: 5, HeatMissing: 12, Weight: 7,
		Lures: []string{"lure_garage_sale", "lure_openhouse"}, Drives: true,
		Flavor: "You are the manager. You are also the butcher. She had notes on both."},
	{ID: "landlord", Name: "Landlord", Glyph: 'L', Color: "victim_prime", Marbling: 4, Fight: 1, NoticeDays: 20, HeatMissing: 3, Weight: 5,
		Lures: []string{"lure_airbnb"}, Drives: true,
		Flavor: "His tenants threw a party. Nobody has checked on him and nobody will."},
	{ID: "crypto", Name: "Crypto Evangelist", Glyph: '$', Color: "victim_prime", Marbling: 4, Fight: 2, NoticeDays: 8, HeatMissing: 8, Weight: 6,
		Lures:  []string{"lure_wifi"},
		Flavor: "All that steakhouse networking. Marbled like a portfolio, and the portfolio was also fake."},
	{ID: "hoa", Name: "HOA Inspector", Glyph: 'H', Color: "victim_heat", Marbling: 2, Fight: 2, NoticeDays: 2, HeatMissing: 20, Weight: 4, Drives: true,
		Flavor: "Cited your mailbox. Cited your hedge. Did not cite the basement, because he never got that far."},
	{ID: "podcaster", Name: "True Crime Podcaster", Glyph: 'P', Color: "victim_heat", Marbling: 2, Fight: 2, NoticeDays: 4, HeatMissing: -20, Weight: 3,
		Flavor: "Came to the street chasing a story. The final episode was mysteriously never recorded. Downloads went way up."},
}

var victimByID = func() map[string]*Victim {
	m := map[string]*Victim{}
	for i := range Victims {
		m[Victims[i].ID] = &Victims[i]
	}
	return m
}()

func VictimByID(id string) *Victim {
	if v, ok := victimByID[id]; ok {
		return v
	}
	return &Victim{ID: id, Name: "Stranger", Glyph: '?', Color: "victim_mid", Marbling: 2, Fight: 2, NoticeDays: 7, HeatMissing: 10}
}

var captiveNames = []string{
	"Brayden", "Kayleigh", "Chad", "Tiffani", "Hunter", "Mackenzie", "Tanner", "Brittani",
	"Blake", "Skyler", "Chip", "Bethany", "Dakota", "Jaxon", "Paisley", "Colton",
	"Madison", "Trent", "Krystal", "Kyle", "Amber", "Preston", "Destiny", "Wyatt",
}

// Fixture kinds. A fixture is a slot on the map; its Kind says what the sim
// does with it.
const (
	KindPassive   = "passive"   // a bonus, no process (fence, curtains, knife set)
	KindStation   = "station"   // runs recipes (stove, smoker, ...)
	KindDisposal  = "disposal"  // destroys evidence over time
	KindPlot      = "plot"      // burial capacity
	KindContainer = "container" // holds items (fridge, pantry shelf, back step)
	KindLure      = "lure"      // pulls passers-by onto the porch
	KindTrap      = "trap"      // fires on a lured passer-by
	KindCell      = "cell"      // holds one livestock
	KindBed       = "bed"       // workers sleep here
	KindStand     = "stand"     // sells dishes at the curb
	KindChair     = "chair"     // brainwashing
	KindBlock     = "block"     // butcher station
	KindVan       = "van"       // hauls cars
)

// Fixture is something you can own. Level/Slot say where it sits; Kind and the
// numbers say what it does. Draw is per-hour utility cost while owned.
type Fixture struct {
	ID       string
	Name     string
	Level    Level
	Slot     byte
	Cost     int
	Prereq   string
	Glyph    rune
	Color    string
	Kind     string
	Starter  bool
	Duration int      // minutes per job (disposal/chair/block)
	Capacity int      // container/plot/stand capacity
	Noise    float64  // heat per use (disposal) or per hour (passive noise)
	Draw     float64  // utilities, $/hour
	Accepts  []string // disposal: evidence kinds it takes
	Desc     string
}

var Fixtures = []Fixture{
	// --- porch: trap, lures, stand -----------------------------------------
	{ID: "welcome_mat", Name: "Welcome Mat", Level: Ground, Slot: '0', Cost: 0, Glyph: '_', Color: "fix_lure", Kind: KindTrap, Starter: true,
		Desc: "Says WELCOME. Underneath: a trapdoor and a laundry chute to the basement."},
	{ID: "trapdoor_pro", Name: "Hydraulic Trapdoor", Level: Ground, Slot: '0', Cost: 320, Prereq: "welcome_mat", Glyph: '_', Color: "fix_lure", Kind: KindTrap,
		Desc: "Faster, quieter, wider. Even the CrossFit guy goes down. Replaces the mat in the same spot."},
	{ID: "lure_couch", Name: "FREE COUCH", Level: Ground, Slot: '1', Cost: 30, Glyph: 'c', Color: "fix_lure", Kind: KindLure,
		Desc: "Curb-side. Somebody always stops to look."},
	{ID: "lure_wifi", Name: "Free Wi-Fi Sign", Level: Ground, Slot: '2', Cost: 60, Glyph: 'w', Color: "fix_lure", Kind: KindLure,
		Desc: "Password: password. Tech bros and huns."},
	{ID: "lure_garage_sale", Name: "Garage Sale Sign", Level: Ground, Slot: '3', Cost: 50, Glyph: '$', Color: "fix_lure", Kind: KindLure,
		Desc: "Everything must go. Especially the customers."},
	{ID: "lure_openhouse", Name: "OPEN HOUSE Sign", Level: Ground, Slot: '4', Cost: 80, Glyph: 'O', Color: "fix_lure", Kind: KindLure,
		Desc: "Balloons on the mailbox. Salesmen and Karens roll right in."},
	{ID: "lure_airbnb", Name: "Airbnb Listing", Level: Ground, Slot: '5', Cost: 200, Glyph: 'A', Color: "fix_lure", Kind: KindLure,
		Desc: "Guests check in. Five stars, would eat again. Pays $15 a day too."},
	{ID: "doorbell_cam", Name: "Doorbell Camera", Level: Ground, Slot: '6', Cost: 90, Glyph: 'o', Color: "fix_lure", Kind: KindPassive, Draw: 0.02,
		Desc: "Inspectors take longer to work up the nerve. Passers-by look suspicious in 4K."},
	{ID: "stand", Name: "Curb Stand", Level: Ground, Slot: '7', Cost: 0, Glyph: 'S', Color: "fix_lure", Kind: KindStand, Starter: true, Capacity: 4,
		Desc: "A folding table. Pies, fritters, jerky, broth, soap. Sells one item an hour, nine to six. Workers stock it."},
	{ID: "stand_big", Name: "Farm Stand", Level: Ground, Slot: '7', Cost: 120, Prereq: "stand", Glyph: 'S', Color: "fix_lure", Kind: KindStand, Capacity: 10,
		Desc: "Awning, chalkboard, a jar for tips. Holds ten items. Replaces the folding table."},
	// --- kitchen: stations, cold storage, passives -------------------------
	{ID: "stove", Name: "Stove", Level: Ground, Slot: 'A', Cost: 0, Glyph: '#', Color: "fix_kitchen", Kind: KindStation, Starter: true, Draw: 0.04,
		Desc: "Four burners and an oven. Everything starts here."},
	{ID: "fridge", Name: "Fridge", Level: Ground, Slot: 'B', Cost: 0, Glyph: '%', Color: "fix_cold", Kind: KindContainer, Starter: true, Capacity: 8, Draw: 0.06,
		Desc: "Holds 8 cuts. Fresh for a few days, then it isn't."},
	{ID: "fridge_big", Name: "Sub-Zero Fridge", Level: Ground, Slot: 'C', Cost: 150, Prereq: "fridge", Glyph: '%', Color: "fix_cold", Kind: KindContainer, Capacity: 8, Draw: 0.08,
		Desc: "+8 cuts of cold storage. Stainless. The realtor said it adds value."},
	{ID: "freezer", Name: "Freezer", Level: Ground, Slot: 'D', Cost: 120, Prereq: "fridge", Glyph: '*', Color: "fix_cold", Kind: KindContainer, Capacity: 10, Draw: 0.08,
		Desc: "10 frozen cuts. Overflow from the fridge freezes here; frozen meat never spoils but cooks a star lower."},
	{ID: "grinder", Name: "Meat Grinder", Level: Ground, Slot: 'E', Cost: 80, Glyph: 'g', Color: "fix_kitchen", Kind: KindPassive,
		Desc: "Turns scraps into pie filling. Unlocks Mrs. Lovett's pies."},
	{ID: "smoker", Name: "Offset Smoker", Level: Ground, Slot: 'F', Cost: 200, Glyph: 'S', Color: "fix_kitchen", Kind: KindStation, Noise: 0.1,
		Desc: "Fourteen hours, low and slow. Unlocks Long Pig Pulled Pork; the whole street smells it."},
	{ID: "sous_vide", Name: "Sous Vide Circulator", Level: Ground, Slot: 'G', Cost: 180, Glyph: '~', Color: "fix_kitchen", Kind: KindStation, Draw: 0.04,
		Desc: "Edge to edge pink. Unlocks the five-star loin."},
	{ID: "dehydrator", Name: "Dehydrator", Level: Ground, Slot: 'H', Cost: 90, Glyph: 'd', Color: "fix_kitchen", Kind: KindStation, Draw: 0.04,
		Desc: "Eight hours to jerky that keeps a month."},
	{ID: "blender", Name: "Blender", Level: Ground, Slot: 'I', Cost: 40, Glyph: 'b', Color: "fix_kitchen", Kind: KindStation,
		Desc: "Unlocks the Soylent smoothie. It's people. It's always people."},
	{ID: "stockpot", Name: "Stockpot", Level: Ground, Slot: 'J', Cost: 60, Glyph: 'u', Color: "fix_kitchen", Kind: KindStation,
		Desc: "Bone broth: bones go in, evidence comes out as collagen. Also soap, with molds."},
	{ID: "microwave", Name: "Microwave", Level: Ground, Slot: 'K', Cost: 50, Glyph: 'w', Color: "fix_kitchen", Kind: KindDisposal, Duration: 2, Accepts: []string{"phone"},
		Desc: "Thirty seconds on high ends a phone's location history."},
	{ID: "knife_set", Name: "Japanese Knife Set", Level: Ground, Slot: 'L', Cost: 70, Glyph: '!', Color: "fix_kitchen", Kind: KindPassive,
		Desc: "Cleaner butchery: one extra prime cut per body, and the block works faster."},
	{ID: "pantry", Name: "Pantry Shelf", Level: Ground, Slot: 'y', Cost: 0, Glyph: '=', Color: "fix_kitchen", Kind: KindContainer, Starter: true, Capacity: 8,
		Desc: "Where finished dishes wait. Eight of them."},
	{ID: "pantry_big", Name: "Walk-In Pantry", Level: Ground, Slot: 'x', Cost: 90, Prereq: "pantry", Glyph: '=', Color: "fix_kitchen", Kind: KindContainer, Capacity: 8,
		Desc: "+8 dishes. Room to build a stockpile before market day."},
	{ID: "bed", Name: "Bed", Level: Ground, Slot: 'z', Cost: 0, Glyph: 'b', Color: "fix_living", Kind: KindBed, Starter: true,
		Desc: "Eight hours a night, or the knife slips."},
	// --- living room -------------------------------------------------------
	{ID: "dining_set", Name: "Dining Set", Level: Ground, Slot: 'O', Cost: 100, Glyph: 'T', Color: "fix_living", Kind: KindPassive,
		Desc: "Every evening a four-star dish is on the shelf, neighbors come over and eat it. Heat drops."},
	{ID: "curtains", Name: "Blackout Curtains", Level: Ground, Slot: 'P', Cost: 60, Glyph: '|', Color: "fix_living", Kind: KindPassive,
		Desc: "Inspections see less. Heat cools faster."},
	{ID: "big_tv", Name: "85-inch TV", Level: Ground, Slot: 'Q', Cost: 80, Glyph: 'V', Color: "fix_living", Kind: KindPassive, Draw: 0.02,
		Desc: "Renfields stay loyal when there's something on."},
	{ID: "wine_rack", Name: "Wine Rack", Level: Ground, Slot: 'N', Cost: 60, Glyph: 'Y', Color: "fix_living", Kind: KindPassive,
		Desc: "A nice Chianti. Unlocks the liver with fava beans."},
	// --- closet ------------------------------------------------------------
	{ID: "chair", Name: "The Chair", Level: Ground, Slot: 'R', Cost: 250, Glyph: 'h', Color: "fix_closet", Kind: KindChair, Duration: 360, Draw: 0.02,
		Desc: "Eyelid clamps, Beethoven, a slideshow. Six hours turns livestock into a Renfield who works for you."},
	{ID: "vhs", Name: "Motivational VHS Library", Level: Ground, Slot: 'U', Cost: 120, Prereq: "chair", Glyph: '=', Color: "fix_closet", Kind: KindPassive,
		Desc: "The Chair works in half the time. Tapes 1 through 12 of 'You Belong Here'."},
	// --- garage ------------------------------------------------------------
	{ID: "van", Name: "Windowless Van", Level: Ground, Slot: 'W', Cost: 300, Glyph: 'v', Color: "fix_garage", Kind: KindVan, Duration: 60,
		Desc: "'FREE WIFI' on the side. A worker hauls a victim's car to a lot two counties over in an hour."},
	{ID: "chest_freezer", Name: "Chest Freezer", Level: Ground, Slot: 'X', Cost: 220, Prereq: "freezer", Glyph: '*', Color: "fix_cold", Kind: KindContainer, Capacity: 16, Draw: 0.1,
		Desc: "+16 frozen cuts. Every suburban garage has one. Yours is honest about it."},
	{ID: "chipper", Name: "Wood Chipper", Level: Ground, Slot: 'Y', Cost: 350, Glyph: '@', Color: "fix_garage", Kind: KindDisposal, Duration: 5, Noise: 3, Accepts: []string{"bones", "clothes", "phone"},
		Desc: "Five minutes per item. Very loud. Very Fargo."},
	{ID: "chemistry", Name: "Chemistry Set", Level: Ground, Slot: 'Z', Cost: 90, Glyph: 'q', Color: "fix_garage", Kind: KindPassive,
		Desc: "Chloroform on the mat. The trap fires more often."},
	{ID: "lye_tub", Name: "Plastic Tub of Lye", Level: Ground, Slot: 'a', Cost: 150, Glyph: 'l', Color: "fix_garage", Kind: KindDisposal, Duration: 720, Accepts: []string{"bones", "clothes"},
		Desc: "Twelve hours per item, silent. Plastic, not ceramic. You've seen the show."},
	// --- basement ----------------------------------------------------------
	{ID: "cell_1", Name: "Cell 1", Level: Basement, Slot: 'A', Cost: 0, Glyph: 'n', Color: "fix_cell", Kind: KindCell, Starter: true,
		Desc: "A cot, a bucket, a motivational poster."},
	{ID: "cell_2", Name: "Cell 2", Level: Basement, Slot: 'B', Cost: 100, Prereq: "cell_1", Glyph: 'n', Color: "fix_cell", Kind: KindCell,
		Desc: "Dug over a weekend. The neighbors assumed a wine cellar."},
	{ID: "cell_3", Name: "Cell 3", Level: Basement, Slot: 'C', Cost: 200, Prereq: "cell_2", Glyph: 'n', Color: "fix_cell", Kind: KindCell,
		Desc: "Now it's a wing."},
	{ID: "cell_4", Name: "Cell 4", Level: Basement, Slot: 'D', Cost: 350, Prereq: "cell_3", Glyph: 'n', Color: "fix_cell", Kind: KindCell,
		Desc: "Structural engineer said no. Structural engineer is in cell 4."},
	{ID: "cell_5", Name: "Cell 5", Level: Basement, Slot: 'E', Cost: 500, Prereq: "cell_4", Glyph: 'n', Color: "fix_cell", Kind: KindCell,
		Desc: "The HOA would call it an unpermitted ADU."},
	{ID: "block", Name: "Butcher Block", Level: Basement, Slot: 'K', Cost: 0, Glyph: 'K', Color: "fix_kitchen", Kind: KindBlock, Duration: 60, Starter: true,
		Desc: "An hour per body. Cuts go up to the fridge, the rest goes out the back."},
	{ID: "soundproof", Name: "Soundproofing", Level: Basement, Slot: 'F', Cost: 180, Prereq: "cell_1", Glyph: '#', Color: "fix_cell", Kind: KindPassive,
		Desc: "Egg-crate foam floor to ceiling. Livestock stop radiating Heat."},
	{ID: "gruel", Name: "Gruel Dispenser", Level: Basement, Slot: 'G', Cost: 120, Prereq: "cell_1", Glyph: 'G', Color: "fix_cell", Kind: KindPassive, Draw: 0.02,
		Desc: "Feeds every cell automatically at half the cost. No worker needed."},
	{ID: "buffet", Name: "All-You-Can-Eat Trough", Level: Basement, Slot: 'H', Cost: 150, Prereq: "cell_1", Glyph: 'U', Color: "fix_cell", Kind: KindPassive,
		Desc: "Livestock fatten twice as fast. Marbling is a lifestyle."},
	// --- backyard ----------------------------------------------------------
	{ID: "back_step", Name: "Back Step", Level: Backyard, Slot: 'P', Cost: 0, Glyph: 'x', Color: "heat", Kind: KindContainer, Starter: true, Capacity: 99,
		Desc: "Where the bags pile up until something deals with them. Every bag here is Heat."},
	{ID: "plot_1", Name: "Flower Bed", Level: Backyard, Slot: 'A', Cost: 0, Glyph: '"', Color: "fix_yard", Kind: KindPlot, Starter: true, Capacity: 3, Duration: 30,
		Desc: "Petunias. Room for three underneath. Half an hour with a shovel each."},
	{ID: "plot_2", Name: "Rose Garden", Level: Backyard, Slot: 'B', Cost: 60, Prereq: "plot_1", Glyph: '"', Color: "fix_yard", Kind: KindPlot, Capacity: 4, Duration: 30,
		Desc: "The roses have never looked better. Four more underneath."},
	{ID: "plot_3", Name: "Koi Pond", Level: Backyard, Slot: 'C', Cost: 90, Prereq: "plot_2", Glyph: 'o', Color: "fix_cold", Kind: KindPlot, Capacity: 4, Duration: 20,
		Desc: "The koi are enormous. Four more underneath."},
	{ID: "compost", Name: "Compost Bin", Level: Backyard, Slot: 'D', Cost: 80, Glyph: 'C', Color: "fix_yard", Kind: KindDisposal, Duration: 1440, Accepts: []string{"bones", "clothes"},
		Desc: "A day per item, silent. Black gold for the garden."},
	{ID: "garden", Name: "Heirloom Tomato Garden", Level: Backyard, Slot: 'E', Cost: 120, Prereq: "compost", Glyph: 't', Color: "fix_yard", Kind: KindPassive,
		Desc: "Suspiciously fertile. Pays daily, more for every bag the compost has eaten."},
	{ID: "fence", Name: "Eight-Foot Privacy Fence", Level: Backyard, Slot: 'F', Cost: 200, Glyph: '|', Color: "fix_yard", Kind: KindPassive,
		Desc: "The dog stays out, the yard stays private, Heat cools faster."},
	{ID: "grill", Name: "Kamado Grill", Level: Backyard, Slot: 'G', Cost: 110, Glyph: 'Q', Color: "fix_kitchen", Kind: KindStation,
		Desc: "Ceramic. Unlocks Cul-de-Sac Ribs."},
	{ID: "shed", Name: "Tool Shed", Level: Backyard, Slot: 'H', Cost: 140, Glyph: 'D', Color: "fix_yard", Kind: KindPassive,
		Desc: "Better shovels: burying takes half the time and makes no noise."},
	{ID: "pool", Name: "Pool", Level: Backyard, Slot: 'I', Cost: 400, Prereq: "fence", Glyph: '~', Color: "fix_cold", Kind: KindLure, Draw: 0.06,
		Desc: "Pool party. Influencers arrive uninvited."},
	{ID: "gnome", Name: "Garden Gnome", Level: Backyard, Slot: 'J', Cost: 25, Glyph: 'g', Color: "fix_yard", Kind: KindPassive,
		Desc: "He's seen things. Slightly lucky."},
	// --- attic -------------------------------------------------------------
	{ID: "cameras", Name: "Surveillance Rig", Level: Attic, Slot: 'A', Cost: 250, Glyph: 'o', Color: "fix_attic", Kind: KindPassive, Draw: 0.04,
		Desc: "Livestock trying the door get caught before they get out."},
	{ID: "drying_rack", Name: "Drying Racks", Level: Attic, Slot: 'B', Cost: 100, Glyph: '=', Color: "fix_attic", Kind: KindStation,
		Desc: "Air-cured charcuterie, a day at a time."},
	{ID: "telescope", Name: "Telescope", Level: Attic, Slot: 'C', Cost: 90, Glyph: 'T', Color: "fix_attic", Kind: KindPassive,
		Desc: "For astronomy. Joggers get lured; the trap fires better after dark."},
	{ID: "cot", Name: "Renfield Cot", Level: Attic, Slot: 'D', Cost: 80, Glyph: 'n', Color: "fix_attic", Kind: KindBed,
		Desc: "A Renfield needs somewhere to sleep. One cot, one Renfield."},
	{ID: "cot_2", Name: "Second Cot", Level: Attic, Slot: 'E', Cost: 120, Prereq: "cot", Glyph: 'n', Color: "fix_attic", Kind: KindBed,
		Desc: "Bunk beds, technically."},
	{ID: "soap_molds", Name: "Soap Molds", Level: Attic, Slot: 'F', Cost: 70, Prereq: "stockpot", Glyph: 'm', Color: "fix_attic", Kind: KindPassive,
		Desc: "Paper Street Soap Co. Scraps into money on Etsy, one bar a day."},
}

var fixtureByID = func() map[string]*Fixture {
	m := map[string]*Fixture{}
	for i := range Fixtures {
		m[Fixtures[i].ID] = &Fixtures[i]
	}
	return m
}()

func FixtureByID(id string) *Fixture { return fixtureByID[id] }

// Recipe runs at a Station for Duration minutes, consuming Cuts (or a bag of
// bones), and produces one Dish. Needs are passive fixtures.
type Recipe struct {
	ID       string
	Name     string
	Station  string
	Needs    []string
	Cuts     map[string]int
	Bones    bool
	Duration int
	Restore  float64 // hunger restored
	Base     int     // stars before marbling/freshness adjust
	Price    int     // > 0: sellable
	ShelfMin int     // minutes before it spoils
	Food     bool
	Flavor   string
}

const (
	CutPrime  = "prime"
	CutOffal  = "offal"
	CutScraps = "scraps"
)

const day = 1440

var Recipes = []Recipe{
	{ID: "tartare", Name: "Suburban Tartare", Station: "stove", Cuts: map[string]int{CutPrime: 1}, Duration: 10, Restore: 25, Base: 2, ShelfMin: day, Food: true,
		Flavor: "Raw, capers, a quail egg. Not a recipe so much as a confession."},
	{ID: "pan_sear", Name: "Pan-Seared Neighbor", Station: "stove", Cuts: map[string]int{CutPrime: 1}, Duration: 30, Restore: 40, Base: 3, ShelfMin: 3 * day, Food: true,
		Flavor: "Salt, pepper, butter baste. Rest five minutes. He won't mind."},
	{ID: "liver_onions", Name: "Liver and Onions", Station: "stove", Cuts: map[string]int{CutOffal: 1}, Duration: 30, Restore: 30, Base: 2, ShelfMin: 2 * day, Food: true,
		Flavor: "Grandma's recipe. Grandma's not around to ask where she got the liver."},
	{ID: "fritters", Name: "Farmer Vincent's Fritters", Station: "stove", Cuts: map[string]int{CutScraps: 2}, Duration: 40, Restore: 30, Base: 3, Price: 20, ShelfMin: 3 * day, Food: true,
		Flavor: "It takes all kinds of critters to make Farmer Vincent's fritters."},
	{ID: "fava", Name: "Liver, Fava Beans, a Nice Chianti", Station: "stove", Needs: []string{"wine_rack"}, Cuts: map[string]int{CutOffal: 1}, Duration: 45, Restore: 30, Base: 4, ShelfMin: day, Food: true,
		Flavor: "Fffffffff. You know the noise. Everyone knows the noise."},
	{ID: "meat_pie", Name: "Mrs. Lovett's Meat Pies", Station: "stove", Needs: []string{"grinder"}, Cuts: map[string]int{CutScraps: 1, CutPrime: 1}, Duration: 90, Restore: 35, Base: 3, Price: 30, ShelfMin: 4 * day, Food: true,
		Flavor: "The worst pies in London, the best on the cul-de-sac. Sells out at the stand."},
	{ID: "pulled_pork", Name: "Long Pig Pulled Pork", Station: "smoker", Cuts: map[string]int{CutPrime: 2}, Duration: 840, Restore: 55, Base: 4, Price: 40, ShelfMin: 4 * day, Food: true,
		Flavor: "Fourteen hours over hickory. The secret's in the sauce. The secret is Brayden."},
	{ID: "soylent", Name: "Soylent Green Smoothie", Station: "blender", Cuts: map[string]int{CutScraps: 1}, Duration: 5, Restore: 20, Base: 1, ShelfMin: 2 * day, Food: true,
		Flavor: "It's people. Post-workout, it's people."},
	{ID: "jerky", Name: "Trail Mix (Heavy On The Trail)", Station: "dehydrator", Cuts: map[string]int{CutPrime: 1}, Duration: 480, Restore: 25, Base: 2, Price: 15, ShelfMin: 30 * day, Food: true,
		Flavor: "Keeps a month. Tastes like a jogger, because."},
	{ID: "broth", Name: "Bone Broth (Wellness!)", Station: "stockpot", Bones: true, Duration: 360, Restore: 15, Base: 2, Price: 25, ShelfMin: 5 * day, Food: true,
		Flavor: "Six hours. The evidence becomes collagen. Wellness moms pay by the jar."},
	{ID: "sous_vide", Name: "Sous Vide Loin, 131F", Station: "sous_vide", Cuts: map[string]int{CutPrime: 1}, Duration: 180, Restore: 45, Base: 5, ShelfMin: 2 * day, Food: true,
		Flavor: "Edge-to-edge pink. Chef's kiss. Chef's mouth. Chef's problem."},
	{ID: "ribs", Name: "Cul-de-Sac Ribs", Station: "grill", Cuts: map[string]int{CutPrime: 2}, Duration: 240, Restore: 50, Base: 4, Price: 35, ShelfMin: 3 * day, Food: true,
		Flavor: "Fall-off-the-bone, and the bone is a problem."},
	{ID: "charcuterie", Name: "Delicatessen Board", Station: "drying_rack", Needs: []string{"smoker", "fridge_big"}, Cuts: map[string]int{CutPrime: 1, CutOffal: 1, CutScraps: 1}, Duration: 1440, Restore: 50, Base: 5, Price: 90, ShelfMin: 6 * day, Food: true,
		Flavor: "Cured, smoked, aged. Served with cornichons and a look of understanding."},
	{ID: "soap", Name: "Paper Street Soap", Station: "stockpot", Needs: []string{"soap_molds"}, Cuts: map[string]int{CutScraps: 1}, Duration: 120, Restore: 0, Base: 3, Price: 25, ShelfMin: 999 * day, Food: false,
		Flavor: "Lye, tallow, lavender. First rule: don't talk about the tallow."},
}

var recipeByID = func() map[string]*Recipe {
	m := map[string]*Recipe{}
	for i := range Recipes {
		m[Recipes[i].ID] = &Recipes[i]
	}
	return m
}()

func RecipeByID(id string) *Recipe { return recipeByID[id] }

var relations = []string{
	"Uncle", "Aunt", "Cousin", "Nephew", "Niece", "Grandma", "Grandpa", "Great-Aunt",
	"Stepdad", "Stepmom", "Twin", "Estranged Brother", "Half-Sister", "Godfather",
}

var cannibalNames = []string{
	"Lovett", "Vincent", "Sawyer", "Dexter", "Fester", "Hannibal", "Morticia", "Gomez",
	"Ruth", "Buck", "Eddie", "Leatherface Jr.", "Clarice", "Ravenous Ray", "Sweeney",
	"Raoul", "Delores", "Chianti", "Barbecue Bob", "Mildred", "Ottis", "Patrick",
	"Norma", "Norman", "Ed", "Judith", "Fannie", "Cletus", "Marge", "Tobias",
}

var inspectors = map[int]string{
	1: "Gladys from next door, with a bundt cake and questions",
	2: "the HOA compliance committee, clipboards out",
	3: "a Channel 6 reporter with great hair",
	4: "two podcasters with a boom mic and a Patreon",
	5: "a very polite FBI agent who has already read your file",
}
