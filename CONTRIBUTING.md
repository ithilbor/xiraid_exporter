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
sudo yum install -y kernel-devel.x86_64 kernel-headers.x86_64 kernel-modules.x86_64 kernel-modules-extra.x86_64 kernel-tools.x86_64 kernel-tools-libs.x86_64 kernel-rpm-macros.noarch git wget

# Optional: adding other usefull packages
sudo yum install -y vim jq htop

# Disable SELinux
sudo sed -i 's/^SELINUX=.*/SELINUX=disabled/' /etc/selinux/config

# Reboot the machine to ensure SELinux is disabled and to load any newly installed kernel modules
sudo reboot

# Optional: check the new SELinux status
getenforce
```

Now you are able to install the other tools on your OS respectiong this order by following their official guides:

* [OpenSSL](https://github.com/openssl/openssl/blob/master/INSTALL.md#installing-openssl) - remember to create the new certificats as shown in the [README.md](./README.md#openssl-installation)
* [xiRAID](https://xinnor.io/resources/xiraid-classic/) - check also the [README.md](./README.md#xiraid-installation)
* [Go compiler](https://golang.org/dl/)
* [xiraid_exporter](./README.md#xiraid_exporter-installation-and-usage)
* [Goreleaser](https://goreleaser.com/install/#go-install) - suggestion: install the binaries via `bash script`
* [Golangci-lint](https://golangci-lint.run/welcome/install/#local-installation)
* [ProtocolBuffer](https://protobuf.dev/installation/) - suggestion: install the precompiled binaries from the github repository

If you want a stable installation please use these versions:

| **Software/compilators/libraries**   | **Version**       |
|--------------------------------------|-------------------|
| xiraid_exporter                      | `1.1.0`           |
| OpenSSL                              | `3.0.7`           |
| xiRAID                               | `4.2.0`           |
| Go compiler                          | `1.23.2`          |
| Goreleaser                           | `2.7.0`           |
| Golangci-lint                        | `1.62.0`          |
| ProtocolBuffer                       | `29.3.0`          |

Viceversa feel free to test other version and to update them if needed but **remeber** to put
these changes on the pull request description.

If you want to use a new version of xiRAID that introduce chenges to the protocol buffers and you need to regenerate the GO code use these commands:
**NB: sobstitute the <...> with the appropiate value**

```bash
# Check where are the protocol buffers ( tiplically they are in: /var/lib/xraid/gRPC/protobuf )
rpm -ql <xiraid-package> | grep -E '^*.proto$'

# Add in each of these files after the directve `syntax = "proto3"` add the line: `option go_package = "github.com/ironcub3/xiraid_exporter`
sudo sed -i '/^syntax = "proto3";$/a\
option go_package = "github.com/ironcub3/xiraid_exporter";' </path/to/xiraid/proto/buffer>/*.proto

# Check if the line was added
grep -h '^option' </path/to/xiraid/proto/buffer>/*.proto

# Go into the main folder of the GitHub project and generate the new GO code from the protocol buffer
cd </path/to/xiraid/github/project>
protoc -I</path/to/xiraid/proto/buffer> --go_out=./protos --go_opt=paths=source_relative --go-grpc_out=./protos --go-grpc_opt=paths=source_relative </path/to/xiraid/proto/buffer>/*.proto
```

## Collector Implementation Guidelines

The `xiraid_exporter` is not a general monitoring agent. Its sole purpose is only to
expose the xiRAID metrics.

A Collector may only use grpc calls to retrieve xiRAID metrics.
It may not require root privileges. Running external commands is 
not allowed for performance and reliability reasons.

The `xiraid_exporter` main purpose is to support all the xiRAID metrics provvided by the protobuffers
 developed by Xinnor and packaged with xiRAID software, using the grpc client-server architecture.