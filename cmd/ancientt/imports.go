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

package main

// This file contains the imports for each output, parser, runner and tester.
// Importing them from the, e.g., `outputs` pkg would cause a import cycle.

import (
	// Outputs
	_ "github.com/galexrt/ancientt/outputs/csv"
	_ "github.com/galexrt/ancientt/outputs/dump"
	_ "github.com/galexrt/ancientt/outputs/excelize"
	_ "github.com/galexrt/ancientt/outputs/gochart"
	_ "github.com/galexrt/ancientt/outputs/mysql"
	_ "github.com/galexrt/ancientt/outputs/sqlite"

	// Parsers
	_ "github.com/galexrt/ancientt/parsers/iperf3"
	_ "github.com/galexrt/ancientt/parsers/pingparsing"

	// Runners
	_ "github.com/galexrt/ancientt/runners/ansible"
	_ "github.com/galexrt/ancientt/runners/kubernetes"
	_ "github.com/galexrt/ancientt/runners/mock"

	// Testers
	_ "github.com/galexrt/ancientt/testers/iperf3"
	_ "github.com/galexrt/ancientt/testers/pingparsing"
)
