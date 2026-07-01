package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sundowndev/phoneinfoga/v2/lib/number"
	"github.com/sundowndev/phoneinfoga/v2/lib/remote"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api/store"
)

type Scanner struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type GetAllScannersResponse struct {
	Scanners []Scanner `json:"scanners"`
}

// GetAllScanners is an HTTP handler
// @ID GetAllScanners
// @Tags Numbers
// @Summary Get all available scanners.
// @Description This route returns all available scanners.
// @Produce  json
// @Success 200 {object} GetAllScannersResponse
// @Router /v2/scanners [get]
func GetAllScanners(*gin.Context) *api.Response {
	var scanners []Scanner
	for _, s := range RemoteLibrary.GetAllScanners() {
		scanners = append(scanners, Scanner{
			Name:        s.Name(),
			Description: s.Description(),
		})
	}

	return &api.Response{
		Code: http.StatusOK,
		JSON: true,
		Data: GetAllScannersResponse{
			Scanners: scanners,
		},
	}
}

type DryRunScannerInput struct {
	Number  string                `json:"number" binding:"number,required"`
	Options remote.ScannerOptions `json:"options" validate:"dive,required"`
}

type DryRunScannerResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// DryRunScanner is an HTTP handler
// @ID DryRunScanner
// @Tags Numbers
// @Summary Dry run a single scanner
// @Description This route performs a dry run with the given phone number. This doesn't perform an actual scan.
// @Accept  json
// @Produce  json
// @Param request body DryRunScannerInput true "Request body"
// @Success 200 {object} DryRunScannerResponse
// @Success 404 {object} api.ErrorResponse
// @Success 500 {object} api.ErrorResponse
// @Router /v2/scanners/{scanner}/dryrun [post]
// @Param scanner path string true "Scanner name" validate(required)
func DryRunScanner(ctx *gin.Context) *api.Response {
	var input DryRunScannerInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		return &api.Response{
			Code: http.StatusBadRequest,
			JSON: true,
			Data: api.ErrorResponse{Error: "Invalid phone number: please provide an integer without any special chars"},
		}
	}

	if input.Options == nil {
		input.Options = make(remote.ScannerOptions)
	}

	scanner := RemoteLibrary.GetScanner(ctx.Param("scanner"))
	if scanner == nil {
		return &api.Response{
			Code: http.StatusNotFound,
			JSON: true,
			Data: api.ErrorResponse{Error: "Scanner not found"},
		}
	}

	num, err := number.NewNumber(input.Number)
	if err != nil {
		return &api.Response{
			Code: http.StatusBadRequest,
			JSON: true,
			Data: api.ErrorResponse{Error: err.Error()},
		}
	}

	err = scanner.DryRun(*num, input.Options)
	if err != nil {
		return &api.Response{
			Code: http.StatusBadRequest,
			JSON: true,
			Data: DryRunScannerResponse{
				Success: false,
				Error:   err.Error(),
			},
		}
	}

	return &api.Response{
		Code: http.StatusOK,
		JSON: true,
		Data: DryRunScannerResponse{
			Success: true,
		},
	}
}

type RunScannerInput struct {
	Number  string                `json:"number" binding:"number,required"`
	Options remote.ScannerOptions `json:"options" validate:"dive,required"`
	// LookupID, when set, attaches this scan's result to a lookup record (spec §4). Absent
	// (the CLI and callers that don't persist) keeps the endpoint's original behavior.
	LookupID string `json:"lookupId"`
}

// persistScannerResult best-effort persists a scanner result when a lookupId is supplied and
// a store is configured. Persistence must never affect the /run response (spec §7); the store
// error is intentionally swallowed here and surfaced as a warning in task 4.3.
func persistScannerResult(ctx *gin.Context, lookupID, scanner, status, errMessage string, raw json.RawMessage, startedAt, finishedAt time.Time) {
	if Store == nil || lookupID == "" {
		return
	}
	_ = Store.SaveScannerResult(ctx.Request.Context(), store.ScannerResult{
		LookupID:     lookupID,
		Scanner:      scanner,
		Status:       status,
		ErrorMessage: errMessage,
		RawResponse:  raw,
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
		DurationMs:   finishedAt.Sub(startedAt).Milliseconds(),
	})
}

type RunScannerResponse struct {
	Result interface{} `json:"result"`
}

// RunScanner is an HTTP handler
// @ID RunScanner
// @Tags Numbers
// @Summary Run a single scanner
// @Description This route runs a single scanner with the given phone number
// @Accept  json
// @Produce  json
// @Param request body RunScannerInput true "Request body"
// @Success 200 {object} RunScannerResponse
// @Success 404 {object} api.ErrorResponse
// @Success 500 {object} api.ErrorResponse
// @Router /v2/scanners/{scanner}/run [post]
// @Param scanner path string true "Scanner name" validate(required)
func RunScanner(ctx *gin.Context) *api.Response {
	var input RunScannerInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		return &api.Response{
			Code: http.StatusBadRequest,
			JSON: true,
			Data: api.ErrorResponse{Error: "Invalid phone number: please provide an integer without any special chars"},
		}
	}

	if input.Options == nil {
		input.Options = make(remote.ScannerOptions)
	}

	scanner := RemoteLibrary.GetScanner(ctx.Param("scanner"))
	if scanner == nil {
		return &api.Response{
			Code: http.StatusNotFound,
			JSON: true,
			Data: api.ErrorResponse{Error: "Scanner not found"},
		}
	}

	num, err := number.NewNumber(input.Number)
	if err != nil {
		return &api.Response{
			Code: http.StatusBadRequest,
			JSON: true,
			Data: api.ErrorResponse{Error: err.Error()},
		}
	}

	startedAt := time.Now().UTC()
	result, err := scanner.Run(*num, input.Options)
	finishedAt := time.Now().UTC()

	if err != nil {
		persistScannerResult(ctx, input.LookupID, scanner.Name(), store.ResultStatusError,
			err.Error(), nil, startedAt, finishedAt)
		return &api.Response{
			Code: http.StatusInternalServerError,
			JSON: true,
			Data: api.ErrorResponse{Error: err.Error()},
		}
	}

	// Persist the scanner's exact payload verbatim (same marshaling as the live response).
	var raw json.RawMessage
	if b, mErr := json.Marshal(result); mErr == nil {
		raw = b
	}
	persistScannerResult(ctx, input.LookupID, scanner.Name(), store.ResultStatusSuccess,
		"", raw, startedAt, finishedAt)

	return &api.Response{
		Code: http.StatusOK,
		JSON: true,
		Data: RunScannerResponse{
			Result: result,
		},
	}
}
