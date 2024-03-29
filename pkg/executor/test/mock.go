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

// Package test mock executor heavily inspired from the Executor from github.com/rook/rook/pkg/util/exec pkg#
package test

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/galexrt/ancientt/pkg/executor"
	"go.uber.org/zap"
)

// MockExecutor Mock Executor implementation for tests
type MockExecutor struct {
	executor.Executor
	Logger *zap.Logger

	MockExecuteCommand               func(ctx context.Context, actionName string, command string, arg ...string) error
	MockExecuteCommandWithOutput     func(ctx context.Context, actionName string, command string, arg ...string) (string, error)
	MockExecuteCommandWithOutputByte func(ctx context.Context, actionName string, command string, arg ...string) ([]byte, error)
	MockSetEnv                       func(e []string)

	env []string
}

// ExecuteCommand execute a given command with its arguments but don't return any output
func (ce MockExecutor) ExecuteCommand(ctx context.Context, actionName string, command string, arg ...string) error {
	if ce.MockExecuteCommand != nil {
		return ce.MockExecuteCommand(ctx, actionName, command, arg...)
	}

	cmd := exec.CommandContext(ctx, command, arg...)

	ce.Logger.With(zap.String("command", command), zap.Strings("args", arg)).Info("executing")

	return cmd.Run()
}

// ExecuteCommandWithOutput execute a given command with its arguments and return the output as a string
func (ce MockExecutor) ExecuteCommandWithOutput(ctx context.Context, actionName string, command string, arg ...string) (string, error) {
	if ce.MockExecuteCommandWithOutput != nil {
		return ce.MockExecuteCommandWithOutput(ctx, actionName, command, arg...)
	}

	out, err := ce.ExecuteCommandWithOutputByte(ctx, actionName, command, arg...)
	return string(out), err
}

// ExecuteCommandWithOutputByte execute a given command with its arguments and return the output as a byte array ([]byte)
func (ce MockExecutor) ExecuteCommandWithOutputByte(ctx context.Context, actionName string, command string, arg ...string) ([]byte, error) {
	if ce.MockExecuteCommandWithOutputByte != nil {
		out, err := ce.MockExecuteCommandWithOutputByte(ctx, actionName, command, arg...)
		ce.Logger.Debug(string(out), zap.String("action", actionName))
		return out, err
	}

	cmd := exec.CommandContext(ctx, command, arg...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	ce.Logger.Debug(string(out), zap.String("action", actionName))

	return out, nil
}

// SetEnv set env for command execution
func (ce MockExecutor) SetEnv(e []string) {
	ce.Logger.Debug(fmt.Sprintf("%+v", e), zap.String("action", "setEnv()"))
	if ce.MockSetEnv != nil {
		ce.MockSetEnv(e)
		return
	}

	ce.env = e
}
