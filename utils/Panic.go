package utils

import (
	"fmt"
	"path/filepath"
	"runtime"
)

func Panic(msg string) {
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		panic(msg)
	}

	fn := runtime.FuncForPC(pc)

	panic(fmt.Sprintf(
		"%s:%d (%s): %s",
		filepath.Base(file),
		line,
		fn.Name(),
		msg,
	))
}
