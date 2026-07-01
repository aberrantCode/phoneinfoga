package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sundowndev/phoneinfoga/v2/lib/number"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api/store"
)

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
