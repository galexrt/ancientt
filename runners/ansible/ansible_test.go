/*
Copyright 2019 Cloudical Deutschland GmbH. All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ansible

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/galexrt/ancientt/pkg/config"
	exectest "github.com/galexrt/ancientt/pkg/executor/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetHostsForTest(t *testing.T) {
	var lock sync.Mutex
	run := 1

	logConfig := zap.NewProductionConfig()
	logConfig.Level.SetLevel(zap.DebugLevel)
	logger, err := logConfig.Build()
	assert.NoError(t, err)

	mockexec := exectest.MockExecutor{
		Logger: logger,
		MockExecuteCommandWithOutputByte: func(ctx context.Context, actionName string, command string, arg ...string) ([]byte, error) {
			lock.Lock()
			defer func() {
				run++
				lock.Unlock()
			}()

			switch run {
			case 1:
				return []byte(`{
    "_meta": {
        "hostvars": {}
    },
    "all": {
        "children": [
            "clients",
            "server",
            "ungrouped"
        ]
    },
    "clients": {
        "hosts": [
            "server1",
            "server2"
        ]
    },
    "server": {
        "hosts": [
            "server4"
		]
    }
}`), nil
			case 2, 3, 4:
				return []byte(fmt.Sprintf(`192.0.2.5 | SUCCESS => {
	"ansible_facts": {
		"ansible_default_ipv4": {
			"address": "192.0.2.1%d"
		},
		"ansible_default_ipv6": {
			"address": "2001:DB8::%d337"
		}
	}
}`, run, run)), nil
			default:
				err := fmt.Errorf("no command for run %d (actionName: %s; cmd: %s; args: %s", run, actionName, command, arg)
				t.Fatal(err)
				return nil, err
			}
		},
	}

	conf := &config.RunnerAnsible{
		InventoryFilePath: "/tmp/test-ancientt-ansible-inventory",
	}
	conf.SetDefaults()
	a := Ansible{
		logger:   zap.NewNop().With(zap.String("runner", Name)),
		config:   conf,
		executor: mockexec,
	}
	require.NotNil(t, a)

	hosts, err := a.GetHostsForTest(&config.Test{})
	require.Nil(t, err)

	assert.Equal(t, 2, len(hosts.Clients))
	assert.Equal(t, 1, len(hosts.Servers))
}
