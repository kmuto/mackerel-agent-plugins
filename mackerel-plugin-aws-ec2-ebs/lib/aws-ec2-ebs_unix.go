package mpawsec2ebs

import (
	"syscall"
)

func init() {
	defaultSignal = syscall.SIGTERM
}
