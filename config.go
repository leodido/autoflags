package autoflags

import (
	"fmt"

	"github.com/spf13/viper"
)

func UseConfig(readWhen func() bool) (bool, string) {
	str := ""
	ret := false
	if readWhen == nil || readWhen() {
		// If a config file is found, read it in
		if err := viper.ReadInConfig(); err == nil {
			str = fmt.Sprintf("Using config file: %s", viper.ConfigFileUsed())
			ret = true
		} else {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// Config file not found, ignore...
				str = "Running without a configuration file"
			} else {
				// Config file was found but another error was produced
				str = fmt.Sprintf("Error running with config file: %s: %v", viper.ConfigFileUsed(), err)
			}
		}
	}

	return ret, str
}
