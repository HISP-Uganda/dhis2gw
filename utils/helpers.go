package utils

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/goccy/go-json"
)

const alphabet = `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ`
const allowedCharacters = "0123456789" + alphabet
const codeSize = 11

// GenerateUID return a Unique ID for our resources
func GenerateUID() string {
	// Use crypto/rand for better uniqueness guarantees in production systems.
	// For this example's use of math/rand, we use the recommended initialization
	// for non-concurrent scenarios where performance is key.
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	numberOfAllowedChars := len(allowedCharacters)
	numberOfAlphabetChars := len(alphabet)

	var s strings.Builder
	s.Grow(codeSize) // Pre-allocate memory to improve performance

	// Ensure the first character is a letter from the alphabet
	s.WriteByte(alphabet[r.Intn(numberOfAlphabetChars)])

	// Generate the rest of the UID using all allowed characters
	for i := 1; i < codeSize; i++ {
		s.WriteByte(allowedCharacters[r.Intn(numberOfAllowedChars)])
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

func PrettyPrint(v interface{}) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(string(b))
}

func ToPrettyJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

var uidRegex = regexp.MustCompile(`^[a-zA-Z0-9]{11}$`)

func ValidUID(s string) bool {
	return uidRegex.MatchString(s)
}

func FilterValidUIDs(dataValues map[string]string) map[string]string {
	filtered := make(map[string]string)
	for k, v := range dataValues {
		if ValidUID(k) {
			filtered[k] = v
		}
	}
	return filtered
}

func FilterValidUIDsSlice(dataValues map[string]string) []string {
	keys := make([]string, 0, len(dataValues))
	for k := range dataValues {
		if ValidUID(k) {
			keys = append(keys, k)
		}
	}
	return keys
}

// CoalesceString returns the first non-empty string
func CoalesceString(s ...string) string {
	for _, v := range s {
		if v != "" {
			return v
		}
	}
	return ""
}

// AnyMissing returns true if any string in subset is not found in set.
func AnyMissing(subset, set []string) bool {
	setMap := make(map[string]struct{}, len(set))
	for _, s := range set {
		setMap[s] = struct{}{}
	}

	for _, s := range subset {
		if _, ok := setMap[s]; !ok {
			return true
		}
	}
	return false
}

// MissingStrings returns a slice of strings from subset that are not found in set.
func MissingStrings(subset []string, set map[string]struct{}) []string {
	var missing []string

	for _, s := range subset {
		if _, ok := set[s]; !ok {
			missing = append(missing, s)
		}
	}

	return missing
}

// SetDifference returns keys in set that are not found in subset.
func SetDifference(subset []string, set map[string]struct{}) []string {
	// Convert subset slice to a fast lookup map
	subsetMap := make(map[string]struct{}, len(subset))
	for _, s := range subset {
		subsetMap[s] = struct{}{}
	}

	// Check which keys in set are missing from subset
	var extras []string
	for key := range set {
		if _, ok := subsetMap[key]; !ok {
			extras = append(extras, key)
		}
	}

	return extras
}
