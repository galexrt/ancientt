/*
Copyright 2019 Cloudical Deutschland GmbH
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

package config

// Test
type Test struct {
	Type       string     `yaml:"type"`
	RunOptions RunOptions `yaml:"runOptions"`
	Hosts      TestHosts  `yaml:"hosts"`
	IPerf      *IPerf     `yaml:"iperf"`
	IPerf3     *IPerf3    `yaml:"iperf3"`
	Siege      *Siege     `yaml:"siege"`
}

const (
	// RunModeSequential run tasks in sequential / serial order
	RunModeSequential = "sequential"
	// RunModeParallel run tasks in parallel (WARNING! Be sure what you cause with this, e.g., 100 iperfs might not be good for a production environment)
	RunModeParallel = "parallel"
)

// RunOptions options for running the tasks
type RunOptions struct {
	Rounds        int    `yaml:"rounds"`
	Interval      string `yaml:"interval"`
	Mode          string `yaml:"mode"`
	ParallelCount int    `yaml:"parallelCount"`
}

// TestHosts list of clients and servers hosts for use in the test(s)
type TestHosts struct {
	Clients []Hosts `yaml:"clients"`
	Servers []Hosts `yaml:"servers"`
}

// IPerf
type IPerf struct {
	WindowSizeCalculation IPerfWindowSizeCalculation `yaml:"windowSizeCalculation"`
	AdditionalFlags       IPerfAdditionalFlags       `yaml:"additionalFlags"`
	UDP                   bool                       `yaml:"udp"`
}

// IPerfWindowSizeCalculation
type IPerfWindowSizeCalculation struct {
	Auto bool `yaml:"auto"`
}

// IPerfAdditionalFlags
type IPerfAdditionalFlags struct {
	Client []string `yaml:"client"`
	Server []string `yaml:"server"`
}

// IPerf3
type IPerf3 struct {
	WindowSizeCalculation IPerfWindowSizeCalculation `yaml:"windowSizeCalculation"`
	AdditionalFlags       IPerfAdditionalFlags       `yaml:"additionalFlags"`
	UDP                   *bool                      `yaml:"udp"`
}

// Siege
type Siege struct {
}
