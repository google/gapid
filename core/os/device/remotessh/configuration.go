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
	"io/ioutil"
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
	Host string
	// User is the username to use for login
	User string
	// Which port should be used
	Port int16
	// The pem encoded public key file to use for the connection.
	// If not specified uses ~/.ssh/id_rsa
	Keyfile string
	// The known_hosts file to use for authentication. Defaults to
	// ~/.ssh/known_hosts
	KnownHosts string
}

func (c *Configuration) UnmarshalJSON(data []byte) error {
	type configAlias Configuration
	u, err := user.Current()
	if err != nil {
		return err
	}
	newC := &configAlias{
		Name:       "",
		Host:       "",
		User:       u.Username,
		Port:       22,
		Keyfile:    u.HomeDir + "/.ssh/id_rsa",
		KnownHosts: u.HomeDir + "/.ssh/known_hosts",
	}
	if err := json.Unmarshal(data, newC); err != nil {
		return err
	}

	*c = Configuration(*newC)
	return nil
}

func ReadConfiguration(r io.Reader) ([]Configuration, error) {
	cfg := []Configuration{}

	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(bytes, &cfg)
	return cfg, err
}
