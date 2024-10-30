package plugin

import (
	"errors"
	"fmt"
	"os"
)

const pluginDir = "." // TODO: Make specific default directory (~/.cofide/plugins)

func PluginExists(pluginPath string) (bool, error) {
	if _, err := os.Stat(pluginPath); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func GetPluginPath(pluginName string) string {
	return fmt.Sprintf("%s/%s", pluginDir, pluginName)
}
