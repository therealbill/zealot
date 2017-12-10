package main

import (
	"errors"
	"fmt"
	"strconv"

	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
)

var (
	lock *consul.Lock
)

// JobConfig holds our info for configuring the service
type JobConfig struct {
	Confbase      string
	Name          string
	JobType       string
	ConsulAddress string
	client        *consul.Client
	ServiceChecks consul.AgentServiceChecks
}

// Connect will connect the service to Consul
func (s *JobConfig) Connect() error {
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

// NewJobConfig returns an JobConfig from the given parameters
func NewJobConfig(name string, jobtype string) *JobConfig {
	sc := DefaultConfig(name, jobtype)
	sc.Connect()
	return sc
}

// DefaultConfig needs a name and port, and will return a built JobConfig
// for localhost consul agent
func DefaultConfig(n, j string) *JobConfig {
	return &JobConfig{
		Confbase:      "jobconfig/" + "zealot/" + n + "/",
		Name:          n,
		JobType:       j,
		ConsulAddress: "localhost:8500",
	}
}

// GetBase returns the base URL for configurations
func (s *JobConfig) GetBase() string {
	return s.Confbase
}

// AddCheck adds a consul.AgentServiceCheck
// will only be effective if issued prior to RegisterService
func (s *JobConfig) AddCheck(c consul.AgentServiceCheck) {
	s.ServiceChecks = append(s.ServiceChecks, &c)
}

// SetBase sets the base URL for configurations
func (s *JobConfig) SetBase(b string) {
	s.Confbase = b
}

// SetValue updates/sets the specified key and value in Consul
func (s *JobConfig) SetValue(k, v string) error {
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
func (s *JobConfig) GetString(k string, bail bool) (string, error) {
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
func (s *JobConfig) GetInteger(k string, bail bool) (int, error) {
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
func (s *JobConfig) GetBool(k string, bail bool) (bool, error) {
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
