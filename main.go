// Copyright 2012-2022 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

//go:generate go run server/errors_gen.go

import (
	"flag"
	"fmt"
	"memphis-broker/analytics"
	"memphis-broker/background_tasks"
	"memphis-broker/db"
	"memphis-broker/handlers"
	"memphis-broker/server"
	"os"
	"sync"

	"go.uber.org/automaxprocs/maxprocs"
)

var usageStr = `
Usage: nats-server [options]

Server Options:
    -a, --addr <host>                Bind to host address (default: 0.0.0.0)
    -p, --port <port>                Use port for clients (default: 4222)
    -n, --name <server_name>         Server name (default: auto)
    -P, --pid <file>                 File to store PID
    -m, --http_port <port>           Use port for http monitoring
    -ms,--https_port <port>          Use port for https monitoring
    -c, --config <file>              Configuration file
    -t                               Test configuration and exit
    -sl,--signal <signal>[=<pid>]    Send signal to nats-server process (stop, quit, reopen, reload)
                                     <pid> can be either a PID (e.g. 1) or the path to a PID file (e.g. /var/run/nats-server.pid)
        --client_advertise <string>  Client URL to advertise to other servers

Logging Options:
    -l, --log <file>                 File to redirect log output
    -T, --logtime                    Timestamp log entries (default: true)
    -s, --syslog                     Log to syslog or windows event log
    -r, --remote_syslog <addr>       Syslog server addr (udp://localhost:514)
    -D, --debug                      Enable debugging output
    -V, --trace                      Trace the raw protocol
    -VV                              Verbose trace (traces system account as well)
    -DV                              Debug and trace
    -DVV                             Debug and verbose trace (traces system account as well)

JetStream Options:
    -js, --jetstream                 Enable JetStream functionality.
    -sd, --store_dir <dir>           Set the storage directory.

Authorization Options:
        --user <user>                User required for connections
        --pass <password>            Password required for connections
        --auth <token>               Authorization token required for connections

TLS Options:
        --tls                        Enable TLS, do not verify clients (default: false)
        --tlscert <file>             Server certificate file
        --tlskey <file>              Private key for server certificate
        --tlsverify                  Enable TLS, verify client certificates
        --tlscacert <file>           Client certificate CA for verification

Cluster Options:
        --routes <rurl-1, rurl-2>    Routes to solicit and connect
        --cluster <cluster-url>      Cluster URL for solicited routes
        --cluster_name <string>      Cluster Name, if not set one will be dynamically generated
        --no_advertise <bool>        Do not advertise known cluster information to clients
        --cluster_advertise <string> Cluster URL to advertise to other servers
        --connect_retries <number>   For implicit routes, number of connect retries

Common Options:
    -h, --help                       Show this message
    -v, --version                    Show version
        --help_tls                   TLS help
`

// usage will print out the flag options for the server.
func usage() {
	fmt.Printf("%s\n", usageStr)
	os.Exit(0)
}

func handleError(s *server.Server, message string, err error) {
	if err != nil {
		s.Errorf(message + " " + err.Error())
	}
}

func runMemphis(s *server.Server) {

	err := db.InitializeDbConnection(s)
	handleError(s, "Failed initializing db connection: ", err)

	err = analytics.InitializeAnalytics()
	handleError(s, "Failed initializing analytics: ", err)

	handlers.InitializeHandlers(s)

	err = handlers.CreateRootUserOnFirstSystemLoad()
	handleError(s, "Failed to create root user: ", err)

	background_tasks.InitializeZombieResources(s)

	defer db.Close()

	// defer broker.Close()
	defer analytics.Close()

	wg := new(sync.WaitGroup)
	wg.Add(4)

	// go tcp_server.InitializeTcpServer(wg)
	// go http_server.InitializeHttpServer(wg, s)
	go background_tasks.KillZombieResources(wg)
	// go background_tasks.ListenForPoisonMessages()

	var env string
	if os.Getenv("DOCKER_ENV") != "" {
		env = "Docker"
		s.Noticef("\n**********\n\nDashboard: http://localhost:9000\nMemphis broker: localhost:5555 (Management Port) / 7766 (Data Port) / 6666 (TCP Port)\nUI/CLI root username - root\nUI/CLI root password - memphis\nSDK root connection token - memphis  \n\n**********")
	} else {
		env = "K8S"
	}

	s.Noticef("Memphis broker is up and running, ENV: " + env)
	wg.Wait()
}

func main() {
	exe := "nats-server"

	// Create a FlagSet and sets the usage
	fs := flag.NewFlagSet(exe, flag.ExitOnError)
	fs.Usage = usage

	// Configure the options from the flags/config file
	opts, err := server.ConfigureOptions(fs, os.Args[1:],
		server.PrintServerAndExit,
		fs.Usage,
		server.PrintTLSHelpAndDie)
	if err != nil {
		server.PrintAndDie(fmt.Sprintf("%s: %s", exe, err))
	} else if opts.CheckConfig {
		fmt.Fprintf(os.Stderr, "%s: configuration file %s is valid\n", exe, opts.ConfigFile)
		os.Exit(0)
	}

	// Create the server with appropriate options.
	s, err := server.NewServer(opts)
	if err != nil {
		server.PrintAndDie(fmt.Sprintf("%s: %s", exe, err))
	}

	// Configure the logger based on the flags
	s.ConfigureLogger()

	// Start things up. Block here until done.
	if err := server.Run(s); err != nil {
		server.PrintAndDie(err.Error())
	}

	// Adjust MAXPROCS if running under linux/cgroups quotas.
	undo, err := maxprocs.Set(maxprocs.Logger(s.Debugf))
	if err != nil {
		server.PrintAndDie(fmt.Sprintf("failed to set GOMAXPROCS: %v", err))
	}
	defer undo()
	runMemphis(s)
	s.WaitForShutdown()
}
