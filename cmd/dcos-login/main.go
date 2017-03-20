package main

import (
	dcoslogin "github.com/Originate/dcos-login"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	Version string

	options = dcoslogin.Options{
		ClusterURL: kingpin.Flag("cluster-url", "URL of the DC/OS master(s) (e.g https://example.com)").Envar("CLUSTER_URL").Required().String(),

		Username: kingpin.Flag("username", "Github username used for logging in").Short('u').Envar("GH_USERNAME").Required().String(),
		Password: kingpin.Flag("password", "Github password used for logging in").Short('p').Envar("GH_PASSWORD").Required().String(),

		AllowInsecureTLS: kingpin.Flag("insecure", "Set this when targeting a cluster with a self-signed certificate").Short('k').Default("false").Bool(),
	}

	debug = kingpin.Flag("debug", "Enable debugging mode. This *WILL* print credentials.").Default("false").Bool()
)

func init() {
	kingpin.CommandLine.HelpFlag.Short('h')
}

func main() {
	kingpin.Version(Version)
	kingpin.Parse()

	if *debug {
		dcoslogin.Debug = true
	}

	if err := dcoslogin.Login(&options); err != nil {
		kingpin.Fatalf("Unexpected error:\n %v", err)
	}
}
