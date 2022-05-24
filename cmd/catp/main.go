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
	"io"
	"net"
	"os"

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

func main() {
	// nolint: exhaustivestruct
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	var (
		inAdrress  string
		outAddress string
		verbosity  string
	)

	// nolint: exhaustivestruct
	rootCmd := &cobra.Command{
		Use:     name,
		Example: fmt.Sprintf("%s --in myserver:1961", name),
		Short:   "Cat Pipe : pipe stdin or stdout to a cat balancer server",
		Version: fmt.Sprintf(`%v (commit=%v date=%v by=%v)
Copyright (C) 2021 CGI France
License GPLv3: GNU GPL version 3 <https://gnu.org/licenses/gpl.html>.
This is free software: you are free to change and redistribute it.
There is NO WARRANTY, to the extent permitted by law.`, version, commit, buildDate, builtBy),

		RunE: func(cmd *cobra.Command, args []string) error {
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

			return run(cmd, inAdrress, outAddress)
		},
	}

	rootCmd.PersistentFlags().StringVarP(&inAdrress, "in", "i", "",
		"input server's address (empty for stdin by default)")
	rootCmd.PersistentFlags().StringVarP(&outAddress, "out", "o", "",
		"output server's address (empty for stdout by default)")
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

func run(cmd *cobra.Command, in string, out string) error {
	streamIn := cmd.InOrStdin()
	streamOut := cmd.OutOrStdout()

	if in != "" {
		conIn, err := net.Dial("tcp", in)
		if err != nil {
			return fmt.Errorf("%w", err)
		}
		defer conIn.Close()

		streamIn = conIn
	}

	if out != "" {
		conOut, err := net.Dial("tcp", out)
		if err != nil {
			return fmt.Errorf("%w", err)
		}
		defer conOut.Close()

		streamOut = conOut
	}

	_, err := io.Copy(streamOut, streamIn)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}
