package cmd

import (
	"fmt"
	"os"

	"github.com/coby-spotim/servedir/internal/servedir"

	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "servedir",
	Short: "A simple file server",
	Long: `servedir is a simple and lazy file server that generates a
file server for the specified directory`,
	Run: servedir.Serve,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.servedir.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().IntP("port", "p", 8080, "port to listen on")
	rootCmd.Flags().StringP("dir", "d", ".", "directory to serve")
	rootCmd.Flags().BoolP("compression", "c", true, "whether or not to use brotli compression to compress the files")
	rootCmd.Flags().IntP("compression-level", "cl", 11, "level of brotli compression to use (0-11)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".test-cobra" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".servedir")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
