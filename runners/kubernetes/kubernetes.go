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
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/galexrt/ancientt/parsers"
	"github.com/galexrt/ancientt/pkg/cmdtemplate"
	"github.com/galexrt/ancientt/pkg/config"
	"github.com/galexrt/ancientt/pkg/hostsfilter"
	"github.com/galexrt/ancientt/pkg/k8sutil"
	"github.com/galexrt/ancientt/pkg/util"
	"github.com/galexrt/ancientt/runners"
	"github.com/galexrt/ancientt/testers"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Name Kubernetes Runner Name
const Name = "kubernetes"

func init() {
	runners.Factories[Name] = NewRunner
}

// Kubernetes Kubernetes runner struct
type Kubernetes struct {
	runners.Runner
	logger     *zap.Logger
	config     *config.RunnerKubernetes
	k8sclient  kubernetes.Interface
	runOptions config.RunOptions
}

// NewRunner return a new Kubernetes Runner
func NewRunner(logger *zap.Logger, cfg *config.Config) (runners.Runner, error) {
	conf := cfg.Runner.Kubernetes

	clientset, err := k8sutil.NewClient(cfg.Runner.Kubernetes.InClusterConfig, cfg.Runner.Kubernetes.Kubeconfig)
	if err != nil {
		return nil, err
	}

	return &Kubernetes{
		logger:    logger.With(zap.String("runner", Name), zap.String("namespace", cfg.Runner.Kubernetes.Namespace)),
		config:    conf,
		k8sclient: clientset,
	}, nil
}

// GetHostsForTest return a mocked list of hots for the given test config
func (k *Kubernetes) GetHostsForTest(test *config.Test) (*testers.Hosts, error) {
	hosts := &testers.Hosts{
		Clients: map[string]*testers.Host{},
		Servers: map[string]*testers.Host{},
	}

	k8sNodes, err := k.k8sNodesToHosts()
	if err != nil {
		return nil, err
	}

	// Go through Hosts Servers list to get the servers hosts
	for _, servers := range test.Hosts.Servers {
		filtered, err := hostsfilter.FilterHostsList(k8sNodes, servers)
		if err != nil {
			return nil, err
		}
		for _, host := range filtered {
			if _, ok := hosts.Servers[host.Name]; !ok {
				hosts.Servers[host.Name] = host
			}
		}
	}

	// Go through Hosts Clients list to get the clients hosts
	for _, clients := range test.Hosts.Clients {
		filtered, err := hostsfilter.FilterHostsList(k8sNodes, clients)
		if err != nil {
			return nil, err
		}
		for _, host := range filtered {
			if _, ok := hosts.Clients[host.Name]; !ok {
				hosts.Clients[host.Name] = host
			}
		}
	}

	k.logger.Debug("returning Kubernetes hosts list")

	return hosts, nil
}

func (k *Kubernetes) k8sNodesToHosts() ([]*testers.Host, error) {
	hosts := []*testers.Host{}
	ctx := context.TODO()
	nodes, err := k.k8sclient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Quick conversion from a Kubernetes CoreV1 Nodes object to testers.Host
	for _, node := range nodes.Items {
		// Check if node is unschedulable
		if (k.config.Hosts != nil && *k.config.Hosts.IgnoreSchedulingDisabled) && node.Spec.Unschedulable {
			k.logger.Debug("skipping unschedulable node", zap.String("node", node.ObjectMeta.Name))
			continue
		}

		// Check if the taints on the node match the given tolerations
		tolerations := []corev1.Toleration{}
		if k.config.Hosts != nil {
			tolerations = k.config.Hosts.Tolerations
		}

		if !k8sutil.NodeIsTolerable(node, tolerations) {
			continue
		}

		hosts = append(hosts, &testers.Host{
			Labels: node.ObjectMeta.Labels,
			Name:   node.ObjectMeta.Name,
		})
	}

	return hosts, nil
}

// Prepare prepare Kubernetes for usage with ancientt, e.g., create Namespace.
func (k *Kubernetes) Prepare(runOpts config.RunOptions, plan *testers.Plan) error {
	k.runOptions = runOpts

	if err := k.prepareKubernetes(); err != nil {
		return err
	}

	return nil
}

// Execute run the given commands and return the logs of it and / or error
func (k *Kubernetes) Execute(plan *testers.Plan, parser chan<- parsers.Input) error {
	// TODO Add option to go through Service IPs instead of Pod IPs

	// Iterate over given plan.Commands to then run each task
	for round, tasks := range plan.Commands {
		k.logger.Info(fmt.Sprintf("running commands round %d of %d", round+1, len(plan.Commands)))
		for i, task := range tasks {
			if task.Sleep != 0 {
				k.logger.Info(fmt.Sprintf("waiting %s to pass before continuing next round", task.Sleep.String()))
				time.Sleep(task.Sleep)
				continue
			}
			k.logger.Info(fmt.Sprintf("running task round %d of %d", i+1, len(tasks)))

			// Create the Pods for the server task and client tasks
			if err := k.createPodsForTasks(round, task, plan, parser); err != nil {
				if !*plan.RunOptions.ContinueOnError {
					return err
				}
				k.logger.Warn("continuing after err", zap.Error(err))
			}
		}
	}

	return nil
}

// prepareKubernetes prepares Kubernetes by creating the namespace if it does not exist
func (k *Kubernetes) prepareKubernetes() error {
	// Check if namespaces exists, if not try create it
	ctx := context.TODO()
	if _, err := k.k8sclient.CoreV1().Namespaces().Get(ctx, k.config.Namespace, metav1.GetOptions{}); err != nil {
		// If namespace not found, create it
		if errors.IsNotFound(err) {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"created-by": "ancientt",
					},
					Name: k.config.Namespace,
				},
			}
			k.logger.Info("trying to create namespace")
			ctx := context.TODO()
			if _, err := k.k8sclient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{}); err != nil {
				return fmt.Errorf("failed to create namespace %s^", k.config.Namespace, err)
			}
			k.logger.Info("created namespace")
		} else {
			return fmt.Errorf("error while getting namespace %s. %+v", k.config.Namespace, err)
		}
	}
	return nil
}

// createPodsForTasks create the Pods that are needed for the task(s)
func (k *Kubernetes) createPodsForTasks(round int, mainTask *testers.Task, plan *testers.Plan, parser chan<- parsers.Input) error {
	logger := k.logger.With(zap.Int("round", round))

	var wg sync.WaitGroup

	taskName := util.GetTaskName(plan.Tester, plan.TestStartTime)

	// Create server Pod first
	serverPodName := util.GetPNameFromTask(round, mainTask.Host.Name, mainTask.Command, util.PNameRoleServer, plan.TestStartTime)

	// Create initial cmdtemplate.Variables
	templateVars := cmdtemplate.Variables{
		ServerPort: 5601,
	}

	if err := cmdtemplate.Template(mainTask, templateVars); err != nil {
		logger.Error("failed to template main task command and / or args", zap.Error(err))
		mainTask.Status.AddFailedServer(mainTask.Host, err)
		return nil
	}

	pod := k.getPodSpec(serverPodName, taskName, mainTask)
	k.applyServiceAccountToPod(pod, serverRole)

	logger = logger.With(zap.String("pod", serverPodName))
	logger.Debug("(re)creating server pod")
	if err := k8sutil.PodRecreate(k.k8sclient, pod, k.config.Timeouts.DeleteTimeout); err != nil {
		logger.Error(fmt.Sprintf("failed to create server pod %s/%s", k.config.Namespace, serverPodName, err, zap.Error(err)))
		mainTask.Status.AddFailedServer(mainTask.Host, err)
		return nil
	}

	logger.Info("waiting for server pod to run")
	running, err := k8sutil.WaitForPodToRun(k.k8sclient, k.config.Namespace, serverPodName, k.config.Timeouts.RunningTimeout)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to wait for server pod %s/%s", k.config.Namespace, serverPodName, err, zap.Error(err)))
		mainTask.Status.AddFailedServer(mainTask.Host, err)
		return nil
	}
	if !running {
		logger.Error(fmt.Sprintf("server pod %s/%s not running after runTimeout", k.config.Namespace, serverPodName, zap.Error(err)))
		mainTask.Status.AddFailedServer(mainTask.Host, err)
		return nil
	}

	// Get server Pod to have the server IP for each client task
	ctx := context.TODO()
	pod, err = k.k8sclient.CoreV1().Pods(k.config.Namespace).Get(ctx, serverPodName, metav1.GetOptions{})
	if err != nil {
		logger.Error(fmt.Sprintf("failed to get server pod %s/%s", k.config.Namespace, serverPodName, err, zap.Error(err)))
		mainTask.Status.AddFailedServer(mainTask.Host, err)
		return nil
	}
	if pod.Status.PodIP == "" {
		mainTask.Status.AddFailedServer(mainTask.Host,
			fmt.Errorf("failed to get server pod %s/%s IP, got '%s'", k.config.Namespace, serverPodName, pod.Status.PodIP))
		return nil
	}

	templateVars.ServerAddressV4 = pod.Status.PodIP

	for i, task := range mainTask.SubTasks {
		logger.Info(fmt.Sprintf("running sub task %d of %d", i+1, len(mainTask.SubTasks)))

		wg.Add(1)
		go func(task *testers.Task) {
			defer wg.Done()

			testTime := time.Now()

			pName := util.GetPNameFromTask(round, task.Host.Name, task.Command, util.PNameRoleClient, plan.TestStartTime)
			logger := logger.With(zap.String("pod", pName))

			// Template command and args for each task
			if err := cmdtemplate.Template(task, templateVars); err != nil {
				k.logger.Error("failed to template task command and / or args", zap.Error(err))
				mainTask.Status.AddFailedClient(task.Host, err)
				return
			}

			pod = k.getPodSpec(pName, taskName, task)
			k.applyServiceAccountToPod(pod, clientsRole)

			logger.Debug("(re)creating client pod")
			if err := k8sutil.PodRecreate(k.k8sclient, pod, k.config.Timeouts.DeleteTimeout); err != nil {
				k.logger.Error(fmt.Sprintf("failed to create pod %s/%s", k.config.Namespace, pName, err, zap.Error(err)))
				mainTask.Status.AddFailedClient(task.Host, err)
				return
			}

			logger.Info("waiting for client pod to run or succeed")
			running, err := k8sutil.WaitForPodToRunOrSucceed(k.k8sclient, k.config.Namespace, pName, k.config.Timeouts.RunningTimeout)
			if err != nil {
				k.logger.Error(fmt.Sprintf("failed to wait for pod %s/%s", k.config.Namespace, pName), zap.Error(err))
				mainTask.Status.AddFailedClient(task.Host, err)
				return
			}
			if !running {
				k.logger.Error(fmt.Sprintf("pod %s/%s not running after runTimeout", k.config.Namespace, pName), zap.Error(err))
				mainTask.Status.AddFailedClient(task.Host, err)
				return
			}

			logger.Debug("about to pushLogsToParser")
			if err := k.pushLogsToParser(parser, plan.TestStartTime, testTime, round, plan.Tester, mainTask.Host.Name, task.Host.Name, pName); err != nil {
				k.logger.Error(fmt.Sprintf("failed to push pod %s/%s logs to parser", k.config.Namespace, pName), zap.Error(err))
				mainTask.Status.AddFailedClient(task.Host, err)
				return
			}

			logger.Info("deleting client pod")
			if err := k8sutil.PodDelete(k.k8sclient, pod, k.config.Timeouts.DeleteTimeout); err != nil {
				logger.Error(fmt.Sprintf("failed to delete client pod %s/%s", k.config.Namespace, pName), zap.Error(err))
				mainTask.Status.AddFailedClient(task.Host, err)
				return
			}

			mainTask.Status.AddSuccessfulClient(task.Host)
		}(task)

		if k.runOptions.Mode != config.RunModeParallel {
			wg.Wait()
		}
	}

	// When RunOptions.Mode `parallel` then we wait after all test tasks have been run
	if k.runOptions.Mode == config.RunModeParallel {
		wg.Wait()
	}

	// Delete server pod
	logger.Info("deleting server pod")
	if err := k8sutil.PodDeleteByName(k.k8sclient, k.config.Namespace, serverPodName, k.config.Timeouts.DeleteTimeout); err != nil {
		logger.Error("failed to delete server pod", zap.Error(err))
		mainTask.Status.AddFailedServer(mainTask.Host, err)
		return nil
	}

	mainTask.Status.AddSuccessfulServer(mainTask.Host)

	logger.Debug("done running tasks for test in kubernetes for plan")

	return nil
}

func (k *Kubernetes) pushLogsToParser(parserInput chan<- parsers.Input, plannedTime time.Time, testTime time.Time, round int, tester string, serverHost string, clientHost string, podName string) error {
	// Wait for the Pod to succeed because that is the "sign" that the test for that Pod is done.
	succeeded, err := k8sutil.WaitForPodToSucceed(k.k8sclient, k.config.Namespace, podName, k.config.Timeouts.SucceedTimeout)
	if err != nil {
		return err
	}

	if succeeded {
		// "Generate" request for logs of Pod
		req := k.k8sclient.CoreV1().Pods(k.config.Namespace).GetLogs(podName, &corev1.PodLogOptions{})

		// Start the log stream
		ctx := context.TODO()
		podLogs, err := req.Stream(ctx)
		if err != nil {
			return err
		}
		// Don't close the `podLogs` here, that is the responsibility of the parser!

		// Send the logs to the parser.InputChan
		parserInput <- parsers.Input{
			TestStartTime:  plannedTime,
			TestTime:       testTime,
			Round:          round,
			DataStream:     &podLogs,
			Tester:         tester,
			ServerHost:     serverHost,
			ClientHost:     clientHost,
			AdditionalInfo: podName,
		}
		return nil
	}

	return fmt.Errorf("pod %s/%s has not succeeded", k.config.Namespace, podName)
}

// Cleanup remove all (left behind) Kubernetes resources created for the given Plan.
func (k *Kubernetes) Cleanup(plan *testers.Plan) error {
	var wg sync.WaitGroup

	// Delete all Pods with label XYZ
	if err := k8sutil.PodDeleteByLabels(k.k8sclient, k.config.Namespace, map[string]string{
		k8sutil.TaskIDLabel: util.GetTaskName(plan.Tester, plan.TestStartTime),
	}); err != nil {
		k.logger.Error("error during pod delete by labels in cleanup", zap.Error(err))
		return err
	}
	wg.Wait()

	return nil
}
