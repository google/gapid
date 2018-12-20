// Copyright (C) 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package remotessh

import (
	"encoding/json"
	"io"
	"os/user"
)

// Configuration represents a configuration for connecting
// to an SSH remote client.
// The SSH agent is first used to attempt connection,
// followed by the given Keyfile
type Configuration struct {
	// The name to use for this connection
	Name string
	// The hostname to connect to
	Host string `json:"host"`
	// User is the username to use for login
	User string `json:"user"`
	// Which port should be used
	Port uint16 `json:"port,string"`
	// The pem encoded public key file to use for the connection.
	// If not specified uses ~/.ssh/id_rsa
	Keyfile string `json:"keyPath"`
	// The known_hosts file to use for authentication. Defaults to
	// ~/.ssh/known_hosts
	KnownHosts string `json:"knownHostsPath"`
	// Environment variables to set on the connection
	Env []string
}

// ReadConfigurations reads a set of configurations from then
// given reader, and returns the configurations to the user.
func ReadConfigurations(r io.Reader) ([]Configuration, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}

	cfgs := []Configuration{}
	d := json.NewDecoder(r)
	if _, err := d.Token(); err != nil {
		return nil, err
	}
	for d.More() {
		cfg := Configuration{
			Name:       "",
			Host:       "",
			User:       u.Username,
			Port:       22,
			Keyfile:    u.HomeDir + "/.ssh/id_rsa",
			KnownHosts: u.HomeDir + "/.ssh/known_hosts",
		}
		if err := d.Decode(&cfg); err != nil {
			return nil, err
		}
		cfgs = append(cfgs, cfg)
	}
	if _, err := d.Token(); err != nil {
		return nil, err
	}
	return cfgs, nil
}
