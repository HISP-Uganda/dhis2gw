package utils

import (
	"math/rand"
	"strings"
	"time"
)

const alphabet = `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ`
const allowedCharacters = "0123456789" + alphabet
const codeSize = 11

// GenerateUID return a Unique ID for our resources
func GenerateUID() string {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source) // Creates a new instance of rand.Rand, safe for concurrent use

	numberOfCodePoints := len(allowedCharacters)

	var s strings.Builder
	s.Grow(codeSize) // Pre-allocate memory to improve performance

	// Ensure the first character is an uppercase letter from the alphabet
	s.WriteByte(allowedCharacters[r.Intn(26)] - 32) // Convert to uppercase

	// Generate the rest of the UID
	for i := 1; i < codeSize; i++ {
		s.WriteByte(allowedCharacters[r.Intn(numberOfCodePoints)])
	}

	return s.String()
}

func LastXDays(x int) []string {
	days := make([]string, x)
	for i := 0; i < x; i++ {
		days[i] = time.Now().AddDate(0, 0, -(x - 1 - i)).Format("2006-01-02")
	}
	return days
}

func DaysInRange(startDay, endDay string) []string {
	startDate, err := time.Parse("2006-01-02", startDay)
	if err != nil {
		return nil
	}
	endDate, err := time.Parse("2006-01-02", endDay)
	if err != nil {
		return nil
	}

	days := make([]string, 0, int(endDate.Sub(startDate).Hours()/24)+1)
	for t := startDate; !t.After(endDate); t = t.AddDate(0, 0, 1) {
		days = append(days, t.Format("2006-01-02"))
	}
	return days
}

func IndexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}
