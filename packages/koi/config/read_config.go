package config

import (
	"errors"
	"github.com/goccy/go-yaml"
	"koi/env"
	"os"
	"path/filepath"
	"strings"
)

var (
	redirectPath = createRedirectPath()
)

func ReadConfig(path string) (*KoiConfig, error) {
	config, err := readConfigIntl(path, 1)
	if err != nil {
		l.Error(err.Error())
		return nil, err
	}
	return config, nil
}

func createRedirectPath() *yaml.Path {
	r, err := yaml.PathString("$.redirect")
	if err != nil {
		l.Fatal("Err create redirect yaml path.")
	}
	return r
}

func readConfigIntl(path string, recur int) (*KoiConfig, error) {
	if recur >= 64 {
		return nil, errors.New("infinite redirection detected. Check your koi.config file")
	}

	l.Debugf("Loading config: %s", path)

	absPath := path
	if !filepath.IsAbs(absPath) {
		absPath = env.Resolve(env.DirName, absPath)
	}
	configDir := filepath.Dir(absPath)

	l.Debugf("Reading config file: %s", absPath)
	l.Debugf("Config dir: %s", configDir)
	file, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	l.Debug("Detecting redirect field.")
	var redirect string
	err = redirectPath.Read(strings.NewReader(string(file)), &redirect)
	if err == nil {
		l.Debugf("'redirect' field detected: %s", redirect)
		return readConfigIntl(env.Resolve(configDir, redirect), recur+1)
	}

	l.Debug("Parsing config.")
	config := new(KoiConfig)
	err = yaml.Unmarshal(file, config)
	if err != nil {
		return nil, err
	}

	l.Debug("Config parsed successfully.")

	l.Debug("Now processing postConfig.")
	config.InternalConfigDir = configDir
	postConfig(config)

	return config, nil
}

func postConfig(c *KoiConfig) {
	c.InternalDataDir = env.Resolve(c.InternalConfigDir, "data")
	c.InternalHomeDir = env.Resolve(c.InternalDataDir, "home")
	c.InternalNodeDir = env.Resolve(c.InternalDataDir, "node")
	c.InternalTempDir = env.Resolve(c.InternalDataDir, "tmp")
	c.InternalInstanceDir = env.Resolve(c.InternalDataDir, "instances")
}
