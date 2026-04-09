package output_test

import (
	"testing"

	"github.com/smichalabs/britivectl/internal/output"
)

func TestNewSpinner(t *testing.T) {
	s := output.NewSpinner("testing...")
	if s == nil {
		t.Fatal("NewSpinner() returned nil")
	}
}

func TestSpinner_Success(t *testing.T) {
	s := output.NewSpinner("testing...")
	s.Start()
	s.Success("done")
}

func TestSpinner_Fail(t *testing.T) {
	s := output.NewSpinner("testing...")
	s.Start()
	s.Fail("failed")
}

func TestSpinner_Stop(t *testing.T) {
	s := output.NewSpinner("testing...")
	s.Start()
	s.Stop()
}

func TestSpinner_SuccessWithoutStart(t *testing.T) {
	s := output.NewSpinner("testing...")
	s.Success("done")
}

func TestSpinner_FailWithoutStart(t *testing.T) {
	s := output.NewSpinner("testing...")
	s.Fail("failed")
}
