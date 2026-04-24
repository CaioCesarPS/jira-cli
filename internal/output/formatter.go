package output

import (
	"encoding/json"
	"fmt"
	"os"
)

type Result struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data,omitempty"`
	Error      string      `json:"error,omitempty"`
	StatusCode int         `json:"status_code,omitempty"`
}

var JSONMode bool

func Print(msg string) {
	fmt.Println(msg)
}

func PrintResult(data interface{}, humanMsg string) {
	if JSONMode {
		printJSON(Result{Success: true, Data: data})
		return
	}
	fmt.Println(humanMsg)
}

func PrintError(err error, statusCode int) {
	if JSONMode {
		printJSON(Result{Success: false, Error: err.Error(), StatusCode: statusCode})
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
	}
}

func printJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
