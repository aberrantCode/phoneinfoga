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

func TestHLRScanner_Metadata(t *testing.T) {
	scanner := remote.NewHLRScanner(&mocks.HLRSupplier{})
	assert.Equal(t, remote.HLR, scanner.Name())
	assert.NotEmpty(t, scanner.Description())
}

func TestHLRScanner(t *testing.T) {
	dummyError := errors.New("dummy")
	yes := true

	mobile := func() *number.Number { n, _ := number.NewNumber("33612345678"); return n }()
	landline := func() *number.Number { n, _ := number.NewNumber("33123456789"); return n }()

	testcases := []struct {
		name       string
		number     *number.Number
		opts       remote.ScannerOptions
		mocks      func(*mocks.HLRSupplier)
		expected   map[string]interface{}
		wantErrors map[string]error
	}{
		{
			name:   "successful mobile lookup",
			number: mobile,
			opts:   map[string]interface{}{"HLR_API_KEY": "key", "HLR_API_URL": "https://hlr.test"},
			mocks: func(s *mocks.HLRSupplier) {
				s.On("Lookup", "https://hlr.test", "key", "+33612345678").
					Return(&suppliers.HLRLookupResponse{
						Status:         "active",
						Active:         &yes,
						CurrentCarrier: "Orange",
						LineType:       "mobile",
						Country:        "FR",
					}, nil).Once()
			},
			expected: map[string]interface{}{
				"hlr": remote.HLRScannerResponse{
					Active:         &yes,
					Status:         "active",
					CurrentCarrier: "Orange",
					LineType:       "mobile",
					Country:        "FR",
				},
			},
			wantErrors: map[string]error{},
		},
		{
			name:       "skipped for non-mobile number",
			number:     landline,
			opts:       map[string]interface{}{"HLR_API_KEY": "key", "HLR_API_URL": "https://hlr.test"},
			mocks:      func(s *mocks.HLRSupplier) {},
			expected:   map[string]interface{}{},
			wantErrors: map[string]error{},
		},
		{
			name:       "skipped without api key",
			number:     mobile,
			opts:       map[string]interface{}{},
			mocks:      func(s *mocks.HLRSupplier) {},
			expected:   map[string]interface{}{},
			wantErrors: map[string]error{},
		},
		{
			name:       "skipped without endpoint",
			number:     mobile,
			opts:       map[string]interface{}{"HLR_API_KEY": "key"},
			mocks:      func(s *mocks.HLRSupplier) {},
			expected:   map[string]interface{}{},
			wantErrors: map[string]error{},
		},
		{
			name:   "supplier error",
			number: mobile,
			opts:   map[string]interface{}{"HLR_API_KEY": "key", "HLR_API_URL": "https://hlr.test"},
			mocks: func(s *mocks.HLRSupplier) {
				s.On("Lookup", "https://hlr.test", "key", "+33612345678").
					Return(nil, dummyError).Once()
			},
			expected: map[string]interface{}{},
			wantErrors: map[string]error{
				"hlr": dummyError,
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			m := &mocks.HLRSupplier{}
			tt.mocks(m)

			scanner := remote.NewHLRScanner(m)
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
