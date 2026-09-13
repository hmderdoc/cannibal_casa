package game

import "math/rand"

// The dynasty: each new cannibal is a relative who "inherits the house".
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

func randomCannibalName(r *rand.Rand) string {
	return relations[r.Intn(len(relations))] + " " + cannibalNames[r.Intn(len(cannibalNames))]
}

// Inspection visitors per Heat tier.
var inspectors = map[int]string{
	1: "Gladys from next door, with a bundt cake and questions",
	2: "the HOA compliance committee, clipboards out",
	3: "a Channel 6 news van and a reporter with great hair",
	4: "two podcasters with a boom mic and a Patreon",
	5: "a very polite FBI agent who has already read your file",
}
