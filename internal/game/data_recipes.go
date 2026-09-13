package game

// Recipe turns cuts (or evidence) into a Dish. Needs lists fixtures; Cuts maps
// kind -> count; Restore is hunger restored per serving; Base is the star
// quality before marbling/freshness adjust it; Price > 0 makes it sellable at
// the market; Shelf is days before it spoils. Bones consumes a bones evidence
// instead of a cut. Food=false is a good you sell, never eat.
type Recipe struct {
	ID      string
	Name    string
	Needs   []string
	Cuts    map[string]int
	Bones   bool
	Restore int
	Base    int
	Price   int
	Shelf   int
	Food    bool
	Starter bool
	Flavor  string
}

var Recipes = []Recipe{
	{ID: "tartare", Name: "Suburban Tartare", Cuts: map[string]int{CutPrime: 1}, Restore: 25, Base: 2, Shelf: 1, Food: true, Starter: true,
		Flavor: "Raw, capers, a quail egg. Not a recipe so much as a confession."},
	{ID: "pan_sear", Name: "Pan-Seared Neighbor", Needs: []string{"stove"}, Cuts: map[string]int{CutPrime: 1}, Restore: 40, Base: 3, Shelf: 3, Food: true, Starter: true,
		Flavor: "Salt, pepper, butter baste. Rest five minutes. He won't mind."},
	{ID: "fritters", Name: "Farmer Vincent's Fritters", Needs: []string{"stove"}, Cuts: map[string]int{CutScraps: 2}, Restore: 30, Base: 3, Price: 20, Shelf: 3, Food: true, Starter: true,
		Flavor: "It takes all kinds of critters to make Farmer Vincent's fritters."},
	{ID: "liver_onions", Name: "Liver and Onions", Needs: []string{"stove"}, Cuts: map[string]int{CutOffal: 1}, Restore: 30, Base: 2, Shelf: 2, Food: true, Starter: true,
		Flavor: "Grandma's recipe. Grandma's not around to ask where she got the liver."},
	{ID: "fava", Name: "Liver, Fava Beans, a Nice Chianti", Needs: []string{"stove", "wine_rack"}, Cuts: map[string]int{CutOffal: 1}, Restore: 30, Base: 4, Shelf: 1, Food: true,
		Flavor: "Fffffffff. You know the noise. Everyone knows the noise."},
	{ID: "meat_pie", Name: "Mrs. Lovett's Meat Pies", Needs: []string{"stove", "grinder"}, Cuts: map[string]int{CutScraps: 1, CutPrime: 1}, Restore: 35, Base: 3, Price: 30, Shelf: 4, Food: true,
		Flavor: "The worst pies in London, the best on the cul-de-sac. Sells out at the market."},
	{ID: "pulled_pork", Name: "Long Pig Pulled Pork", Needs: []string{"smoker"}, Cuts: map[string]int{CutPrime: 2}, Restore: 55, Base: 4, Price: 40, Shelf: 4, Food: true,
		Flavor: "Fourteen hours over hickory. The secret's in the sauce. The secret is Brayden."},
	{ID: "soylent", Name: "Soylent Green Smoothie", Needs: []string{"blender"}, Cuts: map[string]int{CutScraps: 1}, Restore: 20, Base: 1, Shelf: 2, Food: true,
		Flavor: "It's people. Post-workout, it's people."},
	{ID: "jerky", Name: "Trail Mix (Heavy On The Trail)", Needs: []string{"dehydrator"}, Cuts: map[string]int{CutPrime: 1}, Restore: 25, Base: 2, Price: 15, Shelf: 30, Food: true,
		Flavor: "Keeps a month. Tastes like a jogger, because."},
	{ID: "broth", Name: "Bone Broth (Wellness!)", Needs: []string{"stove", "stockpot"}, Bones: true, Restore: 15, Base: 2, Price: 25, Shelf: 5, Food: true,
		Flavor: "Simmer 36 hours. The evidence becomes collagen. Wellness moms pay by the jar."},
	{ID: "sous_vide", Name: "Sous Vide Loin, 131F", Needs: []string{"sous_vide", "stove"}, Cuts: map[string]int{CutPrime: 1}, Restore: 45, Base: 5, Shelf: 2, Food: true,
		Flavor: "Edge-to-edge pink. Chef's kiss. Chef's mouth. Chef's problem."},
	{ID: "ribs", Name: "Cul-de-Sac Ribs", Needs: []string{"grill"}, Cuts: map[string]int{CutPrime: 2}, Restore: 50, Base: 4, Price: 35, Shelf: 3, Food: true,
		Flavor: "The secret's in the sauce. The dinner party favorite. Fall-off-the-bone, and the bone is a problem."},
	{ID: "charcuterie", Name: "Delicatessen Board", Needs: []string{"smoker", "drying_rack", "fridge_big"}, Cuts: map[string]int{CutPrime: 1, CutOffal: 1, CutScraps: 1}, Restore: 50, Base: 5, Price: 90, Shelf: 6, Food: true,
		Flavor: "Cured, smoked, aged. Served with cornichons and a look of understanding."},
	{ID: "soap", Name: "Paper Street Soap", Needs: []string{"stove", "stockpot", "soap_molds"}, Cuts: map[string]int{CutScraps: 1}, Restore: 0, Base: 3, Price: 25, Shelf: 999, Food: false,
		Flavor: "Lye, tallow, lavender. First rule: don't talk about the tallow."},
}

var recipeByID = func() map[string]*Recipe {
	m := map[string]*Recipe{}
	for i := range Recipes {
		m[Recipes[i].ID] = &Recipes[i]
	}
	return m
}()

// RecipeByID returns nil for an unknown id.
func RecipeByID(id string) *Recipe { return recipeByID[id] }

// CanCook reports whether the fixtures for a recipe are owned. Recipes unlock
// by fixture, not by purchase: own the smoker, know the pulled pork.
func (s *State) CanCook(r *Recipe) bool {
	for _, n := range r.Needs {
		if !s.Has(n) {
			return false
		}
	}
	return true
}
