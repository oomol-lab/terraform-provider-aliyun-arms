# Terraform Provider for Alibaba Cloud ARMS

[![Tests](https://github.com/oomol-lab/terraform-provider-aliyun-arms/actions/workflows/test.yml/badge.svg)](https://github.com/oomol-lab/terraform-provider-aliyun-arms/actions/workflows/test.yml)

<p align="center">
  <a href="#english">English</a> | <a href="#简体中文">简体中文</a>
</p>

## English

This Terraform provider manages Alibaba Cloud Application Real-Time Monitoring
Service (ARMS) features that are not yet available in the official
`aliyun/alicloud` provider.

The initial release provides `arms_prometheus_alert_rule`, a resource for
custom PromQL alert rules backed by the ARMS `CreateOrUpdateAlertRule` API.

> This is a community provider maintained by OOMOL Lab. It is not an official
> Alibaba Cloud provider.

### Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) 1.0 or later
- An Alibaba Cloud account with permission to manage ARMS alert rules
- [Go](https://go.dev/doc/install) 1.24 or later, for development only

### Usage

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

### Development

```shell
make fmt
make test
make build
make generate
```

For local Terraform testing, build the provider and configure a Terraform CLI
development override for `registry.terraform.io/oomol-lab/aliyun-arms`.

### Releasing

Releases are built by GitHub Actions and GoReleaser. Create and push a semantic
version tag such as `v0.1.0`; no repository secrets are required. The release
workflow builds platform-specific zip archives, a Terraform manifest, and a
SHA-256 checksum file, then publishes them to GitHub Releases.

Release assets are not currently GPG-signed or published through the Terraform
Registry. Published assets must never be replaced; fixes require a new version.

### Current limitations

- Only custom PromQL (`AlertCheckType=CUSTOM`) rules are supported.
- Rules use `AlertGroup=-1`, `NotifyMode=NORMAL_MODE`, and
  `AlertType=PROMETHEUS_MONITORING_ALERT_RULE`.
- Notification policies remain separately managed; this provider does not set
  `NotifyStrategy`.
- `CheckCycle` is not exposed by the current official ARMS Go SDK and is not
  configurable.
- `GetAlertRules` does not return `DataConfig`, so `no_data_revision` cannot be
  checked for drift.

### License

This project is licensed under the [Mozilla Public License 2.0](LICENSE).

---

## 简体中文

这个 Terraform Provider 用于管理官方 `aliyun/alicloud` Provider 尚未支持的
阿里云应用实时监控服务（ARMS）功能。

首个版本提供 `arms_prometheus_alert_rule` 资源，通过 ARMS
`CreateOrUpdateAlertRule` API 管理自定义 PromQL 告警规则。

> 这是由 OOMOL Lab 维护的社区 Provider，并非阿里云官方 Provider。

### 环境要求

- [Terraform](https://developer.hashicorp.com/terraform/downloads) 1.0 或更高版本
- 具备 ARMS 告警规则管理权限的阿里云账号
- [Go](https://go.dev/doc/install) 1.24 或更高版本（仅开发时需要）

### 使用方法

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

可选的 `profile` 参数用于选择阿里云 CLI 配置。省略该参数时，将使用阿里云默认凭证链。
请勿将凭证提交到 Terraform 配置中。

完整的 Schema 和示例请参阅自动生成的 [Provider 文档](docs/index.md)以及
[`arms_prometheus_alert_rule`](docs/resources/prometheus_alert_rule.md) 资源文档。

### 开发

```shell
make fmt
make test
make build
make generate
```

如需在本地测试 Terraform，请先构建 Provider，再为
`registry.terraform.io/oomol-lab/aliyun-arms` 配置 Terraform CLI 开发覆盖。

### 发布

Release 由 GitHub Actions 和 GoReleaser 构建。创建并推送 `v0.1.0` 之类的语义化版本
标签即可，无需配置仓库 Secret。发布工作流会构建各平台的 zip 压缩包、Terraform
manifest 和 SHA-256 校验文件，并将其发布到 GitHub Releases。

当前发布产物不带 GPG 签名，也未发布到 Terraform Registry。已发布的产物不得替换；
如需修复，必须发布新版本。

### 当前限制

- 仅支持自定义 PromQL（`AlertCheckType=CUSTOM`）规则。
- 规则固定使用 `AlertGroup=-1`、`NotifyMode=NORMAL_MODE` 和
  `AlertType=PROMETHEUS_MONITORING_ALERT_RULE`。
- 通知策略仍需单独管理；本 Provider 不设置 `NotifyStrategy`。
- 当前官方 ARMS Go SDK 未暴露 `CheckCycle`，因此无法配置。
- `GetAlertRules` 不返回 `DataConfig`，因此无法检查 `no_data_revision` 的配置漂移。

### 许可证

本项目采用 [Mozilla Public License 2.0](LICENSE) 许可证。
