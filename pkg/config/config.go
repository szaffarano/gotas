package config

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
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
	// @TODO: verify if the key exists?
	return c.values[key]
}

func (c *Config) GetBool(key string) (value bool) {
	// @TODO: verify if the key exists?
	if str, ok := c.values[key]; ok {
		if value, err := strconv.ParseBool(str); err == nil {
			return value
		}
	}

	return
}

func (c *Config) GetInt(key string) (value int) {
	if str, ok := c.values[key]; ok {
		if value, err := strconv.Atoi(str); err == nil {
			return value
		}
	}

	return
}

func New(path string) (Config, error) {
	cfg := Config{
		path:   path,
		values: make(map[string]string),
	}

	if err := Save(cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func Load(path string) (Config, error) {
	cfg := Config{}

	file, err := os.Open(path)
	if err != nil {
		return cfg, errors.Wrap(err, "error opening config file")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	values := make(map[string]string)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" && !strings.HasPrefix(line, "#") {
			splitted := strings.Split(line, "=")
			if len(splitted) != 2 {
				return cfg, fmt.Errorf("error parsing configuation file: %q", line)
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
		return errors.Wrap(err, "error opening config file")
	}
	defer file.Close()

	// sort the keys to serialize the values deterministically
	keys := make([]string, 0, len(config.values))
	for k := range config.values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buffer bytes.Buffer
	for _, k := range keys {
		fmt.Fprintf(&buffer, "%s = %v\n", k, config.values[k])
	}

	if count, err := file.Write(buffer.Bytes()); err != nil {
		return errors.Wrap(err, "error saving config file")
	} else if count != len(buffer.Bytes()) {
		return errors.Wrap(err, "error saving config file, unexpected bytes saved")
	}

	return nil
}
