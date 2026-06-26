package remote_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sundowndev/phoneinfoga/v2/lib/filter"
	"github.com/sundowndev/phoneinfoga/v2/lib/number"
	"github.com/sundowndev/phoneinfoga/v2/lib/remote"
	"github.com/sundowndev/phoneinfoga/v2/lib/remote/suppliers"
	"github.com/sundowndev/phoneinfoga/v2/mocks"
)

const breachDisclaimer = "Exposure indicators from a third-party breach-aggregation provider. Use only for lawful purposes with a permissible basis."

func boolPtr(b bool) *bool { return &b }

func TestBreachScanner_Metadata(t *testing.T) {
	scanner := remote.NewBreachScanner(&mocks.DehashedSupplier{})
	assert.Equal(t, remote.Breach, scanner.Name())
	assert.NotEmpty(t, scanner.Description())
}

func TestBreachScanner(t *testing.T) {
	dummyError := errors.New("dummy")

	hit := &suppliers.DehashedResponse{
		Success: boolPtr(true),
		Total:   1,
		Entries: []suppliers.DehashedEntry{
			{
				DatabaseName: "ExampleBreach",
				Email:        json.RawMessage(`"bob@example.com"`),
				Username:     json.RawMessage(`"bob"`),
				Phone:        json.RawMessage(`"15556661212"`),
			},
		},
	}

	testcases := []struct {
		name       string
		number     *number.Number
		opts       remote.ScannerOptions
		mocks      func(*mocks.DehashedSupplier)
		expected   map[string]interface{}
		wantErrors map[string]error
	}{
		{
			name: "disabled by default even with credentials",
			number: func() *number.Number {
				n, _ := number.NewNumber("15556661212")
				return n
			}(),
			opts: map[string]interface{}{
				"DEHASHED_EMAIL":   "a@b.com",
				"DEHASHED_API_KEY": "secret",
			},
			mocks:      func(s *mocks.DehashedSupplier) {},
			expected:   map[string]interface{}{},
			wantErrors: map[string]error{},
		},
		{
			name: "enabled but missing credentials",
			number: func() *number.Number {
				n, _ := number.NewNumber("15556661212")
				return n
			}(),
			opts: map[string]interface{}{
				"BREACH_SCANNER_ENABLED": "true",
			},
			mocks:      func(s *mocks.DehashedSupplier) {},
			expected:   map[string]interface{}{},
			wantErrors: map[string]error{},
		},
		{
			name: "minimal mode does not leak values",
			number: func() *number.Number {
				n, _ := number.NewNumber("15556661212")
				return n
			}(),
			opts: map[string]interface{}{
				"BREACH_SCANNER_ENABLED": "true",
				"DEHASHED_EMAIL":         "a@b.com",
				"DEHASHED_API_KEY":       "secret",
			},
			mocks: func(s *mocks.DehashedSupplier) {
				s.On("SearchByPhone", "a@b.com", "secret", `phone:"15556661212"`).
					Return(hit, nil).Once()
			},
			expected: map[string]interface{}{
				"breach": remote.BreachScannerResponse{
					Found:        true,
					TotalRecords: 1,
					Sources:      []string{"ExampleBreach"},
					Entries: []remote.BreachEntry{
						{
							Source:     "ExampleBreach",
							FieldTypes: []string{"email", "username", "phone"},
						},
					},
					Disclaimer: breachDisclaimer,
				},
			},
			wantErrors: map[string]error{},
		},
		{
			name: "extended mode includes values when opted in",
			number: func() *number.Number {
				n, _ := number.NewNumber("15556661212")
				return n
			}(),
			opts: map[string]interface{}{
				"BREACH_SCANNER_ENABLED": "true",
				"BREACH_INCLUDE_FIELDS":  "true",
				"DEHASHED_EMAIL":         "a@b.com",
				"DEHASHED_API_KEY":       "secret",
			},
			mocks: func(s *mocks.DehashedSupplier) {
				s.On("SearchByPhone", "a@b.com", "secret", `phone:"15556661212"`).
					Return(hit, nil).Once()
			},
			expected: map[string]interface{}{
				"breach": remote.BreachScannerResponse{
					Found:        true,
					TotalRecords: 1,
					Sources:      []string{"ExampleBreach"},
					Entries: []remote.BreachEntry{
						{
							Source:     "ExampleBreach",
							FieldTypes: []string{"email", "username", "phone"},
							Fields: map[string]string{
								"email":    "bob@example.com",
								"username": "bob",
								"phone":    "15556661212",
							},
						},
					},
					Disclaimer: breachDisclaimer,
				},
			},
			wantErrors: map[string]error{},
		},
		{
			name: "no hits",
			number: func() *number.Number {
				n, _ := number.NewNumber("15556661212")
				return n
			}(),
			opts: map[string]interface{}{
				"BREACH_SCANNER_ENABLED": "true",
				"DEHASHED_EMAIL":         "a@b.com",
				"DEHASHED_API_KEY":       "secret",
			},
			mocks: func(s *mocks.DehashedSupplier) {
				s.On("SearchByPhone", "a@b.com", "secret", `phone:"15556661212"`).
					Return(&suppliers.DehashedResponse{Success: boolPtr(true), Total: 0}, nil).Once()
			},
			expected: map[string]interface{}{
				"breach": remote.BreachScannerResponse{
					Found:        false,
					TotalRecords: 0,
					Disclaimer:   breachDisclaimer,
				},
			},
			wantErrors: map[string]error{},
		},
		{
			name: "supplier error",
			number: func() *number.Number {
				n, _ := number.NewNumber("15556661212")
				return n
			}(),
			opts: map[string]interface{}{
				"BREACH_SCANNER_ENABLED": "true",
				"DEHASHED_EMAIL":         "a@b.com",
				"DEHASHED_API_KEY":       "secret",
			},
			mocks: func(s *mocks.DehashedSupplier) {
				s.On("SearchByPhone", "a@b.com", "secret", `phone:"15556661212"`).
					Return(nil, dummyError).Once()
			},
			expected: map[string]interface{}{},
			wantErrors: map[string]error{
				"breach": dummyError,
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			dehashedSupplierMock := &mocks.DehashedSupplier{}
			tt.mocks(dehashedSupplierMock)

			scanner := remote.NewBreachScanner(dehashedSupplierMock)
			lib := remote.NewLibrary(filter.NewEngine())
			lib.AddScanner(scanner)

			got, errs := lib.Scan(tt.number, tt.opts)
			if len(tt.wantErrors) > 0 {
				assert.Equal(t, tt.wantErrors, errs)
			} else {
				assert.Len(t, errs, 0)
			}
			assert.Equal(t, tt.expected, got)

			dehashedSupplierMock.AssertExpectations(t)
		})
	}
}
