package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Config represents a generic key=value plain-text configuration
type Config struct {
	path   string
	values map[string]string
}

// Set sets a new value in the configuration.  Overrides an existent value.
func (c *Config) Set(key, value string) {
	c.values[key] = value
}

// SetInt sets a new int value in the configuration.  Overrides an existent
// value.
func (c *Config) SetInt(key string, value int) {
	c.values[key] = strconv.Itoa(value)
}

// SetBool sets a new int value in the configuration.  Overrides an existent
// value.
func (c *Config) SetBool(key string, value bool) {
	c.values[key] = strconv.FormatBool(value)
}

// Get returns the value associated to the given key or the zero value ("") if
// it doesn't exist.
func (c *Config) Get(key string) string {
	// @TODO return error when the key does not exist?
	return c.values[key]
}

// GetInt returns the value as integer associated to the given key or the zero
// value (0) if it doesn't exist or the value can't be parsed as number.
func (c *Config) GetInt(key string) (value int) {
	if str, ok := c.values[key]; ok {
		value, _ = strconv.Atoi(str)
	}
	return
}

// GetBool returns the value as a boolean associated to the given key or the zero
// value (false) if it doesn't exist or the value can't be parsed as a bool.
func (c *Config) GetBool(key string) (value bool) {
	if str, ok := c.values[key]; ok {
		value, _ = strconv.ParseBool(str)
	}
	return
}

// New creates an empty configuration and store it in a given file.
func New(path string) (Config, error) {
	cfg := Config{
		path:   path,
		values: make(map[string]string),
	}

	if err := Save(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Load loads a configuration from a given file.  The file has to have pairs
// of key=value lines.  Empty lines or starting with "#" will be ignored.
func Load(path string) (Config, error) {
	cfg := Config{}
	file, err := os.Open(path)
	if err != nil {
		return cfg, fmt.Errorf("open file %v: %v", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	values := make(map[string]string)
	for scanner.Scan() {
		line := scanner.Text()
		// skip comments and blank lines
		if strings.Trim(line, " ") != "" && !strings.HasPrefix(line, "#") {
			splitted := strings.Split(line, "=")
			if len(splitted) != 2 {
				return cfg, fmt.Errorf("parse line: %v", line)
			}

			values[strings.TrimRight(splitted[0], " ")] = strings.TrimLeft(splitted[1], " ")
		}
	}

	cfg.path = path
	cfg.values = values

	return cfg, nil
}

// Save stores the configuration in the file set when initialized.  In case it
// fails because the configuration wasn't not properly initialized or there is
// an error saving the file, it will return an error.
func Save(config Config) error {
	if config.path == "" {
		return errors.New("uninitialized config")
	}

	file, err := os.Create(config.path)
	if err != nil {
		return fmt.Errorf("open file %v: %v", config.path, err)
	}
	defer file.Close()

	// sort the keys to serialize the values deterministically
	var builder strings.Builder
	for _, k := range sortKeys(config.values) {
		fmt.Fprintf(&builder, "%s = %v\n", k, config.values[k])
	}

	buffer := []byte(builder.String())
	if _, err := file.Write(buffer); err != nil {
		return fmt.Errorf("save file %v: %v", config.path, err)
	}

	return nil
}

func sortKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
