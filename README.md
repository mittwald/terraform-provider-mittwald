# mittwald Terraform provider

This repository contains a [Terraform](https://www.terraform.io) provider for resources on the mittwald cloud platform. 

## Installation

> [!IMPORTANT]
> This provider is not yet available on the Terraform registry. You have to build it yourself; have a look at the [Contributing](#contributing) section for more information.

## Usage

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
