package main

import (
	"fmt"

	"github.com/imajaygiri/compiler/internal/lexer"
	"github.com/imajaygiri/compiler/internal/utils"
	"os"
)

func main() {
	source, err := os.ReadFile("example/01.aj")
	if err != nil {
		utils.LogError(fmt.Sprintf("Error Reading file, Error: %v\n", err))
	}

	fmt.Println(string(source))
	tokens := lexer.Parse(string(source), "01.aj")
	for _, t := range tokens {
		t.Debug()
	}
}
