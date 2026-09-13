package game

// Victim is an archetype of suburban adult who will not be missed (much).
// Marbling drives meal quality, Fight drives capture difficulty, Notice is how
// many days until someone reports them missing, HeatMissing is the Heat that
// lands when they do. Glyph/Color are hints for the door's renderer.
type Victim struct {
	ID          string
	Name        string
	Glyph       rune
	Color       string   // palette name
	Marbling    int      // 1..5
	Fight       int      // 0..5
	Notice      int      // days until reported missing
	HeatMissing int      // heat when reported (negative = the world is relieved)
	Weight      int      // base spawn weight
	Lures       []string // fixtures that raise this archetype's spawn weight
	Pair        bool     // arrives in twos
	MinTier     int      // only shows up once Heat reaches this rung
	Drives      bool     // leaves a car in your driveway
	Flavor      string
}

var Victims = []Victim{
	{ID: "solar", Name: "Door-to-Door Solar Salesman", Glyph: 's', Color: "victim_easy", Marbling: 3, Fight: 1, Notice: 14, HeatMissing: 5, Weight: 12,
		Lures: []string{"lure_openhouse"}, Drives: true,
		Flavor: "Nobody has ever missed one. His company thinks he quit. His mother thinks he's in Tucson."},
	{ID: "mlm", Name: "MLM Hun", Glyph: 'm', Color: "victim_easy", Marbling: 3, Fight: 1, Notice: 10, HeatMissing: 6, Weight: 10,
		Lures:  []string{"lure_garage_sale", "lure_wifi"},
		Flavor: "Hey hun!! Quick question, would you like to join my downline? No. Would you like to join my dinner? Also no, but here we are."},
	{ID: "techbro", Name: "Tech Bro", Glyph: 't', Color: "victim_prime", Marbling: 5, Fight: 2, Notice: 5, HeatMissing: 15, Weight: 8,
		Lures: []string{"lure_wifi", "lure_airbnb"}, Drives: true,
		Flavor: "Keto, intermittent fasting, grass-fed everything. Beautifully marbled. Insufferable, even now."},
	{ID: "vegan", Name: "Militant Vegan", Glyph: 'v', Color: "victim_prime", Marbling: 4, Fight: 1, Notice: 6, HeatMissing: 10, Weight: 8,
		Lures:  []string{"lure_garage_sale", "garden"},
		Flavor: "Grass-fed, free-range, no antibiotics. Premium. The irony is the seasoning."},
	{ID: "crossfit", Name: "CrossFit Guy", Glyph: 'c', Color: "victim_hard", Marbling: 1, Fight: 5, Notice: 3, HeatMissing: 12, Weight: 7,
		Lures:  []string{"lure_couch"},
		Flavor: "Lean. Fights back. Will tell you about his WOD from inside the cell."},
	{ID: "jogger", Name: "3 A.M. Jogger", Glyph: 'j', Color: "victim_hard", Marbling: 2, Fight: 3, Notice: 4, HeatMissing: 10, Weight: 7,
		Lures:  []string{"telescope"},
		Flavor: "Reflective vest, headphones in, zero situational awareness."},
	{ID: "delivery", Name: "Package Delivery Driver", Glyph: 'd', Color: "victim_mid", Marbling: 3, Fight: 2, Notice: 3, HeatMissing: 14, Weight: 8,
		Lures: []string{"lure_airbnb", "lure_openhouse"}, Drives: true,
		Flavor: "Fast, tired, on a schedule. The packages will keep coming. The packages will pile up."},
	{ID: "missionary", Name: "Missionary", Glyph: 'M', Color: "victim_mid", Marbling: 3, Fight: 1, Notice: 9, HeatMissing: 12, Weight: 6, Pair: true,
		Lures:  []string{"lure_couch"},
		Flavor: "They travel in pairs. Two for one. They knocked. You answered. Everyone's day changed."},
	{ID: "influencer", Name: "Lifestyle Influencer", Glyph: 'i', Color: "victim_mid", Marbling: 3, Fight: 1, Notice: 7, HeatMissing: 22, Weight: 6,
		Lures: []string{"lure_airbnb", "pool"}, Drives: true,
		Flavor: "Her followers assumed a digital detox. On day seven they assumed the worst and were correct."},
	{ID: "karen", Name: "Wants To Speak To The Manager", Glyph: 'K', Color: "victim_hard", Marbling: 3, Fight: 3, Notice: 5, HeatMissing: 12, Weight: 7,
		Lures: []string{"lure_garage_sale", "lure_openhouse"}, Drives: true,
		Flavor: "You are the manager. You are also the butcher. She had notes on both."},
	{ID: "landlord", Name: "Landlord", Glyph: 'L', Color: "victim_prime", Marbling: 4, Fight: 1, Notice: 20, HeatMissing: 3, Weight: 5,
		Lures: []string{"lure_airbnb"}, Drives: true,
		Flavor: "His tenants threw a party. Nobody has checked on him and nobody will."},
	{ID: "crypto", Name: "Crypto Evangelist", Glyph: '$', Color: "victim_prime", Marbling: 4, Fight: 2, Notice: 8, HeatMissing: 8, Weight: 6,
		Lures:  []string{"lure_wifi"},
		Flavor: "All that steakhouse networking. Marbled like a portfolio, and the portfolio was also fake."},
	{ID: "hoa", Name: "HOA Inspector", Glyph: 'H', Color: "victim_heat", Marbling: 2, Fight: 2, Notice: 2, HeatMissing: 20, Weight: 4, MinTier: 2, Drives: true,
		Flavor: "Cited your mailbox. Cited your hedge. Did not cite the basement, because he never got that far."},
	{ID: "podcaster", Name: "True Crime Podcaster", Glyph: 'P', Color: "victim_heat", Marbling: 2, Fight: 2, Notice: 4, HeatMissing: -20, Weight: 5, MinTier: 3,
		Flavor: "Came to the street chasing a story. The final episode was mysteriously never recorded. Downloads went way up."},
	{ID: "vigilante_target", Name: "Guy Who Definitely Had It Coming", Glyph: 'X', Color: "victim_easy", Marbling: 3, Fight: 3, Notice: 30, HeatMissing: -10, Weight: 0,
		Flavor: "The code says only the guilty. The code also says brine overnight."},
}

var victimByID = func() map[string]*Victim {
	m := map[string]*Victim{}
	for i := range Victims {
		m[Victims[i].ID] = &Victims[i]
	}
	return m
}()

// VictimByID looks up an archetype; unknown ids return a harmless placeholder.
func VictimByID(id string) *Victim {
	if v, ok := victimByID[id]; ok {
		return v
	}
	return &Victim{ID: id, Name: "Stranger", Glyph: '?', Color: "victim_mid", Marbling: 2, Fight: 2, Notice: 7, HeatMissing: 10}
}

// Captive first names, so the basement roster reads like a dinner list.
var captiveNames = []string{
	"Brayden", "Kayleigh", "Chad", "Tiffani", "Hunter", "Mackenzie", "Tanner", "Brittani",
	"Blake", "Skyler", "Chip", "Bethany", "Dakota", "Jaxon", "Paisley", "Colton",
	"Madison", "Trent", "Krystal", "Kyle", "Amber", "Preston", "Destiny", "Wyatt",
}
