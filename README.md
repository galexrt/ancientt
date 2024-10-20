# ancientt

A tool to automate network testing tools, like iperf3, in dynamic environments such as Kubernetes and more to come.

Container Image available from:

* [GHCR.io](https://github.com/users/galexrt/packages/container/package/ancientt)
* [Quay.io](https://quay.io/repository/galexrt/ancientt?tab=tags)

Container Image Tags:

* `main` - Latest build of the `main` branch.
* `vX.Y.Z` - Tagged build of the application.

## Features

**TL;DR** A network test tool, like `iperf3` can be run in, e.g., Kubernetes, cluster from all-to-all Nodes.

* Run network tests with the following projects:
  * [`iperf3`](https://iperf.fr/)
  * [PingParsing](https://github.com/thombashi/pingparsing)
  * Soon more tools will be available as well, see [GitHub Issues with "testers" Label](https://github.com/galexrt/ancientt/issues?utf8=%E2%9C%93&q=is%3Aissue+is%3Aopen+label%3Atesters+).
* Tests can be run through the following "runners":
  * Ansible (an inventory file is needed)
  * Kubernetes (a kubeconfig connected to a cluster)
* Results of the network tests can be output in different formats:
  * CSV
  * Dump (uses `pp.Sprint()` ([GitHub k0kubun/pp](https://github.com/k0kubun/pp), pretty print library))
  * Excel files (using [Excelize](https://github.com/qax-os/excelize) library)
  * go-chart Charts (WIP)
  * MySQL
  * SQLite

## Usage

Either [build (`go get`)](#building), download the `ancientt` executable from the GitHub release page or use the Container image.

A config file containing test definitions must be given by flag `--testdefinition` (or short flag `-c`) or named `testdefinition.yaml` in the current directory.

Below command will try loading `your-testdefinitions.yaml` as the test definitions config:

```shell
$ ancientt --testdefinition your-testdefinitions.yaml
# You can also use the short flag `-c` instead of `--testdefinition`
# and also with `-y` run the tests immediately
$ ancientt -c your-testdefinitions.yaml -y
```

## Demos

See [Demos](docs/demos.md).

## Goals of this Project

* A bit like Prometheus blackbox exporter which contains "definitions" for probes. The "tests" would be pluggable through a Golang interface.
* "Runner" interface, e.g., for Kubernetes, Ansible, etc. The "runner" abstracts the "how it is run", e.g., for Kubernetes creates Pods and Jobs, Ansible to trigger a playbook to run the test.
* Store the result data in different formats, e.g., CSV, Excel, MySQL
  * Up for discussion: graph database ([Dgraph](https://dgraph.io/)) or TSDB support
* "Visualization" for humans, e.g., the possibility to automatically draw "shiny" graphs from the results.

## Development

**Golang version**: `v1.22` or higher (tested with `v1.22.4` on `linux/amd64`)

### Creating Release

1. Add a new entry for release to [`CHANGELOG.md`](CHANGELOG.md).
2. Update [`VERSION`](VERSION) with the new version number.
3. `git commit` and `git push` both changes (e.g., `version: update to VERSION_HERE`).
4. Now create the git tag and push the tag `git tag VERSION_HERE` followed by `git push --tags`.

### Dependencies

`go mod` is used to manage the dependencies.

### Building

The quickest way to just get `ancientt` built is to run the following command (requires `go` installed on the system):

```console
go get -u github.com/galexrt/ancientt/cmd/ancientt
```

## Licensing

ancientt is licensed under the Apache 2.0 License.

## Why the unfork?

`ancientt` was initially created at [Cloudical](https://cloudical.io).

It is easier for me to update the project this way and keep it going.
