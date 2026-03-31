package config

import "os"

type Config struct {
	DatabaseURL    string
	Port           string
	RPName         string
	RPID           string
	RPOrigin       string
	IsProd         bool
	AllowedOrigins []string
}

func FromEnv() Config {
	rpOrigin := getEnv("RP_ORIGIN", "http://localhost:3001")
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
		RPID:           getEnv("RP_ID", "localhost"),
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
