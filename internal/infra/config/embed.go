package config

import (
	_ "embed"
	"os"
)

// configFileName is the config file looked up (and, when absent, materialised)
// in the working directory. It is intentionally kept in sync with the viper
// config name ("config" + ".yaml").
const configFileName = "config.yaml"

// defaultConfig is the built-in configuration embedded into the binary. It is
// written out as config.yaml on first run when no config file exists yet, so a
// freshly deployed binary is self-bootstrapping.
//
//go:embed default_config.yaml
var defaultConfig []byte

// ensureConfigFile materialises the embedded default config to configFileName
// when that file does not yet exist. If the file is already present it is left
// untouched (the user's config always wins). Any other stat error is returned
// so the caller can surface it.
func ensureConfigFile() error {
	if _, err := os.Stat(configFileName); err == nil {
		// Config already exists: read it as-is, do nothing else.
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	// No config file: release the embedded default.
	return os.WriteFile(configFileName, defaultConfig, 0o644)
}
