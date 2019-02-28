// Copyright (C) 2017 Google Inc.
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

package remotessh_test

import (
	"bytes"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device/remotessh"
)

func TestReadConfiguration(t *testing.T) {
	ctx := log.Testing(t)

	input := `
[
	{
		"Name": "name",
		"host": "localhost",
		"port": "22",
		"user": "me",
		"keyPath": "~/.ssh/id_rsa",
		"knownHostsPath": "~/.ssh/known_hosts"
	},
	{
		"Name": "FirstConnection",
		"user": "me",
		"host": "example.com",
		"port": "443",
		"keyPath": "~/.ssh/id_rsa",
		"knownHostsPath": "~/.ssh/known_hosts"
	},
	{
		"Name": "Connection2",
		"keyPath": "id_dsa",
		"knownHostsPath": "someFile",
		"user": "me"
	}
]
`
	reader := bytes.NewReader([]byte(input))
	configs, err := remotessh.ReadConfigurations(reader)

	assert.For(ctx, "err").ThatError(err).Succeeded()

	for i, test := range []remotessh.Configuration{
		remotessh.Configuration{
			Name:       "name",
			Host:       "localhost",
			User:       "me",
			Port:       22,
			Keyfile:    "~/.ssh/id_rsa",
			KnownHosts: "~/.ssh/known_hosts",
		},
		remotessh.Configuration{
			Name:       "FirstConnection",
			Host:       "example.com",
			User:       "me",
			Port:       443,
			Keyfile:    "~/.ssh/id_rsa",
			KnownHosts: "~/.ssh/known_hosts",
		},
		remotessh.Configuration{
			Name:       "Connection2",
			User:       "me",
			Host:       "",
			Port:       22,
			Keyfile:    "id_dsa",
			KnownHosts: "someFile",
		},
	} {
		assert.For(ctx, "configs[%v]", i).That(configs[i]).DeepEquals(test)
	}
}
