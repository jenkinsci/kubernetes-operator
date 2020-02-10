package log

import (
	"log"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// Log represents global logger
var Log = logf.Log.WithName("controller-jenkins")

// Debug indicates that debug level is set
var Debug bool

const (
	// VWarn defines warning log level
	VWarn = -1
	// VDebug defines debug log level
	VDebug = 1
)

func zapLogger(debug bool) logr.Logger {
	var zapLog *zap.Logger
	var err error
	zapLogCfg := zap.NewDevelopmentConfig()
	if debug {
		zapLogCfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		zapLogCfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	zapLog, err = zapLogCfg.Build(zap.AddStacktrace(zap.DPanicLevel), zap.AddCallerSkip(1))
	// who watches the watchmen?
	fatalIfErr(err, log.Fatalf)
	return zapr.NewLogger(zapLog)
}

func fatalIfErr(err error, f func(format string, v ...interface{})) {
	if err != nil {
		f("unable to construct the logger: %v", err)
	}
}

// SetupLogger setups global logger
func SetupLogger(debug bool) {
	Debug = debug
	logf.SetLogger(zapLogger(debug))
	Log = logf.Log.WithName("controller-jenkins")
}
