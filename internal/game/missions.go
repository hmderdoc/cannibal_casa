package game

// MissionDef is an away-from-home job. Phases is how many phase ends it takes;
// while away, the house runs on Renfield loyalty and security fixtures.
type MissionDef struct {
	ID      string
	Name    string
	Phases  int
	MinTier int
	Desc    string
}

var Missions = []MissionDef{
	{ID: "day_job", Name: "Remote Work", Phases: 2,
		Desc: "Three Zoom calls, camera off, 'consulting'. Pays $40 and nobody asks what you do."},
	{ID: "market", Name: "Farmers Market", Phases: 1,
		Desc: "Sell pies, jerky, broth, soap and tomatoes to people who ask if it's organic. It is."},
	{ID: "city_hunt", Name: "Hunting Trip Downtown", Phases: 2,
		Desc: "Nobody downtown knows you. Come back with someone. Takes the evening."},
	{ID: "tinder", Name: "Dinner Date", Phases: 2,
		Desc: "Swipe right on dinner. Your bio says 'loves to cook'. Not a lie."},
	{ID: "burglary", Name: "Five-Finger Shopping", Phases: 1,
		Desc: "The Hendersons are in Cabo. Their garage is not."},
	{ID: "vigilante", Name: "The Code", Phases: 3, MinTier: 1,
		Desc: "Only the guilty. A guy who definitely had it coming. The street sleeps easier; so do you."},
}

var missionByID = func() map[string]*MissionDef {
	m := map[string]*MissionDef{}
	for i := range Missions {
		m[Missions[i].ID] = &Missions[i]
	}
	return m
}()

func MissionByID(id string) *MissionDef { return missionByID[id] }
