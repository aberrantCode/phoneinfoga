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

func TestValidationScanner_Metadata(t *testing.T) {
	provider := &mocks.ValidationProvider{}
	provider.On("Name").Return("veriphone")

	scanner := remote.NewValidationScanner(provider)
	assert.Equal(t, "veriphone", scanner.Name())
	assert.NotEmpty(t, scanner.Description())
}

func TestValidationScanner(t *testing.T) {
	dummyError := errors.New("dummy")
	num := func() *number.Number { n, _ := number.NewNumber("33612345678"); return n }()

	testcases := []struct {
		name       string
		number     *number.Number
		opts       remote.ScannerOptions
		mocks      func(*mocks.ValidationProvider)
		expected   map[string]interface{}
		wantErrors map[string]error
	}{
		{
			name:   "successful validation",
			number: num,
			opts:   map[string]interface{}{"VERIPHONE_API_KEY": "key"},
			mocks: func(p *mocks.ValidationProvider) {
				p.On("Name").Return("veriphone")
				p.On("KeyEnvVar").Return("VERIPHONE_API_KEY")
				p.On("Validate", "key", "+33612345678").
					Return(&suppliers.ValidationResult{
						Valid:       true,
						Number:      "+33612345678",
						CountryName: "France",
						Location:    "France",
						Carrier:     "Orange",
						LineType:    "mobile",
					}, nil).Once()
			},
			expected: map[string]interface{}{
				"veriphone": remote.ValidationScannerResponse{
					Provider:    "veriphone",
					Valid:       true,
					Number:      "+33612345678",
					CountryName: "France",
					Location:    "France",
					Carrier:     "Orange",
					LineType:    "mobile",
				},
			},
			wantErrors: map[string]error{},
		},
		{
			name:   "skipped without api key",
			number: num,
			opts:   map[string]interface{}{},
			mocks: func(p *mocks.ValidationProvider) {
				p.On("Name").Return("veriphone")
				p.On("KeyEnvVar").Return("VERIPHONE_API_KEY")
			},
			expected:   map[string]interface{}{},
			wantErrors: map[string]error{},
		},
		{
			name:   "provider error",
			number: num,
			opts:   map[string]interface{}{"VERIPHONE_API_KEY": "key"},
			mocks: func(p *mocks.ValidationProvider) {
				p.On("Name").Return("veriphone")
				p.On("KeyEnvVar").Return("VERIPHONE_API_KEY")
				p.On("Validate", "key", "+33612345678").
					Return(nil, dummyError).Once()
			},
			expected: map[string]interface{}{},
			wantErrors: map[string]error{
				"veriphone": dummyError,
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			p := &mocks.ValidationProvider{}
			tt.mocks(p)

			scanner := remote.NewValidationScanner(p)
			lib := remote.NewLibrary(filter.NewEngine())
			lib.AddScanner(scanner)

			got, errs := lib.Scan(tt.number, tt.opts)
			if len(tt.wantErrors) > 0 {
				assert.Equal(t, tt.wantErrors, errs)
			} else {
				assert.Len(t, errs, 0)
			}
			assert.Equal(t, tt.expected, got)

			p.AssertExpectations(t)
		})
	}
}
