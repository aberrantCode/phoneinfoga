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

func TestTwilioScanner_Metadata(t *testing.T) {
	scanner := remote.NewTwilioScanner(&mocks.TwilioSupplier{})
	assert.Equal(t, remote.Twilio, scanner.Name())
	assert.NotEmpty(t, scanner.Description())
}

func TestTwilioScanner(t *testing.T) {
	dummyError := errors.New("dummy")

	testcases := []struct {
		name       string
		number     *number.Number
		opts       remote.ScannerOptions
		mocks      func(*mocks.TwilioSupplier)
		expected   map[string]interface{}
		wantErrors map[string]error
	}{
		{
			name: "successful line type scan",
			number: func() *number.Number {
				n, _ := number.NewNumber("15556661212")
				return n
			}(),
			opts: map[string]interface{}{
				"TWILIO_ACCOUNT_SID": "ACxxxx",
				"TWILIO_AUTH_TOKEN":  "secret",
			},
			mocks: func(s *mocks.TwilioSupplier) {
				s.On("Lookup", "ACxxxx", "secret", "+15556661212", []string{"line_type_intelligence"}).
					Return(&suppliers.TwilioLookupResponse{
						Valid:          true,
						NationalFormat: "(555) 666-1212",
						LineTypeIntelligence: &suppliers.TwilioLineTypeIntelligence{
							Type:              "mobile",
							CarrierName:       "T-Mobile USA, Inc.",
							MobileCountryCode: "310",
							MobileNetworkCode: "800",
						},
					}, nil).Once()
			},
			expected: map[string]interface{}{
				"twilio": remote.TwilioScannerResponse{
					Valid:             true,
					NationalFormat:    "(555) 666-1212",
					LineType:          "mobile",
					CarrierName:       "T-Mobile USA, Inc.",
					MobileCountryCode: "310",
					MobileNetworkCode: "800",
				},
			},
			wantErrors: map[string]error{},
		},
		{
			name: "unavailable package becomes a note",
			number: func() *number.Number {
				n, _ := number.NewNumber("15556661212")
				return n
			}(),
			opts: map[string]interface{}{
				"TWILIO_ACCOUNT_SID":   "ACxxxx",
				"TWILIO_AUTH_TOKEN":    "secret",
				"TWILIO_LOOKUP_FIELDS": "line_type_intelligence,caller_name",
			},
			mocks: func(s *mocks.TwilioSupplier) {
				code := 60601
				s.On("Lookup", "ACxxxx", "secret", "+15556661212", []string{"line_type_intelligence", "caller_name"}).
					Return(&suppliers.TwilioLookupResponse{
						Valid:          true,
						NationalFormat: "(555) 666-1212",
						LineTypeIntelligence: &suppliers.TwilioLineTypeIntelligence{
							Type:              "mobile",
							CarrierName:       "T-Mobile USA, Inc.",
							MobileCountryCode: "310",
							MobileNetworkCode: "800",
						},
						CallerName: &suppliers.TwilioCallerName{
							ErrorCode: &code,
						},
					}, nil).Once()
			},
			expected: map[string]interface{}{
				"twilio": remote.TwilioScannerResponse{
					Valid:             true,
					NationalFormat:    "(555) 666-1212",
					LineType:          "mobile",
					CarrierName:       "T-Mobile USA, Inc.",
					MobileCountryCode: "310",
					MobileNetworkCode: "800",
					Notes:             []string{"caller_name unavailable (error 60601)"},
				},
			},
			wantErrors: map[string]error{},
		},
		{
			name: "failed scan",
			number: func() *number.Number {
				n, _ := number.NewNumber("15556661212")
				return n
			}(),
			opts: map[string]interface{}{
				"TWILIO_ACCOUNT_SID": "ACxxxx",
				"TWILIO_AUTH_TOKEN":  "secret",
			},
			mocks: func(s *mocks.TwilioSupplier) {
				s.On("Lookup", "ACxxxx", "secret", "+15556661212", []string{"line_type_intelligence"}).
					Return(nil, dummyError).Once()
			},
			expected: map[string]interface{}{},
			wantErrors: map[string]error{
				"twilio": dummyError,
			},
		},
		{
			name: "should not run without credentials",
			number: func() *number.Number {
				n, _ := number.NewNumber("15556661212")
				return n
			}(),
			mocks:      func(s *mocks.TwilioSupplier) {},
			expected:   map[string]interface{}{},
			wantErrors: map[string]error{},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			twilioSupplierMock := &mocks.TwilioSupplier{}
			tt.mocks(twilioSupplierMock)

			scanner := remote.NewTwilioScanner(twilioSupplierMock)
			lib := remote.NewLibrary(filter.NewEngine())
			lib.AddScanner(scanner)

			got, errs := lib.Scan(tt.number, tt.opts)
			if len(tt.wantErrors) > 0 {
				assert.Equal(t, tt.wantErrors, errs)
			} else {
				assert.Len(t, errs, 0)
			}
			assert.Equal(t, tt.expected, got)

			twilioSupplierMock.AssertExpectations(t)
		})
	}
}
