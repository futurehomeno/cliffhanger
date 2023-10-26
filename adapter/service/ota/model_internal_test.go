package ota

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateReportValidation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		report  UpdateReport
		wantErr bool
	}{
		{
			name: "idle - do nothing",
			report: UpdateReport{
				Status: StatusIdle,
			},
			wantErr: false,
		},
		{
			name: "in progress",
			report: UpdateReport{
				Status: StatusInProgress,
				Progress: ProgressData{
					Progress: 20,
				},
			},
			wantErr: false,
		},
		{
			name: "in progress - with remaining minutes and seconds",
			report: UpdateReport{
				Status: StatusInProgress,
				Progress: ProgressData{
					Progress:         20,
					RemainingMinutes: 1,
					RemainingSeconds: 32,
				},
			},
			wantErr: false,
		},
		{
			name: "done",
			report: UpdateReport{
				Status: StatusDone,
				Result: ResultData{
					Error: ErrInvalidImage,
				},
			},
			wantErr: false,
		},
		{
			name: "done - empty error",
			report: UpdateReport{
				Status: StatusDone,
				Result: ResultData{
					Error: "",
				},
			},
			wantErr: false,
		},
		{
			name: "in progress - progress above 100",
			report: UpdateReport{
				Status: StatusInProgress,
				Progress: ProgressData{
					Progress: 120,
				},
			},
			wantErr: true,
		},
		{
			name: "in progress- progress below 0",
			report: UpdateReport{
				Status: StatusInProgress,
				Progress: ProgressData{
					Progress: -1,
				},
			},
			wantErr: true,
		},
		{
			name: "in progress- remaining minutes below 0",
			report: UpdateReport{
				Status: StatusInProgress,
				Progress: ProgressData{
					Progress:         20,
					RemainingMinutes: -1,
				},
			},
			wantErr: true,
		},
		{
			name: "in progress- remaining seconds below 0",
			report: UpdateReport{
				Status: StatusInProgress,
				Progress: ProgressData{
					Progress:         20,
					RemainingSeconds: -1,
				},
			},
			wantErr: true,
		},
		{
			name: "done - invalid error",
			report: UpdateReport{
				Status: StatusDone,
				Result: ResultData{
					Error: "oops",
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.report.validate()

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
