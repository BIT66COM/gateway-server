package config

import (
	"errors"
	"net/url"

	"github.com/stellar/go-stellar-base/keypair"
)

type Config struct {
	Port              *int
	Horizon           *string
	ApiKey            string `mapstructure:"api_key"`
	NetworkPassphrase string `mapstructure:"network_passphrase"`
	Assets            []string
	Database          struct {
		Type string
		Url  string
	}
	Accounts *Accounts
	Hooks    *Hooks
}

type Accounts struct {
	AuthorizingSeed    *string `mapstructure:"authorizing_seed"`
	IssuingSeed        *string `mapstructure:"issuing_seed"`
	ReceivingAccountId *string `mapstructure:"receiving_account_id"`
}

type Hooks struct {
	Receive *string
	Error   *string
}

func (c *Config) Validate() (err error) {
	if c.Port == nil {
		err = errors.New("port param is required")
		return
	}

	if c.Horizon == nil {
		err = errors.New("horizon param is required")
		return
	} else {
		_, err = url.Parse(*c.Horizon)
		if err != nil {
			err = errors.New("Cannot parse horizon param")
			return
		}
	}

	if c.NetworkPassphrase == "" {
		err = errors.New("network_passphrase param is required")
		return
	}

	var dbUrl *url.URL
	dbUrl, err = url.Parse(c.Database.Url)
	if err != nil {
		err = errors.New("Cannot parse database.url param")
		return
	}

	switch c.Database.Type {
	case "mysql":
		// Add `parseTime=true` param to mysql url
		query := dbUrl.Query()
		query.Set("parseTime", "true")
		dbUrl.RawQuery = query.Encode()
		c.Database.Url = dbUrl.String()
	case "postgres":
	case "sqlite3":
		break
	default:
		err = errors.New("Invalid database.type param")
		return
	}

	if c.Accounts != nil {
		if c.Accounts.AuthorizingSeed != nil {
			_, err = keypair.Parse(*c.Accounts.AuthorizingSeed)
			if err != nil {
				err = errors.New("accounts.authorizing_seed is invalid")
				return
			}
		}

		if c.Accounts.IssuingSeed != nil {
			_, err = keypair.Parse(*c.Accounts.IssuingSeed)
			if err != nil {
				err = errors.New("accounts.issuing_seed is invalid")
				return
			}
		}

		if c.Accounts.ReceivingAccountId != nil {
			_, err = keypair.Parse(*c.Accounts.ReceivingAccountId)
			if err != nil {
				err = errors.New("accounts.receiving_account_id is invalid")
				return
			}
		}
	}

	if c.Hooks != nil {
		if c.Hooks.Receive != nil {
			_, err = url.Parse(*c.Hooks.Receive)
			if err != nil {
				err = errors.New("Cannot parse hooks.receive param")
				return
			}
		}

		if c.Hooks.Error != nil {
			_, err = url.Parse(*c.Hooks.Error)
			if err != nil {
				err = errors.New("Cannot parse hooks.error param")
				return
			}
		}
	}

	return
}
