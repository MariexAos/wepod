package ops

import "os"

// getEnv returns the named env var or fallback if empty.
func getEnv(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
