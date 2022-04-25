# Run

Prepare data:
- `./scripts/download_nodes.sh`
- `./scripts/download_pods.sh`

Build app:
- `./scripts/download_deps.sh 1.23.4`
- `go build cmd/main.go bench`

Run:
- `bench all` - bench all configs

Other commands:
- `bench import-nodes` - import nodes from file to kube-scheduler-simulator
- `bench import-pods` - import pods from file to kube-scheduler-simulator
- `bench import-config` - import default config from file to kube-scheduler-simulator
- `bench list-nodes` - display nodes with scheduled pods and calculate stats - CPU/Memory imbalance
- `bench reset` - reset kube-scheduler-simulator state
