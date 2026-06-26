package suppliers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

// NumberingPlanSupplierInterface resolves an NPA-NXX to allocation data through
// a configurable endpoint. Generalizes the OVH range-lookup concept to the NANP.
type NumberingPlanSupplierInterface interface {
	Lookup(baseURL, apiKey, npa, nxx string) (*NumberingPlanResult, error)
}

// NumberingPlanResult models the common fields returned by NPA-NXX data
// sources. Adjust the json tags to match the provider configured via
// NANPA_API_URL.
type NumberingPlanResult struct {
	RateCenter  string `json:"rate_center"`
	State       string `json:"state"`
	OCN         string `json:"ocn"`
	Carrier     string `json:"carrier"`
	LATA        string `json:"lata"`
	BlockHolder string `json:"block_holder"`
}

type NANPASupplier struct{}

func NewNANPASupplier() *NANPASupplier {
	return &NANPASupplier{}
}

func (s *NANPASupplier) Lookup(baseURL, apiKey, npa, nxx string) (*NumberingPlanResult, error) {
	logrus.WithField("npa", npa).WithField("nxx", nxx).Debug("Running NANPA NPA-NXX lookup")

	values := url.Values{}
	values.Set("npa", npa)
	values.Set("nxx", nxx)
	if apiKey != "" {
		values.Set("api_key", apiKey)
	}
	endpoint := fmt.Sprintf("%s?%s", baseURL, values.Encode())

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return &NumberingPlanResult{}, nil
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("NANPA provider returned HTTP %d", resp.StatusCode)
	}

	var result NumberingPlanResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
