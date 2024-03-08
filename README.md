Для запуска сценария тестирования необходимо сбилдить контейнер со сценарием для этого необходимо выполнить команду

```sh
docker build -f Dockerfile.scenario -t omes-scenario .
```
После билда контейнера, мы можем его запустить есть два варианта выполнение:
1. По времени выполнение
	--duration - флаг указывает в течении какого времени будет выполняться сценарий
```sh
docker run -d --name omes-scenario-1 --network temporal-network --network-alias scenario-1 omes-scenario run-scenario --duration 40m --prom-listen-address scenario-1:8079 --max-concurrent 2 --scenario mainscenarios --run-id test1 --server-address 172.17.0.1:7233
```

2. По кол-во итераций
	--iterations - флаг указывает кол-во сценариев которые надо выполнить
```sh
docker run -d --name omes-scenario-1 --network temporal-network --network-alias scenario-1 omes-scenario run-scenario --iterations 500 --prom-listen-address scenario-1:8079 --max-concurrent 2 --scenario mainscenarios --run-id test1 --server-address 172.17.0.1:7233
```

Описание флагов

--scenario            - имя сценария

--max-concurrent      - кол-во одновременно запущенных сценариев

--run-id              - id для генерации имени очереди

--server-address      - адрес кластера temporal

--prom-listen-address - адрес куда будем выкладывать метрики

Для запуска контейнера worker-а необходимо сбилдить контейнер с worker для этого необходимо выполнить команду
```sh
docker build -f Dockerfile.worker -t omes-worker .
```
После билда контейнера, мы можем его запустить:

```sh
docker run -p 8070:8070 -d --name omes-worker-1 --network temporal-network --network-alias worker-1 omes-worker run-worker --scenario mainscenarios --run-id test1 --language go --server-address 172.17.0.1:7233 --worker-prom-listen-address worker-1:8070
```
Описание флагов

--scenario                   - имя сценария

--run-id                     - id для генерации имени очереди

--language                   - язык исполнения сценария

--server-address             - адрес кластера temporal

--worker-prom-listen-address - адрес куда будем выкладывать метрики


# Omes - a load generator for Temporal

This project is for testing load generation scenarios against Temporal. This is primarily used by the Temporal team to
benchmark features and situations. Backwards compatibility may not be maintained.

## Why the weird name?

Omes (pronounced oh-mess) is the Hebrew word for "load" (עומס).

## Prerequisites

- [Go](https://golang.org/) 1.20+
- [Node](https://nodejs.org) 16+
- [Python](https://www.python.org/) 3.10+
  - [Poetry](https://python-poetry.org/): `poetry install`

(More TBD when we support workers in other languages)

## Installation

There's no need to install anything to use this, it's a self-contained Go project.

## Usage

### Define a scenario

Scenarios are defined using plain Go code. They are located in the [scenarios](./scenarios/) folder. There are already
multiple defined that can be used.

A scenario must select an `Executor`. The most common is the `KitchenSinkExecutor` which is a wrapper on the
`GenericExecutor` specific for executing the Kitchen Sink workflow. The Kitchen Sink workflow accepts
[actions](./workers/go/kitchensink/kitchen_sink.go) and is implemented in every worker language.

For example, here is [scenarios/workflow_with_single_noop_activity.go](scenarios/workflow_with_single_noop_activity.go):

```go
func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Each iteration executes a single workflow with a noop activity.",
		Executor: loadgen.KitchenSinkExecutor{
			WorkflowParams: kitchensink.NewWorkflowParams(kitchensink.NopActionExecuteActivity),
		},
	})
}
```

> NOTE: The file name where the `Register` function is called, will be used as the name of the scenario.

#### Scenario Authoring Guidelines

1. Use snake case for scenario file names.
1. Use `KitchenSinkExecutor` for most basic scenarios, adding common/generic actions as need, but for unique
   scenarios use `GenericExecutor`.
1. When using `GenericExecutor`, use methods of `*loadgen.Run` in your `Execute` as much as possible.
1. Liberally add helpers to the `loadgen` package that will be useful to other scenario authors.

### Run a worker for a specific language SDK

```sh
go run ./cmd run-worker --scenario workflow_with_single_noop_activity --run-id local-test-run --language go
```

Notes:

- `--embedded-server` can be passed here to start an embedded localhost server
- `--task-queue-suffix-index-start` and `--task-queue-suffix-index-end` represent an inclusive range for running the
  worker on multiple task queues. The process will create a worker for every task queue from `<task-queue>-<start>`
  through `<task-queue>-end`. This only applies to multi-task-queue scenarios.

### Run a test scenario

```sh
go run ./cmd run-scenario --scenario workflow_with_single_noop_activity --run-id local-test-run
```

Notes:

- Run ID is used to derive ID prefixes and the task queue name, it should be used to start a worker on the correct task queue
  and by the cleanup script.
- By default the number of iterations or duration is specified in the scenario config. They can be overridden with CLI
  flags.
- See help output for available flags.

### Cleanup after scenario run

```sh
go run ./cmd cleanup-scenario --scenario workflow_with_single_noop_activity --run-id local-test-run
```

### Run scenario with worker - Start a worker, an optional dev server, and run a scenario

```sh
go run ./cmd run-scenario-with-worker --scenario workflow_with_single_noop_activity --language go --embedded-server
```

Notes:

- Cleanup is **not** automatically performed here
- Accepts combined flags for `run-worker` and `run-scenario` commands

### Building and publishing docker images

For example, to build a go worker image using v1.24.0 of the Temporal Go SDK:

```sh
go run ./cmd build-worker-image --language go --version v1.24.0
```

This will produce an image tagged like `<current git commit hash>-go-v1.24.0`.

Publishing images is typically done via CI, using the `push-images` command. See the GHA workflows
for more.

## Design decisions

### Kitchen Sink Workflow

The Kitchen Sink workflows accepts a DSL generated by the `kitchen-sink-gen` Rust tool, allowing us
to test a wide variety of scenarios without having to imagine all possible edge cases that could
come up in workflows. Input may be saved for regression testing, or hand written for specific cases.

### Scenario Failure

A scenario can only fail if an `Execute` method returns an error, that means the control is fully in the scenario
authors's hands. For enforcing a timeout for a scenario, use options like workflow execution timeouts or write a
workflow that waits for a signal for a configurable amount of time.

## TODO

- Nicer output that includes resource utilization for the worker (when running all-in-one)
- More lang workers

## Fuzzer trophy case

* Python upsert SA with no initial attributes: [PR](https://github.com/temporalio/sdk-python/pull/440)
* Core cancel-before-start on abandon activities: [PR](https://github.com/temporalio/sdk-core/pull/652)
* Core panic on evicting run with buffered tasks: [PR](https://github.com/temporalio/sdk-core/pull/660)
