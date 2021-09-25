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

type Config struct {
	path   string
	values map[string]string
}

func (c *Config) Set(key, value string) {
	c.values[key] = value
}

func (c *Config) SetInt(key string, value int) {
	c.values[key] = strconv.Itoa(value)
}

func (c *Config) SetBool(key string, value bool) {
	c.values[key] = strconv.FormatBool(value)
}

func (c *Config) Get(key string) string {
	// @TODO return error when the key does not exist?
	return c.values[key]
}

func (c *Config) GetInt(key string) (value int) {
	// @TODO return error when the key does not exist or conversion fails?
	if str, ok := c.values[key]; ok {
		value, _ = strconv.Atoi(str)
	}

	return
}

func (c *Config) GetBool(key string) (value bool) {
	// @TODO return error when the key does not exist or conversion fails?
	if str, ok := c.values[key]; ok {
		value, _ = strconv.ParseBool(str)
	}

	return
}

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
