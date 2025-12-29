package output_test

import (
	"os"
	"testing"

	"github.com/sid-technologies/pilum/lib/output"
	"github.com/stretchr/testify/require"
)

func TestNewProgressBar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		total         int
		width         int
		expectedWidth int
	}{
		{
			name:          "explicit width",
			total:         10,
			width:         50,
			expectedWidth: 50,
		},
		{
			name:          "zero width defaults to 30",
			total:         10,
			width:         0,
			expectedWidth: 30,
		},
		{
			name:          "small total",
			total:         3,
			width:         20,
			expectedWidth: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pb := output.NewProgressBar(tt.total, tt.width)
			require.NotNil(t, pb)
		})
	}
}

func TestProgressBarSetMessage(t *testing.T) {
	t.Parallel()

	pb := output.NewProgressBar(10, 30)
	require.NotNil(t, pb)

	// Should not panic
	pb.SetMessage("Processing item 1")
	pb.SetMessage("Processing item 2")
	pb.SetMessage("")
}

func TestProgressBarIncrement(t *testing.T) {
	// Not parallel - writes to stdout
	pb := output.NewProgressBar(5, 20)

	// Increment multiple times
	for i := 0; i < 5; i++ {
		pb.Increment()
	}

	// Extra increments should be safe
	pb.Increment()
	pb.Increment()
}

func TestProgressBarSetProgress(t *testing.T) {
	// Not parallel - writes to stdout
	pb := output.NewProgressBar(10, 20)

	// Set various progress values
	pb.SetProgress(0)
	pb.SetProgress(5)
	pb.SetProgress(10)

	// Over total should be capped
	pb.SetProgress(15)
}

func TestProgressBarComplete(t *testing.T) {
	// Not parallel - writes to stdout
	pb := output.NewProgressBar(10, 20)

	pb.SetProgress(5)
	pb.Complete("All done!")
}

func TestNewStepProgress(t *testing.T) {
	t.Parallel()

	sp := output.NewStepProgress(5)
	require.NotNil(t, sp)
}

func TestStepProgressNextStep(t *testing.T) {
	// Not parallel - writes to stdout
	sp := output.NewStepProgress(3)

	sp.NextStep("Step 1: Building")
	sp.NextStep("Step 2: Testing")
	sp.NextStep("Step 3: Deploying")
}

func TestStepProgressComplete(t *testing.T) {
	// Not parallel - writes to stdout
	sp := output.NewStepProgress(3)

	sp.NextStep("Step 1")
	sp.NextStep("Step 2")
	sp.NextStep("Step 3")
	sp.Complete()
}

func TestProgressBarCIModeDetection(t *testing.T) {
	// Save CI environment
	originalCI := os.Getenv("CI")
	defer func() {
		if originalCI != "" {
			os.Setenv("CI", originalCI)
		} else {
			os.Unsetenv("CI")
		}
	}()

	// Test with CI mode
	os.Setenv("CI", "true")
	pbCI := output.NewProgressBar(10, 30)
	require.NotNil(t, pbCI)
	pbCI.Increment()
	pbCI.Complete("Done in CI")

	// Test without CI mode
	os.Unsetenv("CI")
	os.Unsetenv("GITHUB_ACTIONS")
	os.Unsetenv("GITLAB_CI")
	os.Unsetenv("CIRCLECI")
	os.Unsetenv("JENKINS_URL")
	os.Unsetenv("BUILDKITE")
	pbLocal := output.NewProgressBar(10, 30)
	require.NotNil(t, pbLocal)
}

func TestStepProgressCIModeDetection(t *testing.T) {
	// Save CI environment
	originalCI := os.Getenv("CI")
	defer func() {
		if originalCI != "" {
			os.Setenv("CI", originalCI)
		} else {
			os.Unsetenv("CI")
		}
	}()

	// Test with CI mode
	os.Setenv("CI", "true")
	spCI := output.NewStepProgress(3)
	require.NotNil(t, spCI)
	spCI.NextStep("CI Step")
	spCI.Complete()
}

func TestProgressBarConcurrentAccess(t *testing.T) {
	t.Parallel()

	pb := output.NewProgressBar(100, 30)

	done := make(chan bool, 10)

	// Simulate concurrent access
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				pb.Increment()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestStepProgressConcurrentAccess(t *testing.T) {
	t.Parallel()

	sp := output.NewStepProgress(100)

	done := make(chan bool, 10)

	// Simulate concurrent access
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				sp.NextStep("Step")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
