# mittwald Terraform provider

This repository contains a [Terraform](https://www.terraform.io) provider for resources on the mittwald cloud platform. 

## Installation

> [!IMPORTANT]
> This provider is an alpha state; we do not fully recommend using it in production environments, as it is not yet feature-complete and may contain bugs. However, we are happy to receive feedback and contributions!

You can install this provider from the [Terraform registry](https://registry.terraform.io/providers/mittwald/mittwald/latest). For this, add the following code to your Terraform configuration:

```hcl
terraform {
  required_providers {
    mittwald = {
      source = "mittwald/mittwald"
      version = "1.0.0-alpha1"
    }
  }
}

provider "mittwald" {
}
```

In order to use this provider, you need to have a [mittwald mStudio account](https://studio.mittwald.de) with an API key. You can then configure the provider with the following environment variables:

- `MITTWALD_API_TOKEN`: Your API token; see our API documentation on [how to obtain an API token](https://developer.mittwald.de/docs/v2/api/intro/).

## Usage

Have a look at the [general provider usage](https://registry.terraform.io/providers/mittwald/mittwald/latest/docs) to see how to get started with this provider.

This provider offers the following resources:

- [`mittwald_project`](https://registry.terraform.io/providers/mittwald/mittwald/latest/docs/resources/project)
- [`mittwald_app`](https://registry.terraform.io/providers/mittwald/mittwald/latest/docs/resources/app)
- [`mittwald_mysql_database`](https://registry.terraform.io/providers/mittwald/mittwald/latest/docs/resources/mysql_database)
- [`mittwald_cronjob`](https://registry.terraform.io/providers/mittwald/mittwald/latest/docs/resources/cronjob)

and the following data sources:

- [`mittwald_systemsoftware`](https://registry.terraform.io/providers/mittwald/mittwald/latest/docs/data-sources/systemsoftware)
- [`mittwald_user`](https://registry.terraform.io/providers/mittwald/mittwald/latest/docs/data-sources/user)

Coming soon:

- `mittwald_mysql_user`
- `mittwald_redis_database`
- `mittwald_ssh_user`
- `mittwald_ingress`
- `mittwald_dns_zone`
- `mittwald_email_address`
- `mittwald_email_deliverybox`

## Contributing

### Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.19

### Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

### Configuring the development override

Take note where you installed the `terraform-provider-mittwald` binary (most likely in your `$GOPATH/bin` directory). Then, create a file called `~/.terraformrc` with the following contents (replacing the path with the path to your binary):

```
provider_installation {
    dev_overrides {
        "registry.terraform.io/mittwald/mittwald" = "/opt/go/bin/terraform-provider-mittwald"
    }

    # For all other providers, install them directly from their origin provider
    # registries as normal. If you omit this, Terraform will _only_ use
    # the dev_overrides block, and so no other providers will be available.
    direct {}
}
```

### Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```
