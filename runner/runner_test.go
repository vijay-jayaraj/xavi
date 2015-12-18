package runner

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/xtracdev/xavi/env"
	"github.com/xtracdev/xavi/plugin"
	"github.com/xtracdev/xavi/plugin/logging"
	"io/ioutil"
	"os"
	"testing"
)

func registerLoggingPlugin() {
	plugin.RegisterWrapperFactory("Logging", logging.NewLoggingWrapper)
}

func TestGetKVStoreEndpointFromEnvironmentVariable(t *testing.T) {
	os.Setenv(env.KVStoreURL, "something")
	assert.Equal(t, getKVStoreEndpoint(), "something")
}

func TestSetup(t *testing.T) {
	tempFile, err := ioutil.TempFile("./", "xavitest")
	assert.Nil(t, err)

	currentDir, err := os.Getwd()
	assert.Nil(t, err)
	fileURL := fmt.Sprintf("file:///%s/%s", currentDir, tempFile.Name())
	println(fileURL)
	os.Setenv(env.KVStoreURL, fileURL)

	kvs := setupXAVIEnvironment(registerLoggingPlugin)
	assert.NotNil(t, kvs)
	assert.True(t, plugin.RegistryContains("Logging"))

	tempFile.Close()
	os.Remove(tempFile.Name())
}

func TestSetLogLevels(t *testing.T) {
	os.Setenv(env.LoggingLevel, "debug")
	setLoggingLevel()
	assert.Equal(t, log.DebugLevel, log.GetLevel())

	os.Setenv(env.LoggingLevel, "warn")
	setLoggingLevel()
	assert.Equal(t, log.WarnLevel, log.GetLevel())

	os.Setenv(env.LoggingLevel, "error")
	setLoggingLevel()
	assert.Equal(t, log.ErrorLevel, log.GetLevel())

	os.Setenv(env.LoggingLevel, "info")
	setLoggingLevel()
	assert.Equal(t, log.InfoLevel, log.GetLevel())

	os.Setenv(env.LoggingLevel, "")
	setLoggingLevel()
	assert.Equal(t, log.InfoLevel, log.GetLevel())
}

func TestFireUpPProf(t *testing.T) {
	os.Setenv(env.PProfEndpoint, "")
	fired := fireUpPProf()
	assert.False(t, fired)

	os.Setenv(env.PProfEndpoint, "localhost:80")
	fired = fireUpPProf()
	assert.True(t, fired)
}
