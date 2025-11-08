package utils

func Queues(prefix string) map[string]int {
	p := ""
	if prefix != "" {
		p = prefix + ":"
	}
	return map[string]int{
		p + "critical": 6,
		p + "default":  3,
		p + "low":      1,
	}
}

// GetDefaultQueue returns the default queue name based on the given prefix.
func GetDefaultQueue(prefix string) string {
	if prefix != "" {
		return prefix + ":default"
	}
	return "default"
}
