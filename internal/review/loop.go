package review

import (
	"fmt"

	"github.com/zon/ralph/internal/architecture"
)

type LoopIteration struct {
	FunctionName string
	FunctionPath string
}

func ExpandLoop(loopType string, archPath string) ([]LoopIteration, error) {
	switch loopType {
	case "domain-function":
		return expandDomainFunctionLoop(archPath)
	default:
		return nil, fmt.Errorf("unknown loop type: %q", loopType)
	}
}

func expandDomainFunctionLoop(archPath string) ([]LoopIteration, error) {
	arch, err := architecture.Load(archPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load architecture: %w", err)
	}

	seen := make(map[string]bool)
	var iterations []LoopIteration

	for _, app := range arch.Apps {
		for _, feature := range app.Features {
			for _, fn := range feature.Functions {
				key := fn.File + "|" + fn.Name
				if seen[key] {
					continue
				}
				seen[key] = true
				iterations = append(iterations, LoopIteration{
					FunctionName: fn.Name,
					FunctionPath: fn.File,
				})
			}
		}
	}

	return iterations, nil
}
