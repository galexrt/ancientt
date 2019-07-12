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

package outputs

import (
	"bytes"
	"fmt"

	"github.com/cloudical-io/acntt/pkg/config"
	chart "github.com/wcharczuk/go-chart"
)

// NameGoChart GoChart output name
const NameGoChart = "gochart"

func init() {
	Factories[NameGoChart] = NewGoChartOutput
}

// GoChart GoChart tester structure
type GoChart struct {
	Output
	config *config.GoChart
}

// NewGoChartOutput return a new GoChart tester instance
func NewGoChartOutput(cfg *config.Config, outCfg *config.Output) (Output, error) {
	goChart := GoChart{
		config: outCfg.GoChart,
	}
	if goChart.config.NamePattern != "" {
		goChart.config.NamePattern = "{{ .UnixTime }}-{{ .Data.Tester }}-{{ .Data.ServerHost }}_{{ .Data.ClientHost }}.goChart"
	}
	return goChart, nil
}

// Do make GoChart charts
func (ip GoChart) Do(data Data) error {
	return fmt.Errorf("gochart not implemented yet")
	dataTable, ok := data.Data.(Table)
	if !ok {
		return fmt.Errorf("data not in table for csv output")
	}

	filename, err := getFilenameFromPattern(ip.config.NamePattern, data, nil)
	if err != nil {
		return err
	}

	_ = filename

	graph := chart.Chart{
		Series: []chart.Series{
			chart.ContinuousSeries{
				XValues: []float64{1.0, 2.0, 3.0, 4.0},
				YValues: []float64{1.0, 2.0, 3.0, 4.0},
			},
		},
	}

	buffer := bytes.NewBuffer([]byte{})
	if err := graph.Render(chart.PNG, buffer); err != nil {
		return err
	}

	// Iterate over header columns
	for _, column := range dataTable.Headers {
		rowCells := []string{}
		for _, row := range column.Rows {
			rowCells = append(rowCells, fmt.Sprintf("%v", row.Value))
		}
		if len(rowCells) == 0 {
			continue
		}

	}

	// Iterate over data columns
	for _, column := range dataTable.Columns {
		rowCells := []string{}
		for _, row := range column.Rows {
			rowCells = append(rowCells, fmt.Sprintf("%v", row.Value))
		}
		if len(rowCells) == 0 {
			continue
		}

	}

	return nil
}
