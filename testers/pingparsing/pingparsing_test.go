/*
Copyright 2020 Cloudical Deutschland GmbH. All rights reserved.
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

package pingparsing

import (
	"testing"

	"github.com/galexrt/ancientt/pkg/config"
	"github.com/galexrt/ancientt/testers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestIPerf3Plan(t *testing.T) {
	tester, err := NewPingParsingTester(zap.NewNop(), nil, nil)
	assert.Nil(t, err)
	assert.NotNil(t, tester)

	env := &testers.Environment{
		Hosts: &testers.Hosts{
			Clients: map[string]*testers.Host{},
			Servers: map[string]*testers.Host{},
		},
	}
	test := &config.Test{
		Type: "pingparsing",
	}

	plan, err := tester.Plan(env, test)
	assert.Nil(t, err)
	require.NotNil(t, plan)
	assert.Equal(t, "pingparsing", plan.Tester)
	assert.Equal(t, 0, len(plan.AffectedServers))
	assert.Equal(t, 0, len(plan.Commands))
}
