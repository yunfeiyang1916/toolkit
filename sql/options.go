package sql

type SQLGroupConfig struct {
	Name      string   `toml:"name"`
	Master    string   `toml:"master"`
	Slaves    []string `toml:"slaves"`
	StatLevel string   `toml:"stat_level"`
	LogFormat string   `toml:"log_format"`
}
