// Package config contains configuration loading, validation,
// field catalog indexing and restructuring utilities used by
// the PrivX inventory import tool.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/SSHcom/inventory_import_privx_dev/internal/field"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
	"github.com/SSHcom/inventory_import_privx_dev/internal/util"
	toml "github.com/pelletier/go-toml/v2"
)

type ConfigKey string

const (
	ConfigKeyBaseURL           ConfigKey = "base_url"
	ConfigKeyJWT               ConfigKey = "jwt"
	ConfigKeyClientID          ConfigKey = "client_id"
	ConfigKeyClientSecret      ConfigKey = "client_secret"
	ConfigKeyOAuthClientID     ConfigKey = "oauth_client_id"
	ConfigKeyOAuthClientSecret ConfigKey = "oauth_client_secret"

	ConfigKeyDeleteMode         ConfigKey = "delete_mode"
	ConfigKeyAllowedServices    ConfigKey = "allowed_services"
	ConfigKeyAllowedDBProtocols ConfigKey = "allowed_db_protocols"
	ConfigKeyUpdateMode         ConfigKey = "update_mode"
	ConfigKeyManageMode         ConfigKey = "manage_mode"

	ConfigKeyMaxRecords ConfigKey = "max_records"
)

// envPrefix is prepended to env variable names.
const envPrefix = "PRIVX_"

// AllConfigKeys contains all supported configuration keys.
var AllConfigKeys = []ConfigKey{
	ConfigKeyBaseURL,
	ConfigKeyJWT,
	ConfigKeyClientID,
	ConfigKeyClientSecret,
	ConfigKeyOAuthClientID,
	ConfigKeyOAuthClientSecret,
	ConfigKeyDeleteMode,
	ConfigKeyAllowedServices,
	ConfigKeyAllowedDBProtocols,
	ConfigKeyUpdateMode,
	ConfigKeyMaxRecords,
	ConfigKeyManageMode,
}

// EnvName returns the environment variable name.
func (ck ConfigKey) EnvName() string {
	return envPrefix + strings.ToUpper(string(ck))
}

// ConfigValues is an array of configuration values that have been read from a source.
type ConfigValues map[ConfigKey]string

// Hard coded defaults for configuration
var Defaults = ConfigValues{
	ConfigKeyDeleteMode:         string(DeleteModeDisable),
	ConfigKeyAllowedServices:    "SSH, RDP, VNC, WEB, DB",
	ConfigKeyAllowedDBProtocols: "postgres, mysql, passthrough, tls",
	ConfigKeyUpdateMode:         string(UpdateModeCreate),
	ConfigKeyMaxRecords:         "100000",
	ConfigKeyManageMode:         string(DefaulManageModeManaged),
}

var (
	// ErrInvalidConfigValue is returned if the config file has invalid values
	ErrInvalidConfigValue = errors.New("invalid value in config file")

	// ErrConfigDirNotFound is returned when no configuration directory can be resolved.
	ErrConfigDirNotFound = fmt.Errorf("config directory not found")
)

// DeleteMode defines the action taken when a host is no longer in imported file.
type DeleteMode string

const (
	DeleteModeDelete  DeleteMode = "delete"
	DeleteModeDisable DeleteMode = "disable"
)

func (dm DeleteMode) AllowedValues() []string {
	return []string{
		string(DeleteModeDelete),
		string(DeleteModeDisable),
	}
}

type DefaultManageMode string

const (
	DefaulManageModeManaged = "managed"
	DefaulManageModeIgnored = "ignored"
)

// UpdateMode defines how importer handles existing or missing hosts.
type UpdateMode string

const (
	// UpdateModeCreate allows only the creation of new hosts.
	UpdateModeCreate UpdateMode = "addonly"
	// UpdateModeModify allows creation of new hosts and modification of existing ones.
	UpdateModeModify UpdateMode = "modify"
	// UpdateModeSync allows full control of hosts, including deletion.
	UpdateModeSync UpdateMode = "sync"
)

// UpdateModes contains all supported update modes.
var UpdateModes = util.Enum[UpdateMode]{
	string(UpdateModeCreate): UpdateModeCreate,
	string(UpdateModeModify): UpdateModeModify,
	string(UpdateModeSync):   UpdateModeSync,
}

var (
	// ErrInvalidUpdateMode indicates unsupported update mode value.
	ErrInvalidUpdateMode = errors.New("invalid update mode")
)

// ParseUpdateMode parses update mode from configuration.
func ParseUpdateMode(s string) (UpdateMode, error) {
	switch UpdateMode(s) {
	case UpdateModeCreate,
		UpdateModeModify,
		UpdateModeSync:
		return UpdateMode(s), nil
	}

	return "", fmt.Errorf("%w: %s", ErrInvalidUpdateMode, s)
}

func (m UpdateMode) String() string {
	return string(m)
}

// Config contains all runtime configuration for the import tool.
type Config struct {
	// Catalog is the list of fields we can handle.
	Catalog FieldCatalog
	// Scope defines which hosts we can touch.
	Scope field.Fields

	DeleteMode        DeleteMode
	UpdateMode        UpdateMode
	DefaultManageMode DefaultManageMode

	AllowedValues map[ConfigKey][]string

	PrivX struct {
		BaseURL           string
		ClientID          string
		ClientSecret      string
		OAuthClientID     string
		OAuthClientSecret string
		JWT               string
	}

	MaxRecords int
}

// NewConfig returns an initialized Config.
func NewConfig() Config {
	return Config{
		AllowedValues: make(map[ConfigKey][]string),
		Catalog:       NewFieldCatalog(),
	}
}

// Apply merges config string values into config.
func (c *Config) Apply(out output.Output, values ConfigValues) bool {
	success := true
	for key, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		switch key {
		case ConfigKeyBaseURL:
			c.PrivX.BaseURL = value

		case ConfigKeyJWT:
			c.PrivX.JWT = value

		case ConfigKeyClientID:
			c.PrivX.ClientID = value

		case ConfigKeyClientSecret:
			c.PrivX.ClientSecret = value

		case ConfigKeyOAuthClientID:
			c.PrivX.OAuthClientID = value

		case ConfigKeyOAuthClientSecret:
			c.PrivX.OAuthClientSecret = value

		case ConfigKeyMaxRecords:
			var err error
			c.MaxRecords, err = strconv.Atoi(value)
			if err != nil {
				out.Printf(output.VerbosityNormal, "Value %q is not valid for configuration option %s, ignored\n", value, key)
				success = false
			}

		case ConfigKeyDeleteMode:
			switch DeleteMode(value) {
			case DeleteModeDelete, DeleteModeDisable:
				c.DeleteMode = DeleteMode(value)
			default:
				out.Printf(output.VerbosityNormal, "Value %q is not valid for configuration option %s, ignored\n", value, key)
				success = false
			}

		case ConfigKeyManageMode:
			switch DefaultManageMode(value) {
			case DefaulManageModeIgnored, DefaulManageModeManaged:
				c.DefaultManageMode = DefaultManageMode(value)
			default:
				out.Printf(output.VerbosityNormal, "Value %q is not valid for configuration option %s, ignored\n", value, key)
				success = false
			}

		case ConfigKeyUpdateMode:
			mode, err := ParseUpdateMode(value)
			if err != nil {
				out.Printf(
					output.VerbosityNormal,
					"Invalid configuration option %s=%q: %v\n",
					key,
					value,
					err,
				)
				success = false
				continue
			}

			c.UpdateMode = mode

		case ConfigKeyAllowedDBProtocols,
			ConfigKeyAllowedServices:
			c.AllowedValues[key] = util.SplitCommaSeparated(value)

		}
	}
	return success
}

// MaskedMap is a map of elements that should not be printed out.
var MaskedMap = map[string]struct{}{
	"JWT":                       {},
	"ClientSecret":              {},
	"OAuthClientSecret":         {},
	"PRIVX_CLIENT_SECRET":       {},
	"PRIVX_OAUTH_CLIENT_SECRET": {},
	"PRIVX_JWT":                 {},
}

// LoadEnv reads configuration values from environment variables.
func LoadEnv(out output.Output) ConfigValues {
	values := make(ConfigValues)

	for _, key := range AllConfigKeys {
		value := strings.TrimSpace(os.Getenv(key.EnvName()))
		if value == "" {
			continue
		}
		values[key] = value

		_, masked := MaskedMap[key.EnvName()]
		if masked {
			value = util.MaskString(value)
		}
		out.Printf(output.VerbosityInfo, "Configuration value %s from environment variable %s: %q\n", key, key.EnvName(), value)
	}

	return values
}

// configFile defines the possible fields in configuration TOML file.
type configFile struct {
	// Scope is a list of key-value pairs used to restrict which hosts can be modified.
	Scope map[string]string `toml:"scope"`

	DeleteMode         string   `toml:"delete_mode"`
	UpdateMode         string   `toml:"update_mode"`
	ManagementMode     string   `toml:"default_management_mode"`
	AllowedServices    []string `toml:"allowed_services"`
	AllowedDBProtocols []string `toml:"allowed_db_protocols"`
	MaxRecords         string   `toml:"max_records"`
	BaseURL            string   `toml:"base_url"`
}

// Values returns scalar configuration values from the config file.
func (f configFile) Values() ConfigValues {
	values := ConfigValues{}

	add := func(key ConfigKey, value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			values[key] = value
		}
	}

	add(ConfigKeyDeleteMode, f.DeleteMode)
	add(ConfigKeyUpdateMode, f.UpdateMode)
	add(ConfigKeyMaxRecords, f.MaxRecords)
	add(ConfigKeyBaseURL, f.BaseURL)

	if len(f.AllowedServices) > 0 {
		values[ConfigKeyAllowedServices] = strings.Join(f.AllowedServices, ",")
	}

	if len(f.AllowedDBProtocols) > 0 {
		values[ConfigKeyAllowedDBProtocols] = strings.Join(f.AllowedDBProtocols, ",")
	}

	return values
}

// LoadConfig reads runtime configuration from configDir.
func LoadConfig(
	out output.Output,
	configDir string,
	// Current values are used as default
	cfg Config,
) (Config, error) {

	file, err := loadConfigFile(filepath.Join(configDir, "config.toml"))
	if err != nil {
		return cfg, err
	}

	cfg.Catalog, err = LoadFieldCatalog(out, filepath.Join(configDir, "host.toml"))
	if err != nil {
		return cfg, err
	}

	// The scope definitions are fields, so they need catalog for decoding.
	cfg.Scope, err = cfg.Catalog.DecodeStringFields(file.Scope)
	if err != nil {
		return cfg, err
	}

	switch DefaultManageMode(file.ManagementMode) {
	case DefaulManageModeIgnored, DefaulManageModeManaged:
		cfg.DefaultManageMode = DefaultManageMode(file.ManagementMode)
	case "":
	default:
		return cfg, fmt.Errorf(
			"%w: management mode=%q",
			ErrInvalidConfigValue,
			file.ManagementMode,
		)
	}

	switch DeleteMode(file.DeleteMode) {
	case DeleteModeDelete, DeleteModeDisable:
		cfg.DeleteMode = DeleteMode(file.DeleteMode)
	case "":
	default:
		return cfg, fmt.Errorf(
			"%w: delete_mode=%q",
			ErrInvalidConfigValue,
			file.DeleteMode,
		)
	}

	cfg.AllowedValues[ConfigKeyAllowedServices] = file.AllowedServices

	return cfg, nil
}

// LoadConfigFile reads scalar config values and scope from config.toml.
func LoadConfigFile(
	out output.Output,
	configDir string,
	catalog FieldCatalog,
) (ConfigValues, field.Fields, error) {
	file, err := loadConfigFile(filepath.Join(configDir, "config.toml"))
	if err != nil {
		return nil, nil, err
	}

	values := file.Values()

	scope, err := catalog.DecodeStringFields(file.Scope)
	if err != nil {
		return nil, nil, err
	}

	return values, scope, nil
}

// loadConfigFile reads the runtime configuration TOML file.
func loadConfigFile(path string) (*configFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &configFile{}, nil
		}

		return nil, fmt.Errorf("read config: %w", err)
	}

	var file configFile

	if err := toml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	return &file, nil
}

// ResolveConfigDir returns the effective configuration directory.
//
// Priority:
// 1. CLI value
// 2. ./config relative to executable
func ResolveConfigDir(cliValue string) (string, error) {
	if cliValue != "" {
		return cliValue, nil
	}

	exePath, err := os.Executable()
	if err != nil {
		return "", ErrConfigDirNotFound
	}

	exeDir := filepath.Dir(exePath)
	configDir := filepath.Clean(filepath.Join(exeDir, "..", "config"))

	info, err := os.Stat(configDir)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrConfigDirNotFound, configDir)
	}

	return configDir, nil
}
