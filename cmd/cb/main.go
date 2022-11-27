// Copyright (C) 2021 CGI France
//
// This file is part of Cat Balancer.
//
// Cat Balancer is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Cat Balancer is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Cat Balancer.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/cgi-fr/cat-balancer/pkg/balancer"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// Provisioned by ldflags
// nolint: gochecknoglobals
var (
	name      string
	version   string
	commit    string
	buildDate string
	builtBy   string
)

// nolint:  funlen
func main() {
	// nolint: exhaustivestruct
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	var (
		producerPort     int
		consumerPort     int
		verbosity        string
		producerPoolSize int
		consumerPoolSize int
		interval         time.Duration
	)

	// nolint: exhaustivestruct
	rootCmd := &cobra.Command{
		Use:   "cb",
		Short: "Cat Balancer : line based load balancer for net cat",
		Version: fmt.Sprintf(`%v (commit=%v date=%v by=%v)
Copyright (C) 2021 CGI France
License GPLv3: GNU GPL version 3 <https://gnu.org/licenses/gpl.html>.
This is free software: you are free to change and redistribute it.
There is NO WARRANTY, to the extent permitted by law.`, version, commit, buildDate, builtBy),
		Run: func(cmd *cobra.Command, args []string) {
			switch verbosity {
			case "trace", "5":
				zerolog.SetGlobalLevel(zerolog.TraceLevel)
				log.Info().Msg("Logger level set to trace")
			case "debug", "4":
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
				log.Info().Msg("Logger level set to debug")
			case "info", "3":
				zerolog.SetGlobalLevel(zerolog.InfoLevel)
				log.Info().Msg("Logger level set to info")
			case "warn", "2":
				zerolog.SetGlobalLevel(zerolog.WarnLevel)
			case "error", "1":
				zerolog.SetGlobalLevel(zerolog.ErrorLevel)
			default:
				zerolog.SetGlobalLevel(zerolog.Disabled)
			}

			run(producerPort, consumerPort, producerPoolSize, consumerPoolSize, interval)
		},
	}
	// nolint: gomnd
	rootCmd.PersistentFlags().IntVarP(&producerPort, "producer-port", "p", 1691, "producer listen port")
	// nolint: gomnd
	rootCmd.PersistentFlags().IntVarP(&consumerPort, "consumer-port", "c", 1961, "consumer listen port")
	rootCmd.PersistentFlags().IntVarP(&producerPoolSize, "producers-pool-size", "P", 0, "Producers expected")
	rootCmd.PersistentFlags().IntVarP(&consumerPoolSize, "consumers-pool-size", "C", 0, "Consumers expected")
	rootCmd.PersistentFlags().
		DurationVarP(&interval,
			"interval",
			"i",
			time.Second,
			"Wait SEC seconds between updates. The default is to update every second.",
		)

	rootCmd.PersistentFlags().
		StringVarP(&verbosity,
			"verbosity",
			"v",
			"info",
			"set level of log verbosity : none (0), error (1), warn (2), info (3), debug (4), trace (5)",
		)

	if err := rootCmd.Execute(); err != nil {
		log.Err(err).Msg("Error when executing command")
		os.Exit(1)
	}
}

func run(producerPort int, consumerPort int, producersPoolSize int, consumersPoolSize int, interval time.Duration) {
	log.Info().Msgf("%v %v (commit=%v date=%v by=%v)", name, version, commit, buildDate, builtBy)

	balancer.New(
		"tcp", fmt.Sprintf(":%d", producerPort),
		"tcp", fmt.Sprintf(":%d", consumerPort),
		producersPoolSize, consumersPoolSize, interval).Start()
}
