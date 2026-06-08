package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL    string
	Port           string
	RPName         string
	RPID           string
	RPOrigin       string
	IsProd         bool
	AllowedOrigins []string
}

// loadDotEnv reads KEY=VALUE pairs from path and sets any that are not already
// present in the process environment. Unreadable files are silently ignored.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		// Strip optional surrounding quotes
		if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
			v = v[1 : len(v)-1]
		}
		if k != "" && os.Getenv(k) == "" {
			os.Setenv(k, v) //nolint:errcheck
		}
	}
}

func FromEnv() Config {
	loadDotEnv(".env")
	rpOrigin := getEnv("RP_ORIGIN", "https://thufir.majid.org")
	origins := []string{
		"http://localhost:5173",
		"http://localhost:4173",
		rpOrigin,
	}
	// Deduplicate
	seen := map[string]struct{}{}
	unique := origins[:0]
	for _, o := range origins {
		if _, ok := seen[o]; !ok {
			seen[o] = struct{}{}
			unique = append(unique, o)
		}
	}

	return Config{
		DatabaseURL:    getEnv("DATABASE_URL", ""),
		Port:           getEnv("PORT", "3001"),
		RPName:         "Thufir",
		RPID:           getEnv("RP_ID", "thufir.majid.org"),
		RPOrigin:       rpOrigin,
		IsProd:         getEnv("GO_ENV", "") == "production",
		AllowedOrigins: unique,
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
