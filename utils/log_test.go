package utils

import (
	"testing"
)

func TestTestingLogger(t *testing.T) {
	logger := NewTestLogger(t)

	logger.Infof("diumaodalaji = %s", "test")

	objectLogger := ObjectLogger{
		Log:        logger,
		ObjectName: "log_test",
	}

	logger = objectLogger.Logger("TestTestingLogger")

	logger.Infof("diumaodalaji = %s", "test")
}
