package auth

// ValidPersons is the list of valid person identifiers.
var ValidPersons = []string{"sebastian", "petra"}

// IsValidPerson checks if the given person identifier is valid.
func IsValidPerson(person string) bool {
	for _, p := range ValidPersons {
		if p == person {
			return true
		}
	}
	return false
}
