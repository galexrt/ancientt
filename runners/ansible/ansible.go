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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/galexrt/ancientt/parsers"
	"github.com/galexrt/ancientt/pkg/ansible"
	"github.com/galexrt/ancientt/pkg/cmdtemplate"
	"github.com/galexrt/ancientt/pkg/config"
	"github.com/galexrt/ancientt/pkg/executor"
	"github.com/galexrt/ancientt/pkg/util"
	"github.com/galexrt/ancientt/runners"
	"github.com/galexrt/ancientt/testers"
	"go.uber.org/zap"
)

const (
	// Name Ansible Runner Name
	Name = "ansible"
)

var (
	jsonHeadCleanRegex = regexp.MustCompile(`(?sm)^.*(=> \{| >>$\n\{)`)
	jsonTailCleanRegex = regexp.MustCompile(`(?sm)(^\}.*)`)
)

func init() {
	runners.Factories[Name] = NewRunner
}

// Ansible Ansible runner struct
type Ansible struct {
	runners.Runner
	logger         *zap.Logger
	config         *config.RunnerAnsible
	runOptions     config.RunOptions
	executor       executor.Executor
	additionalInfo string
}

// NewRunner return a new Ansible Runner
func NewRunner(logger *zap.Logger, cfg *config.Config) (runners.Runner, error) {
	conf := cfg.Runner.Ansible

	return &Ansible{
		logger:   logger.With(zap.String("runner", Name), zap.String("inventoryfile", cfg.Runner.Ansible.InventoryFilePath)),
		config:   conf,
		executor: executor.NewCommandExecutor(logger, "runner:ansible"),
	}, nil
}

// GetHostsForTest return a mocked list of hots for the given test config
func (a *Ansible) GetHostsForTest(test *config.Test) (*testers.Hosts, error) {
	ctx, cancel := context.WithTimeout(context.Background(), a.config.Timeouts.CommandTimeout)
	defer cancel()

	out, err := a.executor.ExecuteCommandWithOutputByte(ctx, "runner:ansible: list hosts from inventory", a.config.AnsibleInventoryCommand, []string{
		fmt.Sprintf("--inventory=%s", a.config.InventoryFilePath),
		"--list",
	}...)
	if err != nil {
		return nil, err
	}

	inv, err := ansible.Parse(cleanAnsibleOutput(out))
	if err != nil {
		return nil, err
	}

	servers := inv.GetHostsForGroup(a.config.Groups.Server)
	clients := inv.GetHostsForGroup(a.config.Groups.Clients)

	hosts := map[string]*testers.Host{}

	var lock sync.Mutex
	var wg sync.WaitGroup
	inCh := make(chan string)
	retCh := make(chan error)

	cmdCtx, cmdCancel := context.WithCancel(context.Background())
	defer cmdCancel()

	// Spanw requested amount of workers
	for i := 0; i < *a.config.ParallelHostFactCalls; i++ {
		wg.Add(1)
		go func(in chan string, retCh chan error) {
			defer wg.Done()
			for host := range in {
				// Create a new timeout context per command run
				addresses, err := a.getHostNetworkAddress(cmdCtx, host)
				if err != nil {
					retCh <- err
					return
				}

				lock.Lock()
				hosts[host] = &testers.Host{
					Name:      host,
					Labels:    map[string]string{},
					Addresses: addresses,
				}
				lock.Unlock()
				retCh <- nil
			}
			retCh <- nil
		}(inCh, retCh)
	}

	go func() {
		// Run retrieval of Ansible host facts in parallel
		uniqHosts := util.UniqueStringSlice(servers, clients)
		for _, h := range uniqHosts {
			inCh <- h
		}
		close(inCh)
		// wait for "callers" to end, before closing the return channel
		wg.Wait()
		close(retCh)
	}()

	var errs []string
	for err := range retCh {
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	// If we encountered errors return errors
	if len(errs) > 0 {
		retErr := fmt.Errorf("errors in retrieving ansible host facts. %+v", strings.Join(errs, " "))
		return nil, retErr
	}

	sHosts, err := getHosts(servers, hosts)
	if err != nil {
		return nil, err
	}
	cHosts, err := getHosts(clients, hosts)
	if err != nil {
		return nil, err
	}

	return &testers.Hosts{
		Servers: sHosts,
		Clients: cHosts,
	}, nil
}

func getHosts(in []string, list map[string]*testers.Host) (map[string]*testers.Host, error) {
	hosts := map[string]*testers.Host{}

	for _, h := range in {
		if v, ok := list[h]; ok {
			hosts[h] = v
		} else {
			return nil, fmt.Errorf("server %q not found in ansible hosts list, this should not have happened", h)
		}
	}

	return hosts, nil
}

/*
ansible_default_ipv4.interface and ansible_default_ipv6.interface
```

	{
	    "ansible_facts": {
	    [...]
	    "ansible_default_ipv4": {
	        "address": "172.16.5.100",
	        [...]
	    },
	    "ansible_default_ipv6": {
	        "address": "2a02:8071:22c8:c486:ea5f:3fc7:5039:8b74",
	        [...]
	    },

[...]
}
```
*/
type facts struct {
	AnsibleFacts networkInterface `json:"ansible_facts"`
}

type networkInterface struct {
	AnsibleDefaultIPv4 networkInterfaceAddress `json:"ansible_default_ipv4"`
	AnsibleDefaultIPv6 networkInterfaceAddress `json:"ansible_default_ipv6"`
}

type networkInterfaceAddress struct {
	Address string `json:"address"`
}

func (a *Ansible) getHostNetworkAddress(bctx context.Context, host string) (*testers.IPAddresses, error) {
	a.logger.Debug("retrieving ansible host facts", zap.String("hostname", host))

	ctx, cancel := context.WithTimeout(bctx, a.config.Timeouts.CommandTimeout)
	defer cancel()

	out, err := a.executor.ExecuteCommandWithOutputByte(ctx, "runner:ansible: list hosts from inventory", a.config.AnsibleCommand, []string{
		fmt.Sprintf("--inventory=%s", a.config.InventoryFilePath),
		host,
		"--module-name=setup",
		"--args=gather_subset=!all,!any,network",
	}...)
	if err != nil {
		return nil, err
	}

	out = cleanAnsibleOutput(out)

	facts := &facts{}
	if err := json.Unmarshal(out, facts); err != nil {
		return nil, err
	}

	addresses := &testers.IPAddresses{}

	if facts.AnsibleFacts.AnsibleDefaultIPv4.Address != "" {
		addresses.IPv4 = []string{
			facts.AnsibleFacts.AnsibleDefaultIPv4.Address,
		}
	}
	if facts.AnsibleFacts.AnsibleDefaultIPv6.Address != "" {
		addresses.IPv6 = []string{
			facts.AnsibleFacts.AnsibleDefaultIPv6.Address,
		}
	}

	if facts.AnsibleFacts.AnsibleDefaultIPv4.Address == "" && facts.AnsibleFacts.AnsibleDefaultIPv6.Address == "" {
		return nil, fmt.Errorf("no default IP addresses for ansible host %s", host)
	}

	a.logger.Debug("retrieved ansible host facts", zap.String("hostname", host))

	return addresses, nil
}

// Prepare prepare Ansible runner for usage, though right now there isn't really anything in need of preparations
func (a *Ansible) Prepare(runOpts config.RunOptions, plan *testers.Plan) error {
	a.runOptions = runOpts

	ctx, cancel := context.WithTimeout(context.Background(), a.config.Timeouts.CommandTimeout)
	defer cancel()

	out, err := a.executor.ExecuteCommandWithOutput(ctx, "runner:ansible: get ansible version", a.config.AnsibleCommand, "--version")
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ansible ") {
			a.additionalInfo = line
			break
		}
	}

	return nil
}

// Execute run the given commands and return the logs of it and / or error
func (a *Ansible) Execute(plan *testers.Plan, parser chan<- parsers.Input) error {
	for round, tasks := range plan.Commands {
		a.logger.Info(fmt.Sprintf("running commands round %d of %d", round+1, len(plan.Commands)))
		for i, task := range tasks {
			if task.Sleep != 0 {
				a.logger.Info(fmt.Sprintf("waiting %s to pass before continuing next round", task.Sleep.String()))
				time.Sleep(task.Sleep)
				continue
			}
			a.logger.Info(fmt.Sprintf("running task round %d of %d", i+1, len(tasks)))

			if err := a.runTasks(round, task, plan.TestStartTime, plan.Tester, util.GetTaskName(plan.Tester, plan.TestStartTime), parser); err != nil {
				if !*plan.RunOptions.ContinueOnError {
					return err
				}
				a.logger.Warn("continuing after err", zap.Error(err))
			}
		}
	}

	return nil
}

func (a *Ansible) runTasks(round int, mainTask *testers.Task, plannedTime time.Time, tester string, taskName string, parser chan<- parsers.Input) error {
	logger := a.logger.With(zap.Int("round", round))

	// Create initial cmdtemplate.Variables
	templateVars := cmdtemplate.Variables{
		ServerPort: 5601,
	}
	if len(mainTask.Host.Addresses.IPv4) > 0 {
		templateVars.ServerAddressV4 = mainTask.Host.Addresses.IPv4[0]
	}
	if len(mainTask.Host.Addresses.IPv6) > 0 {
		templateVars.ServerAddressV6 = mainTask.Host.Addresses.IPv6[0]
	}

	if err := cmdtemplate.Template(mainTask, templateVars); err != nil {
		logger.Error("failed to template main task command and / or args", zap.Error(err))
		mainTask.Status.AddFailedServer(mainTask.Host, err)
		return err
	}

	var mainWG sync.WaitGroup
	var wg sync.WaitGroup

	mainTaskStopped := false
	mainCtx, mainCancel := context.WithCancel(context.Background())
	defer mainCancel()

	mainWG.Add(1)
	go func() {
		defer mainWG.Done()
		err := a.executor.ExecuteCommand(mainCtx, "runner:ansible: run main task command", a.config.AnsibleCommand, []string{
			fmt.Sprintf("--inventory=%s", a.config.InventoryFilePath),
			mainTask.Host.Name,
			"--module-name=shell",
			fmt.Sprintf("--args=%s %s", mainTask.Command, strings.Join(mainTask.Args, " ")),
		}...)
		if err != nil {
			if exiterr, ok := err.(*exec.ExitError); ok {
				fmt.Printf("EXITERR: %+v - %+v - %+v\n", exiterr, exiterr.Pid(), exiterr.ProcessState)

				if err := syscall.Kill(-exiterr.Pid(), syscall.SIGKILL); err != nil {
					logger.Error("failed to kill", zap.String("hostname", mainTask.Host.Name), zap.Error(err))
				}
			}
			// Ignore any error after the main task is stopped
			if mainTaskStopped {
				logger.Debug("ignored error after main task was stopped", zap.Error(err))
				return
			}

			logger.Error("error during main task run", zap.Error(err))
			mainTask.Status.AddFailedServer(mainTask.Host, err)
			return
		}
	}()

	time.Sleep(250 * time.Millisecond)

	ready := false
	checkCtx, checkCancel := context.WithTimeout(context.Background(), a.config.Timeouts.TaskCommandTimeout)
	defer checkCancel()

	tries := *a.config.CommandRetries
	for i := 0; i <= tries; i++ {
		err := a.executor.ExecuteCommand(checkCtx, fmt.Sprintf("runner:ansible: check if main task is running (try: %d/%d)", i, tries), a.config.AnsibleCommand, []string{
			fmt.Sprintf("--inventory=%s", a.config.InventoryFilePath),
			mainTask.Host.Name,
			"--module-name=shell",
			fmt.Sprintf("--args=pgrep %s", mainTask.Command),
		}...)
		if err == nil {
			ready = true
			break
		}
		logger.Error("", zap.Error(err))

		logger.Info(fmt.Sprintf("main task not running yet, sleeping 3 seconds (try: %d/%d) ...", i, tries))
		time.Sleep(3 * time.Second)
	}

	if ready {
		for i, task := range mainTask.SubTasks {
			logger.Info(fmt.Sprintf("running sub task %d of %d", i+1, len(mainTask.SubTasks)), zap.String("hostname", task.Host.Name))

			wg.Add(1)
			go func(task *testers.Task) {
				ctx, cancel := context.WithTimeout(context.Background(), a.config.Timeouts.TaskCommandTimeout)
				defer cancel()

				defer wg.Done()

				// Template command and args for each task
				if err := cmdtemplate.Template(task, templateVars); err != nil {
					erro := fmt.Errorf("failed to template task command and / or args. %+v", err)
					logger.Error("error during createPodsForTasks", zap.String("hostname", task.Host.Name), zap.Error(erro))
					mainTask.Status.AddFailedClient(task.Host, erro)
					return
				}

				testTime := time.Now()

				out, err := a.executor.ExecuteCommandWithOutputByte(ctx, "runner:ansible: run sub task command", a.config.AnsibleCommand, []string{
					fmt.Sprintf("--inventory=%s", a.config.InventoryFilePath),
					task.Host.Name,
					"--module-name=shell",
					fmt.Sprintf("--args=%s %s", task.Command, strings.Join(task.Args, " ")),
				}...)
				if err != nil {
					logger.Error("client task failed", zap.String("hostname", task.Host.Name), zap.Error(err))
					mainTask.Status.AddFailedClient(task.Host, err)
					return
				}

				mainTask.Status.AddSuccessfulClient(task.Host)

				// Clean, "transform" to io.Reader compatible interface and send logs to parsers
				out = cleanAnsibleOutput(out)
				r := ioutil.NopCloser(bytes.NewReader(out))

				parser <- parsers.Input{
					TestStartTime:  plannedTime,
					TestTime:       testTime,
					Round:          round,
					DataStream:     &r,
					Tester:         tester,
					ServerHost:     mainTask.Host.Name,
					ClientHost:     task.Host.Name,
					AdditionalInfo: a.additionalInfo,
				}
			}(task)

			if a.runOptions.Mode != config.RunModeParallel {
				wg.Wait()
			}
		}

		// When RunOptions.Mode `parallel` then we wait after all test tasks have been run
		if a.runOptions.Mode == config.RunModeParallel {
			wg.Wait()
		}

		mainTask.Status.AddSuccessfulServer(mainTask.Host)
		mainTaskStopped = true
	} else {
		err := fmt.Errorf("ansible main test task is not running")
		mainTask.Status.AddFailedServer(mainTask.Host, err)
		return err
	}

	logger.Info("stopping main task")
	mainCancel()
	mainWG.Wait()

	logger.Debug("done running tasks for test in ansible for plan")

	return nil
}

// Cleanup remove all (left behind) Ansible resources created for the given Plan.
func (a *Ansible) Cleanup(plan *testers.Plan) error {
	// Nothing to do here for Ansible (yet)
	return nil
}

func cleanAnsibleOutput(in []byte) []byte {
	return jsonTailCleanRegex.ReplaceAll(
		jsonHeadCleanRegex.ReplaceAll(in, []byte("{")),
		[]byte("}"))
}
