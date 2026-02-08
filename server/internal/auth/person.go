package auth

import "sync"

// ValidPersons is the list of valid person identifiers.
var ValidPersons = []string{"sebastian", "petra"}
var validPersonsMu sync.RWMutex

// SetValidPersons replaces the valid person identifiers.
// Empty values are ignored.
func SetValidPersons(persons []string) {
	filtered := make([]string, 0, len(persons))
	for _, p := range persons {
		if p != "" {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) == 0 {
		return
	}
	validPersonsMu.Lock()
	ValidPersons = filtered
	validPersonsMu.Unlock()
}

// IsValidPerson checks if the given person identifier is valid.
func IsValidPerson(person string) bool {
	validPersonsMu.RLock()
	defer validPersonsMu.RUnlock()
	for _, p := range ValidPersons {
		if p == person {
			return true
		}
	}
	return false
}
