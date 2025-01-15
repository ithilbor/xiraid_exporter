# xiraid_exporter

Prometheus exporter for Xinnor xiRAID metrics, written in Go with pluggable metric collectors.

## Pre-requisites

To successfully install and use the exporter you need to:

1. Install `xiRAID`
2. Install `Openssl`
3. Change the default TLS certificate of `xiRAID`
4. Install the `xiraid_exporter`

Compatibility matrix:

| **xiraid_exporter**       | **xiRAID**       | **Openssl**        |
|---------------------------|------------------|--------------------|
| 1.0.0                     | 4.1.0            | 3.0.7              |

These version are verified so make sure to used them for your production installation.

### xiRAID installation

To install the xiRAID and configure it at your needs follow the [official documentation guide](https://xinnor.io/resources/xiraid-classic/) by selecting the correct version.

### Openssl installation

To install Openssl follow the [official documentation guide](https://github.com/openssl/openssl/blob/master/INSTALL.md#installing-openssl) by selecting the correct version.
Then in order to make the exporter work we need to chenge the certificats using these commands:

```bash
# Create new certificates for xiRAID
sudo openssl genrsa -out ca.key 2048
sudo openssl req -new -x509 -days 365 -key ca.key -subj /C=IL/ST=Haifa/L=Haifa/O=XINNOR/OU=IT/CN=localhost/emailAddress=request@xinnor.io -out ca-cert.crt
sudo openssl req -newkey rsa:2048 -nodes -keyout server-key.key -subj /C=IL/ST=Haifa/L=Haifa/O=XINNOR/OU=IT/CN=localhost/emailAddress=request@xinnor.io -out server-cert.csr
sudo openssl x509 -req -extfile <(printf "subjectAltName=DNS:localhost,DNS:*.e4red,IP:0.0.0.0") -days 365 -in server-cert.csr -CA ca-cert.crt -CAkey ca.key -CAcreateserial -out server-crt.crt

# Restart xiRAID services to use new certificates
systemctl restart xiraid.target

# Optional: check that the CLI is working
xicli raid show
```

## xiraid_exporter installation and usage

To install the xiraid exporter follow these steps:

```bash
# NOTE: Replace the placeholders <VERSION>, <OS>, and <ARCH> with the once aviable.
wget https://github.com/E4-Computer-Engineering/xiraid_exporter/releases/download/v<VERSION>/xiraid_exporter_<OS>_<ARCH>.tar.gz
tar xvfz xiraid_exporter_*.tar.gz
./xiraid_exporter
```

To check that the metrics are exported:

```bash
curl http://localhost:9505/metrics
```

The output of the curl command will be like this:

```bash
# HELP go_gc_duration_seconds A summary of the GC invocation durations.
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0"} 3.8996e-05
go_gc_duration_seconds{quantile="0.25"} 4.5926e-05
go_gc_duration_seconds{quantile="0.5"} 5.846e-05
# etc..
```

The `xiraid_exporter` will listens on HTTP port 9505 by default. See the `--help` output for more options.

## Known installation problems

* Since the xiRAID default TLS certificates are not compatible with the grpc TLS client creation bacause they use an old standard
  it's mandatory to create new certificates as mentioned above.

## Collectors

To let the user decide wich data expose different collectors are defined to manage this task.
Tipically a collector is defined for each type of supported protobuffer method call.

Collectors are enabled by providing a `--collector.<name>` flag.
Collectors that are enabled by default can be disabled by providing a `--no-collector.<name>` flag.

This is the list of all the collectors in this exporter:

| **Collector** | **Default** | **Enable Flag** | **Disable Flag** |
| --- | --- | --- | --- |
| license_show | enable | --collector.xiraid_license_show | --no-collector.xiraid_license_show |
| raid_show  | enable | --collector.xiraid_raid_show  | --no-collector.xiraid_raid_show |

Description of the collectors:

| **Collector** | **Description** |
| --- | --- |
| license_show | Shows the license informations like: `hwkey`, `disk_in_use`, `license_status` etc.. |
| raid_show  | Shows the informations releted to RAIDs like: `riad_name`, `raid_uuid`, `devices`, `device_status`, `raid_status`, etc.. |


## Contributing

If you want to contribute to this reposiotry pleaase see the [CONTRIBUTIng.md](./CONTRIBUTING.md) file for details.

## Code of conduct

This project relies on the Contributor Covenant Code of Conduct. See the [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md) file for details.

## License
This project is licensed under the Apache License, Version 2.0. See the [LICENSE.md](./LICENSE) file for details.

## Authors

This software is developed by Federico Ferrari, <federico.ferr25@gmail.com>, [GitHub](https://github.com/IronCub3), for [E4 Computer Engineering](https://www.e4company.com/).

## Credits

This project includes some logic adapted from the [node_exporter](https://github.com/prometheus/node_exporter) project under the Apache License 2.0. Special thanks to the Prometheus Community for their work.
