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

package kubernetes

import (
	"testing"

	"github.com/creasty/defaults"
	"github.com/galexrt/ancientt/pkg/config"
	"github.com/galexrt/ancientt/pkg/util"
	"github.com/galexrt/ancientt/tests/k8s"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TODO add tests

func TestGetHostsForTest(t *testing.T) {
	// TODO
	clientset, err := k8s.NewClient(3)
	require.Nil(t, err)
	require.NotNil(t, clientset)

	conf := &config.RunnerKubernetes{}
	// Set defaults in the config struct
	require.Nil(t, defaults.Set(conf))

	runner := &Kubernetes{
		logger:    zap.NewNop().With(zap.String("runner", Name), zap.String("namespace", "")),
		config:    conf,
		k8sclient: clientset,
	}

	test := &config.Test{}
	hosts, err := runner.GetHostsForTest(test)
	require.Nil(t, err)
	assert.Equal(t, 0, len(hosts.Servers))
	assert.Equal(t, 0, len(hosts.Clients))

	test.Hosts.Servers = append(test.Hosts.Servers, config.Hosts{All: util.BoolTruePointer()})
	test.Hosts.Clients = append(test.Hosts.Clients, config.Hosts{All: util.BoolTruePointer()})
	hosts, err = runner.GetHostsForTest(test)
	require.Nil(t, err)
	assert.Equal(t, 3, len(hosts.Servers))
	assert.Equal(t, 3, len(hosts.Clients))

	test.Hosts.Servers[0] = config.Hosts{Count: 1, Random: util.BoolTruePointer()}
	hosts, err = runner.GetHostsForTest(test)
	require.Nil(t, err)
	assert.Equal(t, 1, len(hosts.Servers))
	assert.Equal(t, 3, len(hosts.Clients))
}
