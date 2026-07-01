package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sundowndev/phoneinfoga/v2/lib/number"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api/store"
)

// e164FromQuery reads and normalizes the required `number` query parameter to E164. On a
// missing or invalid number it returns an empty string and a 400 response to return as-is.
// History endpoints require a number and only ever return that number's lookups (spec §7).
func e164FromQuery(ctx *gin.Context) (string, *api.Response) {
	raw := ctx.Query("number")
	if strings.TrimSpace(raw) == "" {
		return "", &api.Response{
			Code: http.StatusBadRequest,
			JSON: true,
			Data: api.ErrorResponse{Error: "Missing required query parameter: number"},
		}
	}
	num, err := number.NewNumber(raw)
	if err != nil {
		return "", &api.Response{
			Code: http.StatusBadRequest,
			JSON: true,
			Data: api.ErrorResponse{Error: err.Error()},
		}
	}
	return num.E164, nil
}

// lookupNotFound is the 404 response for an unknown lookup id.
func lookupNotFound() *api.Response {
	return &api.Response{
		Code: http.StatusNotFound,
		JSON: true,
		Data: api.ErrorResponse{Error: "Lookup not found"},
	}
}

// CreateLookupInput is the request body for POST /v2/lookups.
type CreateLookupInput struct {
	Number   string   `json:"number" binding:"number,required"`
	Scanners []string `json:"scanners"`
}

// CreateLookupResponse is returned when a lookup request record is created.
type CreateLookupResponse struct {
	ID                string            `json:"id"`
	Number            AddNumberResponse `json:"number"`
	ScannersRequested []string          `json:"scannersRequested"`
	ClientIP          string            `json:"clientIp"`
	CreatedAt         time.Time         `json:"createdAt"`
	Status            string            `json:"status"`
}

// storeUnavailable is the response used when a persistence handler is reached without a
// configured store (should not happen under `serve`, where persistence is always on).
func storeUnavailable() *api.Response {
	return &api.Response{
		Code: http.StatusInternalServerError,
		JSON: true,
		Data: api.ErrorResponse{Error: "Lookup persistence is not configured"},
	}
}

// numberMetadata projects a parsed number into the response shape shared with AddNumber.
func numberMetadata(n number.Number) AddNumberResponse {
	return AddNumberResponse{
		Valid:         n.Valid,
		RawLocal:      n.RawLocal,
		Local:         n.Local,
		E164:          n.E164,
		International: n.International,
		CountryCode:   n.CountryCode,
		Country:       n.Country,
		Carrier:       n.Carrier,
	}
}

// CreateLookup is an HTTP handler
// @ID CreateLookup
// @Tags Lookups
// @Summary Create a lookup request record
// @Description Creates a pending lookup record (number metadata, requested scanners, client IP, user agent) before any scanner runs.
// @Accept  json
// @Produce  json
// @Param request body CreateLookupInput true "Request body"
// @Success 200 {object} CreateLookupResponse
// @Success 400 {object} api.ErrorResponse
// @Success 500 {object} api.ErrorResponse
// @Router /v2/lookups [post]
func CreateLookup(ctx *gin.Context) *api.Response {
	if Store == nil {
		return storeUnavailable()
	}

	var input CreateLookupInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		return &api.Response{
			Code: http.StatusBadRequest,
			JSON: true,
			Data: api.ErrorResponse{Error: "Invalid phone number: please provide an integer without any special chars"},
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

	scanners := input.Scanners
	if scanners == nil {
		scanners = []string{}
	}

	created, err := Store.CreateLookup(ctx.Request.Context(), store.Lookup{
		E164:              num.E164,
		NumberInput:       input.Number,
		Valid:             num.Valid,
		RawLocal:          num.RawLocal,
		Local:             num.Local,
		International:     num.International,
		CountryCode:       num.CountryCode,
		Country:           num.Country,
		Carrier:           num.Carrier,
		ScannersRequested: scanners,
		ClientIP:          ctx.ClientIP(),
		UserAgent:         ctx.GetHeader("User-Agent"),
	})
	if err != nil {
		return &api.Response{
			Code: http.StatusInternalServerError,
			JSON: true,
			Data: api.ErrorResponse{Error: err.Error()},
		}
	}

	return &api.Response{
		Code: http.StatusOK,
		JSON: true,
		Data: CreateLookupResponse{
			ID:                created.ID,
			Number:            numberMetadata(*num),
			ScannersRequested: created.ScannersRequested,
			ClientIP:          created.ClientIP,
			CreatedAt:         created.CreatedAt,
			Status:            created.Status,
		},
	}
}

// CloseLookupResponse summarizes a finalized lookup.
type CloseLookupResponse struct {
	ID                string     `json:"id"`
	Status            string     `json:"status"`
	ScannersRequested []string   `json:"scannersRequested"`
	CreatedAt         time.Time  `json:"createdAt"`
	CompletedAt       *time.Time `json:"completedAt"`
}

// CloseLookup is an HTTP handler
// @ID CloseLookup
// @Tags Lookups
// @Summary Finalize a lookup
// @Description Sets completed_at and computes the complete/partial status for a lookup.
// @Produce  json
// @Success 200 {object} CloseLookupResponse
// @Success 404 {object} api.ErrorResponse
// @Success 500 {object} api.ErrorResponse
// @Router /v2/lookups/{id}/close [post]
// @Param id path string true "Lookup id" validate(required)
func CloseLookup(ctx *gin.Context) *api.Response {
	if Store == nil {
		return storeUnavailable()
	}

	closed, err := Store.CloseLookup(ctx.Request.Context(), ctx.Param("id"))
	if errors.Is(err, store.ErrLookupNotFound) {
		return lookupNotFound()
	}
	if err != nil {
		return &api.Response{
			Code: http.StatusInternalServerError,
			JSON: true,
			Data: api.ErrorResponse{Error: err.Error()},
		}
	}

	return &api.Response{
		Code: http.StatusOK,
		JSON: true,
		Data: CloseLookupResponse{
			ID:                closed.ID,
			Status:            closed.Status,
			ScannersRequested: closed.ScannersRequested,
			CreatedAt:         closed.CreatedAt,
			CompletedAt:       closed.CompletedAt,
		},
	}
}

// GetLookup is an HTTP handler
// @ID GetLookup
// @Tags Lookups
// @Summary Get a lookup's full detail
// @Description Returns the full lookup detail (metadata + all scanner results).
// @Produce  json
// @Success 200 {object} LookupDetailResponse
// @Success 404 {object} api.ErrorResponse
// @Success 500 {object} api.ErrorResponse
// @Router /v2/lookups/{id} [get]
// @Param id path string true "Lookup id" validate(required)
func GetLookup(ctx *gin.Context) *api.Response {
	if Store == nil {
		return storeUnavailable()
	}

	l, err := Store.GetLookup(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		return &api.Response{
			Code: http.StatusInternalServerError,
			JSON: true,
			Data: api.ErrorResponse{Error: err.Error()},
		}
	}
	if l == nil {
		return lookupNotFound()
	}

	return &api.Response{
		Code: http.StatusOK,
		JSON: true,
		Data: lookupDetail(*l),
	}
}

// GetLatestLookup is an HTTP handler
// @ID GetLatestLookup
// @Tags Lookups
// @Summary Get the most recent lookup for a number
// @Description Returns the newest lookup for a number (full detail) for replay. Requires the number query parameter.
// @Produce  json
// @Success 200 {object} LookupDetailResponse
// @Success 400 {object} api.ErrorResponse
// @Success 404 {object} api.ErrorResponse
// @Success 500 {object} api.ErrorResponse
// @Router /v2/lookups/latest [get]
// @Param number query string true "Phone number"
func GetLatestLookup(ctx *gin.Context) *api.Response {
	if Store == nil {
		return storeUnavailable()
	}

	e164, errResp := e164FromQuery(ctx)
	if errResp != nil {
		return errResp
	}

	l, err := Store.GetLatestLookupByNumber(ctx.Request.Context(), e164)
	if err != nil {
		return &api.Response{
			Code: http.StatusInternalServerError,
			JSON: true,
			Data: api.ErrorResponse{Error: err.Error()},
		}
	}
	if l == nil {
		return lookupNotFound()
	}

	return &api.Response{
		Code: http.StatusOK,
		JSON: true,
		Data: lookupDetail(*l),
	}
}
