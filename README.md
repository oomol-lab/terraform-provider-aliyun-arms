# Terraform Provider for Alibaba Cloud ARMS

[![Tests](https://github.com/oomol-lab/terraform-provider-aliyun-arms/actions/workflows/test.yml/badge.svg)](https://github.com/oomol-lab/terraform-provider-aliyun-arms/actions/workflows/test.yml)

This Terraform provider manages Alibaba Cloud Application Real-Time Monitoring
Service (ARMS) features that are not yet available in the official
`aliyun/alicloud` provider.

The initial release provides `arms_prometheus_alert_rule`, a resource for
custom PromQL alert rules backed by the ARMS `CreateOrUpdateAlertRule` API.

> This is a community provider maintained by OOMOL Lab. It is not an official
> Alibaba Cloud provider.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) 1.0 or later
- An Alibaba Cloud account with permission to manage ARMS alert rules
- [Go](https://go.dev/doc/install) 1.24 or later, for development only

## Usage

```terraform
terraform {
  required_providers {
    arms = {
      source  = "oomol-lab/aliyun-arms"
      version = "~> 0.1"
    }
  }
}

provider "arms" {
  region  = "ap-southeast-1"
  profile = "default"
}
```

The optional `profile` selects an Alibaba Cloud CLI profile. When it is
omitted, the Alibaba Cloud default credential chain is used. Credentials must
not be committed to Terraform configuration.

See the generated [provider documentation](docs/index.md) and
[`arms_prometheus_alert_rule`](docs/resources/prometheus_alert_rule.md)
resource documentation for the full schema and examples.

## Development

```shell
make fmt
make test
make build
make generate
```

For local Terraform testing, build the provider and configure a Terraform CLI
development override for `registry.terraform.io/oomol-lab/aliyun-arms`.

## Releasing

Releases are built by GitHub Actions and GoReleaser. Create and push a semantic
version tag such as `v0.1.0`; no repository secrets are required. The release
workflow builds platform-specific zip archives, a Terraform manifest, and a
SHA-256 checksum file, then publishes them to GitHub Releases.

Release assets are not currently GPG-signed or published through the Terraform
Registry. Published assets must never be replaced; fixes require a new version.

## Current limitations

- Only custom PromQL (`AlertCheckType=CUSTOM`) rules are supported.
- Rules use `AlertGroup=-1`, `NotifyMode=NORMAL_MODE`, and
  `AlertType=PROMETHEUS_MONITORING_ALERT_RULE`.
- Notification policies remain separately managed; this provider does not set
  `NotifyStrategy`.
- `CheckCycle` is not exposed by the current official ARMS Go SDK and is not
  configurable.
- `GetAlertRules` does not return `DataConfig`, so `no_data_revision` cannot be
  checked for drift.

## License

This project is licensed under the [Mozilla Public License 2.0](LICENSE).
