package main

import (
	"errors"
	"fmt"
	"strconv"

	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
)

// AppConfig holds our info for configuring the service
type AppConfig struct {
	Confbase      string
	AppName       string
	ConsulAddress string
	client        *consul.Client
	ServiceChecks consul.AgentServiceChecks
}

// Connect will connect the service to Consul
func (s *AppConfig) Connect() error {
	var err error
	cc := consul.DefaultConfig()
	cc.Address = s.ConsulAddress
	s.client, err = consul.NewClient(cc)
	if err != nil {
		log.Error("unable to open consul")
		return err
	}
	log.Debug("connected to consul")
	return nil
}

// NewAppConfig returns an AppConfig from the given parameters
func NewAppConfig(name string) *AppConfig {
	sc := DefaultAppConfig(name)
	sc.Connect()
	return sc
}

// DefaultConfig needs a name and port, and will return a built AppConfig
// for localhost consul agent
func DefaultAppConfig(n string) *AppConfig {
	return &AppConfig{
		Confbase:      "appconfig/" + n + "/",
		AppName:       n,
		ConsulAddress: "localhost:8500",
	}
}

// GetBase returns the base URL for configurations
func (s *AppConfig) GetBase() string {
	return s.Confbase
}

// SetBase sets the base URL for configurations
func (s *AppConfig) SetBase(b string) {
	s.Confbase = b
}

// SetValue updates/sets the specified key and value in Consul
func (s *AppConfig) SetValue(k, v string) error {
	kurl := s.Confbase + k
	kv := s.client.KV()
	pair := &consul.KVPair{Key: kurl, Value: []byte(v)}
	_, err := kv.Put(pair, nil)
	if err != nil {
		log.Print("in SetValue")
		log.Print(err)
		log.Fatal(errors.New("unable to communicate with backing store"))
	}
	return err
}

// GetString returns the value in consul for the given key name
func (s *AppConfig) GetString(k string, bail bool) (string, error) {
	kurl := s.Confbase + k
	kv := s.client.KV()
	pair, _, err := kv.Get(kurl, nil)
	if err != nil {
		if bail {
			log.Fatal(errors.New("unable to communicate with backing store"))
		}
		return "", err
	}
	if pair == nil {
		return "", fmt.Errorf("key '%s' not found", kurl)
	}
	return string(pair.Value), err
}

// GetInteger returns the value in consul for the given key name
func (s *AppConfig) GetInteger(k string, bail bool) (int, error) {
	kurl := s.Confbase + k
	kv := s.client.KV()
	pair, _, err := kv.Get(kurl, nil)
	if err != nil {
		if bail {
			log.Fatal(errors.New("unable to communicate with backing store"))
		}
		log.Print(err)
		return 0, err
	}
	if pair == nil {
		return 0, errors.New("key not found")
	}
	return strconv.Atoi(string(pair.Value))
}

// GetBool returns the value in consul for the given key name
func (s *AppConfig) GetBool(k string, bail bool) (bool, error) {
	kurl := s.Confbase + k
	kv := s.client.KV()
	pair, _, err := kv.Get(kurl, nil)
	if err != nil {
		if bail {
			log.Fatal(errors.New("unable to communicate with backing store"))
		}
		return false, err
	}
	if pair == nil {
		return false, errors.New("key not found")
	}
	switch string(pair.Value) {
	case "true", "True":
		return true, nil
	}
	return false, nil
}
