package rsmtp

import (
	"os"
	"gopkg.in/yaml.v2"
)

type AuthNamePassword struct {
	User, Password string
}

type Config struct {
	Nsqd *NsqdConfig
	RemoteAcconts []*RemoteAccont
	remoteAcconts map[string]*RemoteAccont
	Users []*User
	users map[string]*User
}

func LoadConfig(pth string) (*Config, error) {
	f, err := os.Open(pth)
	if err != nil {
		return nil, err
	}
	d := yaml.NewDecoder(f)
	var cfg *Config
	err = d.Decode(&cfg)
	if err != nil {
		return nil, err
	}

	cfg.remoteAcconts = make(map[string]*RemoteAccont)

	for _, ra := range cfg.RemoteAcconts {
		if ra.ID == "" {
			ra.ID = ra.User
		}
		cfg.remoteAcconts[ra.ID] = ra
	}

	cfg.users = make(map[string]*User)

	for _, usr := range cfg.Users {
		cfg.users[usr.User] = usr
	}

	return cfg, nil
}

type RemoteAccont struct {
	ID string
	AuthNamePassword
}