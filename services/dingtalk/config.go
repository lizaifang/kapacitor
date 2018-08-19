package dingtalk

import (
	"github.com/pkg/errors"
)

type Config struct {
	Enabled     bool   `toml:"enabled" override:"enabled"`
	AccessToken string `toml:"access_token" override:"access_token,redact"`
}

func NewConfig() Config {
	return Config{
		Enabled: true,
	}
}

func (c Config) Validate() error {
	if c.Enabled {
		if c.AccessToken == "" {
			return errors.New("must specify token")
		}
	}
	return nil
}
