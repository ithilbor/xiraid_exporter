# Contributing

E4 Computer Engineering uses GitHub to manage reviews of pull requests.

* If you have a trivial fix or improvement, go ahead and create a pull request,
  addressing (with `@...`) the maintainer of this repository (see
  [MAINTAINERS.md](./MAINTAINERS.md)) in the description of the pull request.

* In the description of the pull request make sure to write the version of all
  the software and version used besides the go packages, like:
  `OpenSSL`, the `Goreleaser` etc...

* Sign your work to certify that your changes were created by yourself or you
  have the right to submit it under our license. Read
  <https://developercertificate.org/> for all details and append your sign-off to
  every commit message like this:

        Signed-off-by: Firstname Lastname <example@example.com>

## Development building and running

Prerequisites:

* Rocky Linux 8.9 (minimal)

Bootstrap the OS:

```bash
# Add the epel repository
sudo yum install -y epel-release

# Update the OS packages
sudo yum update -y

# Add the needed packages
sudo yum install -y kernel-devel.x86_64 kernel-headers.x86_64 kernel-modules.x86_64 kernel-modules-extra.x86_64 kernel-tools.x86_64 kernel-tools-libs.x86_64 kernel-rpm-macros.noarch git htop wget

# Optional: (jq: useful for formatting and processing JSON output in the CLI)
sudo yum install -y vim jq

# Disable SELinux
sudo sed -i 's/^SELINUX=.*/SELINUX=disabled/' /etc/selinux/config

# Reboot the machine to ensure SELinux is disabled and to load any newly installed kernel modules
sudo reboot

# Optional: check the new SELinux status
getenforce
```

Now you are able to install the other tools on your OS respectiong this order by following their official guides:

* [OpenSSL](https://github.com/openssl/openssl/blob/master/INSTALL.md#installing-openssl)
* [xiRAID](https://xinnor.io/resources/xiraid-classic/)
* [Go compiler](https://golang.org/dl/)
* [xiraid_exporter](./README.md#xiraid_exporter-installation-and-usage)
* [Goreleaser](https://goreleaser.com/install/#go-install) - suggestion: install the binaries via `bash script`
* [Golangci-lint](https://golangci-lint.run/welcome/install/#local-installation)

If you want a stable installation please use these versions:

| **Software/compilators/libraries**   | **Version**       |
|--------------------------------------|-------------------|
| OpenSSL                              | `3.0.7`           |
| xiRAID                               | `4.1.0`           |
| Go compiler                          | `1.23.2`          |
| Goreleaser                           | `2.4.8`           |
| Golangci-lint                        | `1.62.0`          |

Viceversa feel free to test other version and to update them if needed but **remeber** to put
these changes on the pull request description.

## Collector Implementation Guidelines

The `xiraid_exporter` is not a general monitoring agent. Its sole purpose is only to
expose the xiRAID metrics.

A Collector may only use grpc calls to retrieve xiRAID metrics.
It may not require root privileges. Running external commands is 
not allowed for performance and reliability reasons.

The `xiraid_exporter` main purpose is to support all the xiRAID metrics provvided by the protobuffers
 developed by Xinnor and packaged with xiRAID software, using the grpc client-server architecture.