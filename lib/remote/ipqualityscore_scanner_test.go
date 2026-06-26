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

func TestIPQualityScoreScanner_Metadata(t *testing.T) {
	scanner := remote.NewIPQualityScoreScanner(&mocks.IPQSSupplier{})
	assert.Equal(t, remote.IPQualityScore, scanner.Name())
	assert.NotEmpty(t, scanner.Description())
}

func TestIPQualityScoreScanner(t *testing.T) {
	dummyError := errors.New("dummy")
	num := func() *number.Number { n, _ := number.NewNumber("15556661212"); return n }()

	testcases := []struct {
		name       string
		number     *number.Number
		opts       remote.ScannerOptions
		mocks      func(*mocks.IPQSSupplier)
		expected   map[string]interface{}
		wantErrors map[string]error
	}{
		{
			name:   "successful lookup",
			number: num,
			opts:   map[string]interface{}{"IPQS_API_KEY": "key"},
			mocks: func(s *mocks.IPQSSupplier) {
				s.On("Validate", "key", "+15556661212").
					Return(&suppliers.IPQSValidateResponse{
						Success:     true,
						Valid:       true,
						Active:      true,
						FraudScore:  75,
						RecentAbuse: true,
						VOIP:        true,
						LineType:    "Wireless",
						Carrier:     "T-Mobile",
						Country:     "US",
					}, nil).Once()
			},
			expected: map[string]interface{}{
				"ipqualityscore": remote.IPQSScannerResponse{
					Valid:       true,
					Active:      true,
					FraudScore:  75,
					RecentAbuse: true,
					VOIP:        true,
					LineType:    "Wireless",
					Carrier:     "T-Mobile",
					Country:     "US",
				},
			},
			wantErrors: map[string]error{},
		},
		{
			name:       "skipped without api key",
			number:     num,
			opts:       map[string]interface{}{},
			mocks:      func(s *mocks.IPQSSupplier) {},
			expected:   map[string]interface{}{},
			wantErrors: map[string]error{},
		},
		{
			name:   "supplier error",
			number: num,
			opts:   map[string]interface{}{"IPQS_API_KEY": "key"},
			mocks: func(s *mocks.IPQSSupplier) {
				s.On("Validate", "key", "+15556661212").
					Return(nil, dummyError).Once()
			},
			expected: map[string]interface{}{},
			wantErrors: map[string]error{
				"ipqualityscore": dummyError,
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			m := &mocks.IPQSSupplier{}
			tt.mocks(m)

			scanner := remote.NewIPQualityScoreScanner(m)
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
