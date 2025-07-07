package service

import (
	"log/slog"
	"testing"
)

func TestLogger(t *testing.T) {
	//logger = common.GetLogger()
	//logger.Debug("hello %s", "Debug")
	//logger.Info("hello %s", "Info")
	//logger.Warning("hello %s", "Warning")
	//logger.Error("hello %s", "Error")
	//t.Log("End")
	initLogger()
	slog.Info("debug")
	slog.Warn("warn")
	slog.Error("error")
}
