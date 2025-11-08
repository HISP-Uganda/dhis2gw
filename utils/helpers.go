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
	values := make([]string, 0, len(dataValues))
	for _, v := range dataValues {
		if ValidUID(v) {
			values = append(values, v)
		}
	}
	return values
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
func MissingStrings(subset, set []string) []string {
	setMap := make(map[string]struct{}, len(set))
	for _, s := range set {
		setMap[s] = struct{}{}
	}

	var missing []string
	for _, s := range subset {
		if _, ok := setMap[s]; !ok {
			missing = append(missing, s)
		}
	}
	return missing
}
