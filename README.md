# terrascaler

Terrascaler is an external gRPC cloud provider for Kubernetes Cluster Autoscaler.
Instead of creating machines directly, it updates a configured integer value in a
Terraform/OpenTofu repository through the GitLab API. The resulting GitLab commit
can then trigger the repository's CI pipeline to run Terraform/OpenTofu and add
workers.

The implementation follows the Cluster Autoscaler external gRPC provider service:
`clusterautoscaler.cloudprovider.v1.externalgrpc.CloudProvider`.

## Scope

Terrascaler currently supports scale-up only. `NodeGroupIncreaseSize` reads the
configured Terraform attribute, adds the autoscaler delta, and commits the updated
file to GitLab. Node deletion and target-size decreases intentionally return
`Unimplemented`.

## Configuration

All settings can be passed as flags or environment variables.

| Flag                   | Environment                      | Required | Default    | Description                                                   |
| ---------------------- | -------------------------------- | -------- | ---------- | ------------------------------------------------------------- |
| `--listen`             | `TERRASCALER_LISTEN`             | no       | `:8080`    | gRPC listen address                                           |
| `--tls-cert-file`      | `TERRASCALER_TLS_CERT_FILE`      | no       |            | Server TLS certificate file                                   |
| `--tls-key-file`       | `TERRASCALER_TLS_KEY_FILE`       | no       |            | Server TLS key file                                           |
| `--tls-client-ca-file` | `TERRASCALER_TLS_CLIENT_CA_FILE` | no       |            | Client CA file; enables mTLS when set                         |
| `--gitlab-base-url`    | `GITLAB_BASE_URL`                | no       | GitLab.com | GitLab URL, for example `https://gitlab.example.com`          |
| `--gitlab-token`       | `GITLAB_TOKEN`                   | yes      |            | GitLab API token with repository write access                 |
| `--gitlab-project`     | `GITLAB_PROJECT`                 | yes      |            | GitLab project ID or path, for example `group/platform/infra` |
| `--gitlab-branch`      | `GITLAB_BRANCH`                  | no       | `main`     | Branch to update                                              |
| `--file`               | `TERRASCALER_FILE`               | yes      |            | Terraform file path in the repository                         |
| `--block-type`         | `TERRASCALER_BLOCK_TYPE`         | no       | `module`   | Terraform block type containing the field                     |
| `--block-labels`       | `TERRASCALER_BLOCK_LABELS`       | no       |            | Comma-separated block labels                                  |
| `--attribute`          | `TERRASCALER_ATTRIBUTE`          | yes      |            | Integer Terraform attribute to update                         |
| `--node-group-id`      | `TERRASCALER_NODE_GROUP_ID`      | no       | `default`  | Node group ID exposed to Cluster Autoscaler                   |
| `--min-size`           | `TERRASCALER_MIN_SIZE`           | no       | `0`        | Node group minimum                                            |
| `--max-size`           | `TERRASCALER_MAX_SIZE`           | no       | `100`      | Node group maximum                                            |
| `--template-cpu`       | `TERRASCALER_TEMPLATE_CPU`       | no       | `2`        | Template node CPU capacity for scale-up simulation            |
| `--template-memory`    | `TERRASCALER_TEMPLATE_MEMORY`    | no       | `8Gi`      | Template node memory capacity                                 |
| `--template-pods`      | `TERRASCALER_TEMPLATE_PODS`      | no       | `110`      | Template node pod capacity                                    |
| `--template-labels`    | `TERRASCALER_TEMPLATE_LABELS`    | no       |            | Comma-separated labels for the template node, `key=value`     |

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
  --node-group-id hostedcluster-workers \
  --min-size 3 \
  --max-size 20
```

Cluster Autoscaler should use `--cloud-provider=externalgrpc` and a cloud config
that points at this service address, as documented by upstream Kubernetes
Autoscaler.

```yaml
address: terrascaler.default.svc.cluster.local:8080
grpc_timeout: 30s
```

For TLS or mTLS, provide `--tls-cert-file` and `--tls-key-file` to Terrascaler.
Providing `--tls-client-ca-file` enables client certificate verification, matching
the upstream external gRPC provider recommendation.

## Notes

The target Terraform value must currently be a literal integer. Values such as
`worker_count = var.worker_count` are rejected because Terrascaler cannot safely
evaluate them from a single file.

## Development

Common targets follow the other Containeroo Go tools:

```sh
make test
make build
make docker-build IMG=ghcr.io/containeroo/terrascaler:dev
```
