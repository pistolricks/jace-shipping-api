package main

import (
	"net/http"

	"github.com/my-eq/go-usps"
	"github.com/my-eq/go-usps/models"
	uspsApi "github.com/pistolricks/ShippingApi/internal/usps"
)

func (app *application) FormatStandardAddress(w http.ResponseWriter, r *http.Request) {

	var input struct {
		Firm             string `url:"firm,omitempty"`
		StreetAddress    string `url:"streetAddress"`
		SecondaryAddress string `url:"secondaryAddress,omitempty"`
		City             string `url:"city,omitempty"`
		State            string `url:"state"`
		Urbanization     string `url:"urbanization,omitempty"`
		ZIPCode          string `url:"ZIPCode,omitempty"`
		ZIPPlus4         string `url:"ZIPPlus4,omitempty"`
	}

	client := usps.NewClientWithOAuth(app.config.usps.key, app.config.usps.secret)

	// Standardize an address
	req := &models.AddressRequest{
		Firm:             input.Firm,
		StreetAddress:    input.StreetAddress,
		SecondaryAddress: input.SecondaryAddress,
		City:             input.City,
		State:            input.State,
		Urbanization:     input.Urbanization,
		ZIPCode:          input.ZIPCode,
		ZIPPlus4:         input.ZIPPlus4,
	}

	resp, err := uspsApi.StandardizedAddress(client, req)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

	err = app.writeJSON(w, http.StatusOK, envelope{
		"Street":  resp.Address.StreetAddress,
		"City":    resp.Address.City,
		"State":   resp.Address.State,
		"ZIPCode": resp.Address.ZIPCode,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
