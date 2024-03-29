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

package cmdtemplate

import (
	"bytes"
	"text/template"

	"github.com/galexrt/ancientt/testers"
)

// Variables variables used for templating
type Variables struct {
	ServerAddressV4 string
	ServerAddressV6 string
	ServerPort      int32
}

// Template template a given cmd and args with the given host information struct
func Template(task *testers.Task, variables Variables) error {
	templatedArgs := []string{}

	var err error
	task.Command, err = templateString(task.Command, variables)
	if err != nil {
		return err
	}

	for _, arg := range task.Args {
		arg, err = templateString(arg, variables)
		if err != nil {
			return err
		}
		templatedArgs = append(templatedArgs, arg)
	}
	task.Args = templatedArgs
	return nil
}

// templateString execute a given template with the variables given
func templateString(in string, variable interface{}) (string, error) {
	t, err := template.New("main").Parse(in)
	if err != nil {
		return "", err
	}

	var out bytes.Buffer
	if err = t.ExecuteTemplate(&out, "main", variable); err != nil {
		return "", err
	}
	return out.String(), err
}
