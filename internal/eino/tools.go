package eino

import (
	"context"
	"os"
	"os/exec"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type readFileInput struct {
	Path string `json:"path" jsonschema:"required" jsonschema_description:"Path to the file to read"`
}

type writeFileInput struct {
	Path    string `json:"path" jsonschema:"required" jsonschema_description:"Path to the file to write"`
	Content string `json:"content" jsonschema:"required" jsonschema_description:"Content to write to the file"`
}

type bashInput struct {
	Command string `json:"command" jsonschema:"required" jsonschema_description:"Shell command to execute"`
}

func newReadFileTool() (tool.InvokableTool, error) {
	return utils.InferTool[readFileInput, string](
		"read",
		"Read the contents of a file at the given path",
		func(ctx context.Context, input readFileInput) (string, error) {
			data, err := os.ReadFile(input.Path)
			if err != nil {
				return "", err
			}
			return string(data), nil
		},
	)
}

func newWriteFileTool() (tool.InvokableTool, error) {
	return utils.InferTool[writeFileInput, string](
		"write",
		"Write content to a file at the given path, creating it if necessary",
		func(ctx context.Context, input writeFileInput) (string, error) {
			err := os.WriteFile(input.Path, []byte(input.Content), 0644)
			if err != nil {
				return "", err
			}
			return "ok", nil
		},
	)
}

func newBashTool() (tool.InvokableTool, error) {
	return utils.InferTool[bashInput, string](
		"bash",
		"Execute a shell command via bash -c and return the combined stdout and stderr",
		func(ctx context.Context, input bashInput) (string, error) {
			cmd := exec.Command("bash", "-c", input.Command)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return string(output), err
			}
			return string(output), nil
		},
	)
}

func CodingTools() []tool.BaseTool {
	readFile, err := newReadFileTool()
	if err != nil {
		panic(err)
	}
	writeFile, err := newWriteFileTool()
	if err != nil {
		panic(err)
	}
	bash, err := newBashTool()
	if err != nil {
		panic(err)
	}
	return []tool.BaseTool{readFile, writeFile, bash}
}
