package config

import "os"

type Config struct {
	Addr       string
	TLSAddr    string
	CertDir    string
	DBPath     string
	SensorAddr string
	DoorIP     string
}

func Load() Config {
	return Config{
		Addr:       getEnv("ARMORY_ADDR", ":8080"),
		TLSAddr:    getEnv("ARMORY_TLS_ADDR", ":8443"),
		CertDir:    getEnv("ARMORY_CERTS", "certs"),
		DBPath:     getEnv("ARMORY_DB", "armory.db"),
		SensorAddr: getEnv("ARMORY_SENSOR_ADDR", ":47810"),
		DoorIP:     getEnv("ARMORY_DOOR_IP", "192.168.4.49"),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
