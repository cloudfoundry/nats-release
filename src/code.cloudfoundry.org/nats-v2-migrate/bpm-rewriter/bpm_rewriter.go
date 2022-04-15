package bpm_rewriter

//go:generate counterfeiter -generate
import (
	"fmt"
	"io/ioutil"
)

//go:generate counterfeiter -o ../fakes/bpm_rewriter.go --fake-name Rewriter . Rewriter
type Rewriter interface {
	Rewrite(string, string) error
}

type BPMRewriter struct{}

func (*BPMRewriter) Rewrite(sourcePath string, destinationPath string) error {
	bytesRead, err := ioutil.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("Error reading source file: %v", err)
	}

	err = ioutil.WriteFile(destinationPath, bytesRead, 0644)
	if err != nil {
		return fmt.Errorf("Error writing destination file: %v", err)
	}

	return nil
}
