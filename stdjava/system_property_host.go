package stdjava

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

type systemPropertyHostProfile struct {
	values      map[string]string
	unavailable map[string]string
}

// Snapshot at package startup, before application property writes or chdir.
// Reference-JDK OS_NAME/OS_ARCH fields deliberately do not enter this snapshot.
var systemPropertyStartupHost = captureSystemPropertyHost()

func captureSystemPropertyHost() systemPropertyHostProfile {
	result := systemPropertyHostProfile{values: map[string]string{
		"file.separator": string(os.PathSeparator), "path.separator": string(os.PathListSeparator), "line.separator": "\n",
	}, unavailable: map[string]string{}}
	if runtime.GOOS == "windows" {
		result.values["line.separator"] = "\r\n"
	}
	if cwd, err := os.Getwd(); err == nil {
		if physical, err := filepath.EvalSymlinks(cwd); err == nil {
			result.values["user.dir"] = physical
		} else {
			result.unavailable["user.dir"] = err.Error()
		}
	} else {
		result.unavailable["user.dir"] = err.Error()
	}
	if current, err := user.Current(); err == nil {
		result.values["user.name"], result.values["user.home"] = current.Username, current.HomeDir
	} else {
		result.unavailable["user.name"] = err.Error()
		result.unavailable["user.home"] = err.Error()
	}
	identity, err := systemPropertyHostIdentity()
	for key, value := range identity {
		result.values[key] = value
	}
	for _, key := range []string{"os.name", "os.arch", "os.version"} {
		if _, ok := result.values[key]; !ok {
			if err != nil {
				result.unavailable[key] = err.Error()
			} else {
				result.unavailable[key] = "native host identity is unavailable"
			}
		}
	}
	return result
}
