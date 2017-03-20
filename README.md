# dcos-login [![CircleCI](https://circleci.com/gh/Originate/dcos-login.svg?style=svg&circle-token=95537e153102c2e81b3a5b8c72a2e68a0502776a)](https://circleci.com/gh/Originate/dcos-login)

Automates unattended login for the [DC/OS](https://dcos.io/) Community Edition

## Installation

If you just want the latest version, run `go get -u github.com/Originate/dcos-login/cmd/dcos-login`

Otherwise (or if you don't have [Go](https://golang.org/doc/install) installed) you'll find binaries for your
platform on the [release](https://github.com/Originate/dcos-login/releases) page.

## Usage

```
usage: dcos-login --cluster-url=CLUSTER-URL --username=USERNAME --password=PASSWORD [<flags>]

Flags:
  -h, --help                     Show context-sensitive help (also try --help-long and --help-man).
      --cluster-url=CLUSTER-URL  URL of the DC/OS master(s) (e.g https://example.com)
  -u, --username=USERNAME        Github username used for logging in
  -p, --password=PASSWORD        Github password used for logging in
  -k, --insecure                 Set this when targeting a cluster with a self-signed certificate
      --debug                    Enable debugging mode. This *WILL* print credentials.
      --version                  Show application version.
```

Additionally, the following environment variables can be used:

- `$CLUSTER_URL` for `--cluster-url`
- `$GH_USERNAME` for `--username`
- `$GH_PASSWORD` for `--password`

You can authenticate the DC/OS CLI using:
```shell
export CLUSTER_URL="https://my.dcos.cluster.com"
export GH_USERNAME="myorg-ci"
export GH_PASSWORD="secret"

dcos config set core.dcos_acs_token "$(dcos-login)"
```

If your configuration file for the CLI is in a non-standard location (ie. several clusters), you can override
using the following:
```shell
export DCOS_CONFIG="path/to/dcos.toml"
```
