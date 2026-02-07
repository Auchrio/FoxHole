package utils

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Secret  string
	Timeout int
}

func LoadConfig() *Config {
	cfg := &Config{
		Secret:  "super-secret-key",
		Timeout: 30,
	}

	exe, err := os.Executable()
	if err != nil {
		return cfg
	}

	confPath := filepath.Join(filepath.Dir(exe), "pulse.conf")
	file, err := os.Open(confPath)
	if err != nil {
		return cfg
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "user-secret":
			if value != "" {
				cfg.Secret = value
			}
		case "listen-timeout":
			if t, err := strconv.Atoi(value); err == nil && t >= 0 {
				cfg.Timeout = t
			}
		}
	}

	return cfg
}
