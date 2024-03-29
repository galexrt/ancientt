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

// Package executor executor heavily inspired from the Executor from github.com/rook/rook/pkg/util/exec pkg
package executor

import (
	"context"
	"os"
	"os/exec"
	"syscall"

	"go.uber.org/zap"
)

// Executor heavily inspired from the Executor from github.com/rook/rook/pkg/util/exec pkg
type Executor interface {
	ExecuteCommand(ctx context.Context, actionName string, command string, arg ...string) error
	ExecuteCommandWithOutput(ctx context.Context, actionName string, command string, arg ...string) (string, error)
	ExecuteCommandWithOutputByte(ctx context.Context, actionName string, command string, arg ...string) ([]byte, error)
	SetEnv([]string)
}

// CommandExecutor Executor implementation
type CommandExecutor struct {
	Executor
	logger *zap.Logger
	env    []string
}

// NewCommandExecutor create and return a new CommandExecutor
func NewCommandExecutor(logger *zap.Logger, pkg string) Executor {
	return CommandExecutor{
		logger: logger.With(zap.String("executor", pkg)),
	}
}

// ExecuteCommand execute a given command with its arguments but don't return any output
func (ce CommandExecutor) ExecuteCommand(ctx context.Context, actionName string, command string, arg ...string) error {
	cmd := exec.CommandContext(ctx, command, arg...)
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
		Setpgid:   true,
	}

	ce.logger.Info("executing command", zap.String("command", command), zap.Strings("args", arg))

	if err := cmd.Start(); err != nil {
		return err
	}

	return cmd.Wait()
}

// ExecuteCommandWithOutput execute a given command with its arguments and return the output as a string
func (ce CommandExecutor) ExecuteCommandWithOutput(ctx context.Context, actionName string, command string, arg ...string) (string, error) {
	out, err := ce.ExecuteCommandWithOutputByte(ctx, actionName, command, arg...)
	return string(out), err
}

// ExecuteCommandWithOutputByte execute a given command with its arguments and return the output as a byte array ([]byte)
func (ce CommandExecutor) ExecuteCommandWithOutputByte(ctx context.Context, actionName string, command string, arg ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, command, arg...)
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
		Setpgid:   true,
	}

	ce.logger.Info("executing command", zap.String("command", command), zap.Strings("args", arg))

	out, err := cmd.CombinedOutput()
	ce.logger.Debug(string(out), zap.String("action", actionName))

	if err != nil {
		return out, err
	}

	return out, nil
}

// SetEnv set env for command execution
func (ce CommandExecutor) SetEnv(e []string) {
	ce.env = e
}
