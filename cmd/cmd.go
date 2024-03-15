package cmd

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	versionCmd "github.com/plumber-cd/argocd-applicationset-namespaces-generator-plugin/cmd/version"

	serverCmd "github.com/plumber-cd/argocd-applicationset-namespaces-generator-plugin/cmd/server"
)

var rootCmd = &cobra.Command{
	Use: "argocd-applicationset-namespaces-generator-plugin",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("no command specified")
	},
}

func Exec() {
	if err := rootCmd.Execute(); err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			os.Exit(exitErr.ExitCode())
		}

		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().IntP("verbosity", "v", 0, "Set verbosity level")
	rootCmd.PersistentFlags().String("log-format", "json", "Set log output (json, text)")

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		log.Panic(err)
	}

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			log.Panic(err)
		}
	}

	rootCmd.AddCommand(versionCmd.Cmd)
	rootCmd.AddCommand(serverCmd.Cmd)
}

func initConfig() {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.SetEnvPrefix("ARGOCD_APPLICATIONSET_NAMESPACES_PLUGIN")

	format := viper.GetString("log-format")
	level := slog.Level(-viper.GetInt("verbosity"))

	handlerOptions := &slog.HandlerOptions{
		Level:     level,
		AddSource: level <= slog.LevelDebug,
	}
	var handler slog.Handler
	switch format {
	case "json":
		handler = slog.NewJSONHandler(os.Stderr, handlerOptions)
	case "text":
		suppress := func(
			next func([]string, slog.Attr) slog.Attr,
		) func([]string, slog.Attr) slog.Attr {
			return func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey {
					return slog.Attr{}
				}
				if next == nil {
					return a
				}
				return next(groups, a)
			}
		}
		handlerOptions.ReplaceAttr = suppress(handlerOptions.ReplaceAttr)
		handler = slog.NewTextHandler(os.Stdout, handlerOptions)
	default:
		log.Panicf("unknown log format: %s", format)
	}

	slog.SetDefault(slog.New(handler))
}
