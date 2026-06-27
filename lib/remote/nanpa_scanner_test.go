package remote_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sundowndev/phoneinfoga/v2/lib/filter"
	"github.com/sundowndev/phoneinfoga/v2/lib/number"
	"github.com/sundowndev/phoneinfoga/v2/lib/remote"
	"github.com/sundowndev/phoneinfoga/v2/lib/remote/suppliers"
	"github.com/sundowndev/phoneinfoga/v2/mocks"
)

func TestNANPAScanner_Metadata(t *testing.T) {
	scanner := remote.NewNANPAScanner(&mocks.NumberingPlanSupplier{})
	assert.Equal(t, remote.NANPA, scanner.Name())
	assert.NotEmpty(t, scanner.Description())
}

func TestNANPAScanner(t *testing.T) {
	dummyError := errors.New("dummy")
	us := func() *number.Number { n, _ := number.NewNumber("12025550123"); return n }()
	nonUS := func() *number.Number { n, _ := number.NewNumber("33612345678"); return n }()

	testcases := []struct {
		name       string
		number     *number.Number
		opts       remote.ScannerOptions
		mocks      func(*mocks.NumberingPlanSupplier)
		expected   map[string]interface{}
		wantErrors map[string]error
	}{
		{
			name:   "successful lookup",
			number: us,
			opts:   map[string]interface{}{"NANPA_API_URL": "https://nanpa.test"},
			mocks: func(s *mocks.NumberingPlanSupplier) {
				s.On("Lookup", "https://nanpa.test", "", "202", "555").
					Return(&suppliers.NumberingPlanResult{
						RateCenter: "WASHINGTON DC",
						State:      "DC",
						OCN:        "0001",
						Carrier:    "Example Telco",
						LATA:       "236",
					}, nil).Once()
			},
			expected: map[string]interface{}{
				"nanpa": remote.NANPAScannerResponse{
					Found:      true,
					NPA:        "202",
					NXX:        "555",
					RateCenter: "WASHINGTON DC",
					State:      "DC",
					OCN:        "0001",
					Carrier:    "Example Telco",
					LATA:       "236",
				},
			},
			wantErrors: map[string]error{},
		},
		{
			name:       "skipped for non +1 number",
			number:     nonUS,
			opts:       map[string]interface{}{"NANPA_API_URL": "https://nanpa.test"},
			mocks:      func(s *mocks.NumberingPlanSupplier) {},
			expected:   map[string]interface{}{},
			wantErrors: map[string]error{},
		},
		{
			name:       "skipped without endpoint",
			number:     us,
			opts:       map[string]interface{}{},
			mocks:      func(s *mocks.NumberingPlanSupplier) {},
			expected:   map[string]interface{}{},
			wantErrors: map[string]error{},
		},
		{
			name:   "supplier error",
			number: us,
			opts:   map[string]interface{}{"NANPA_API_URL": "https://nanpa.test"},
			mocks: func(s *mocks.NumberingPlanSupplier) {
				s.On("Lookup", "https://nanpa.test", "", "202", "555").
					Return(nil, dummyError).Once()
			},
			expected: map[string]interface{}{},
			wantErrors: map[string]error{
				"nanpa": dummyError,
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			m := &mocks.NumberingPlanSupplier{}
			tt.mocks(m)

			scanner := remote.NewNANPAScanner(m)
			lib := remote.NewLibrary(filter.NewEngine())
			lib.AddScanner(scanner)

			got, errs := lib.Scan(tt.number, tt.opts)
			if len(tt.wantErrors) > 0 {
				assert.Equal(t, tt.wantErrors, errs)
			} else {
				assert.Len(t, errs, 0)
			}
			assert.Equal(t, tt.expected, got)

			m.AssertExpectations(t)
		})
	}
}
