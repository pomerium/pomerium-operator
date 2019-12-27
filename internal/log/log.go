package log

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// L is the global logr instance
var L logr.Logger
var level zap.AtomicLevel = zap.NewAtomicLevel()
var z *zap.Logger

func init() {
	level.SetLevel(zap.InfoLevel)
	z = kzap.NewRaw(setup())
	L = zapr.NewLogger(z).WithName("pomerium-operator")
}

func setup() kzap.Opts {
	return func(o *kzap.Options) {
		o.Level = &level
	}
}

// Debug sets the log output level to debug for the global logr L
func Debug() {
	level.SetLevel(zap.DebugLevel)
}
