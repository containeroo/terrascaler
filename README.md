# terrascaler

Terrascaler is a small Kubernetes autoscaler for clusters whose worker count is
managed by Terraform/OpenTofu in GitLab.

It does not implement a Kubernetes Cluster Autoscaler cloud provider. Instead,
Terrascaler runs its own reduced autoscaling loop:

1. Read the current Terraform worker count from GitLab.
2. List Kubernetes nodes and pods.
3. Find pending unscheduled pods.
4. Estimate whether those pods fit on current worker capacity.
5. If not, compute how many worker nodes are needed.
6. Commit the updated Terraform worker count to GitLab.

The GitLab commit is expected to trigger the repository's CI pipeline, which then
runs Terraform/OpenTofu and adds workers.

## Scope

Terrascaler intentionally supports a narrow feature set:

- scale up only
- one Terraform-managed worker group
- CPU, memory, and pod-count bin packing
- optional node label selector for Terraform-managed workers
- no cloud provider integrations
- no scale down
- no expander strategies, priorities, balancing, or pricing logic

## Configuration

All settings can be passed as flags or environment variables.

| Flag                    | Environment                       | Required | Default           | Description                                                                     |
| ----------------------- | --------------------------------- | -------- | ----------------- | ------------------------------------------------------------------------------- |
| `--kubeconfig`          | `KUBECONFIG`                      | no       | in-cluster config | Path to kubeconfig                                                              |
| `--check-interval`      | `TERRASCALER_CHECK_INTERVAL`      | no       | `1m`              | Autoscaling check interval                                                      |
| `--scale-up-cooldown`   | `TERRASCALER_SCALE_UP_COOLDOWN`   | no       | `5m`              | Minimum time between scale-up commits                                           |
| `--pending-pod-min-age` | `TERRASCALER_PENDING_POD_MIN_AGE` | no       | `30s`             | Minimum age for pending pods without an Unschedulable condition                 |
| `--metrics-address`     | `TERRASCALER_METRICS_ADDRESS`     | no       | `:8080`           | Prometheus metrics listen address; empty disables metrics                       |
| `--once`                | `TERRASCALER_ONCE`                | no       | `false`           | Run one autoscaling check and exit                                              |
| `--dry-run`             | `TERRASCALER_DRY_RUN`             | no       | `false`           | Log intended scaling actions without updating GitLab                            |
| `--gitlab-base-url`     | `GITLAB_BASE_URL`                 | no       | GitLab.com        | GitLab URL, for example `https://gitlab.example.com`                            |
| `--gitlab-token`        | `GITLAB_TOKEN`                    | yes      |                   | GitLab API token with repository write access                                   |
| `--gitlab-project`      | `GITLAB_PROJECT`                  | yes      |                   | GitLab project ID or path, for example `group/platform/infra`                   |
| `--gitlab-branch`       | `GITLAB_BRANCH`                   | no       | `main`            | Branch to update                                                                |
| `--gitlab-merge-request` | `TERRASCALER_GITLAB_MERGE_REQUEST` | no | `false` | Create or update a GitLab merge request instead of committing directly |
| `--gitlab-mr-branch-prefix` | `TERRASCALER_GITLAB_MR_BRANCH_PREFIX` | no | `terrascaler/scale` | Branch prefix for merge request mode |
| `--gitlab-mr-title` | `TERRASCALER_GITLAB_MR_TITLE` | no | `terrascaler: scale worker count` | Merge request title |
| `--gitlab-mr-description` | `TERRASCALER_GITLAB_MR_DESCRIPTION` | no | automated proposal text | Merge request description prefix |
| `--gitlab-mr-labels` | `TERRASCALER_GITLAB_MR_LABELS` | no | `terrascaler` | Comma-separated merge request labels |
| `--gitlab-mr-assignee-ids` | `TERRASCALER_GITLAB_MR_ASSIGNEE_IDS` | no | | Comma-separated GitLab user IDs to assign |
| `--gitlab-mr-reviewer-ids` | `TERRASCALER_GITLAB_MR_REVIEWER_IDS` | no | | Comma-separated GitLab user IDs to request review from |
| `--gitlab-mr-remove-source-branch` | `TERRASCALER_GITLAB_MR_REMOVE_SOURCE_BRANCH` | no | `true` | Remove MR source branch after merge |
| `--file`                | `TERRASCALER_FILE`                | yes      |                   | Terraform file path in the repository                                           |
| `--block-type`          | `TERRASCALER_BLOCK_TYPE`          | no       | `module`          | Terraform block type containing the field                                       |
| `--block-labels`        | `TERRASCALER_BLOCK_LABELS`        | no       |                   | Comma-separated Terraform block labels                                          |
| `--attribute`           | `TERRASCALER_ATTRIBUTE`           | yes      |                   | Integer Terraform attribute to update                                           |
| `--min-size`            | `TERRASCALER_MIN_SIZE`            | no       | `0`               | Minimum target size                                                             |
| `--max-size`            | `TERRASCALER_MAX_SIZE`            | no       | `100`             | Maximum target size                                                             |
| `--node-selector`       | `TERRASCALER_NODE_SELECTOR`       | no       |                   | Comma-separated node labels, `key=value`, identifying Terraform-managed workers |
| `--template-cpu`        | `TERRASCALER_TEMPLATE_CPU`        | no       | `2`               | New worker CPU capacity                                                         |
| `--template-memory`     | `TERRASCALER_TEMPLATE_MEMORY`     | no       | `8Gi`             | New worker memory capacity                                                      |
| `--template-pods`       | `TERRASCALER_TEMPLATE_PODS`       | no       | `110`             | New worker pod capacity                                                         |
| `--template-labels`     | `TERRASCALER_TEMPLATE_LABELS`     | no       |                   | Reserved metadata for new worker labels                                         |

Example for:

```hcl
module "hostedcluster" {
  worker_count = 3
}
```

```sh
terrascaler \
  --gitlab-token "$GITLAB_TOKEN" \
  --gitlab-project "platform/cluster-infra" \
  --gitlab-branch main \
  --file main.tf \
  --block-type module \
  --block-labels hostedcluster \
  --attribute worker_count \
  --min-size 3 \
  --max-size 20 \
  --node-selector node-role.kubernetes.io/worker= \
  --template-cpu 4 \
  --template-memory 16Gi \
  --template-pods 110
```

## Autoscaling Behavior

Terrascaler considers a pod eligible for scale-up when it is pending, not bound to
a node, and either:

- has `PodScheduled=False` with reason `Unschedulable`, or
- is older than `--pending-pod-min-age`.

It subtracts scheduled pod requests from matching ready schedulable nodes, then
tries to pack eligible pending pods into the remaining capacity. Any pods that do
not fit are packed onto synthetic nodes using `--template-cpu`,
`--template-memory`, and `--template-pods`. The number of synthetic nodes needed
is added to the current Terraform target, capped at `--max-size`.

## Monitoring

Terrascaler exposes Prometheus metrics on `/metrics` at `--metrics-address`.

Important metrics:

- `terrascaler_scale_down_potential_nodes`: approximate number of nodes that may
  be removable. Terrascaler only reports this and does not scale down.
- `terrascaler_current_target_nodes`: current Terraform target node count.
- `terrascaler_desired_target_nodes`: desired target node count from the latest
  plan.
- `terrascaler_new_nodes_required`: new nodes needed by the latest plan.
- `terrascaler_last_check_success`: `1` when the last check succeeded.

The target Terraform value must be a literal integer. Values such as
`worker_count = var.worker_count` are rejected because Terrascaler cannot safely
evaluate them from a single file.

## Merge Request Mode

By default Terrascaler commits directly to `--gitlab-branch`. With
`--gitlab-merge-request`, Terrascaler creates a branch from the target branch,
commits the Terraform change there, and opens a GitLab merge request. Optional
assignee and reviewer settings use GitLab numeric user IDs.

## Development

Common targets follow the other Containeroo Go tools:

```sh
make test
make build
make docker-build IMG=ghcr.io/containeroo/terrascaler:dev
```
