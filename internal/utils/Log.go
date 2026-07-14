package utils

import (
	"fmt"
	"path/filepath"
	"runtime"
)

func LogError(msg string) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		fmt.Println(msg)
		return
	}

	fmt.Printf("%s:%d: %s\n", filepath.Base(file), line, msg)
}
