package config

import "os"

type Config struct {
	Addr   string
	DBPath string
}

func Load() Config {
	return Config{
		Addr:   getEnv("ARMORY_ADDR", ":8080"),
		DBPath: getEnv("ARMORY_DB", "armory.db"),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
