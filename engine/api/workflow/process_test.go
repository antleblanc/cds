package workflow

import (
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestComputeRunStatus(t *testing.T) {
	runStatus := &statusCounter{}
	computeRunStatus(sdk.StatusSuccess.String(), runStatus)

	assert.Equal(t, 1, runStatus.success)
	assert.Equal(t, 0, runStatus.building)
	assert.Equal(t, 0, runStatus.failed)
	assert.Equal(t, 0, runStatus.stoppped)

	computeRunStatus(sdk.StatusBuilding.String(), runStatus)

	assert.Equal(t, 1, runStatus.success)
	assert.Equal(t, 1, runStatus.building)
	assert.Equal(t, 0, runStatus.failed)
	assert.Equal(t, 0, runStatus.stoppped)

	computeRunStatus(sdk.StatusWaiting.String(), runStatus)

	assert.Equal(t, 1, runStatus.success)
	assert.Equal(t, 2, runStatus.building)
	assert.Equal(t, 0, runStatus.failed)
	assert.Equal(t, 0, runStatus.stoppped)
}

func TestGetWorkflowRunStatus(t *testing.T) {
	testCases := []struct {
		runStatus statusCounter
		status    string
	}{
		{runStatus: statusCounter{success: 1, building: 0, failed: 0, stoppped: 0}, status: sdk.StatusSuccess.String()},
		{runStatus: statusCounter{success: 1, building: 1, failed: 0, stoppped: 0}, status: sdk.StatusBuilding.String()},
		{runStatus: statusCounter{success: 1, building: 1, failed: 1, stoppped: 0}, status: sdk.StatusBuilding.String()},
		{runStatus: statusCounter{success: 1, building: 0, failed: 1, stoppped: 1}, status: sdk.StatusFail.String()},
		{runStatus: statusCounter{success: 1, building: 0, failed: 0, stoppped: 1}, status: sdk.StatusStopped.String()},
		{runStatus: statusCounter{success: 1, building: 1, failed: 1, stoppped: 1}, status: sdk.StatusBuilding.String()},
		{runStatus: statusCounter{success: 1, building: 1, failed: 1, stoppped: 1, skipped: 1}, status: sdk.StatusBuilding.String()},
		{runStatus: statusCounter{success: 0, building: 0, failed: 1, stoppped: 0, skipped: 1}, status: sdk.StatusFail.String()},
		{runStatus: statusCounter{success: 0, building: 0, failed: 0, stoppped: 0, skipped: 1}, status: sdk.StatusSkipped.String()},
		{runStatus: statusCounter{success: 0, building: 0, failed: 0, stoppped: 0, skipped: 1, disabled: 1}, status: sdk.StatusSkipped.String()},
		{runStatus: statusCounter{success: 0, building: 0, failed: 0, stoppped: 0, skipped: 0, disabled: 1}, status: sdk.StatusDisabled.String()},
		{status: sdk.StatusNeverBuilt.String()},
	}

	for _, tc := range testCases {
		status := getRunStatus(tc.runStatus)
		assert.Equal(t, tc.status, status)
	}
}
