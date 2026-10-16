package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func runAcceptRideHandlerTest(
	rideID string,
	userID any,
	setUserID bool,
) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	context.Request = httptest.NewRequest(
		http.MethodPatch,
		"/api/rides/"+rideID+"/accept",
		nil,
	)

	// Set the route parameter as Gin would during a real request.
	context.Params = gin.Params{
		{Key: "id", Value: rideID},
	}

	if setUserID {
		context.Set("userID", userID)
	}

	AcceptRide(context)
	return recorder
}

func TestAcceptRideRequiresAuthentication(t *testing.T) {
	response := runAcceptRideHandlerTest("3", nil, false)

	if response.Code != http.StatusUnauthorized {
		t.Errorf(
			"expected status %d, got %d; body: %s",
			http.StatusUnauthorized,
			response.Code,
			response.Body.String(),
		)
	}
}

func TestAcceptRideRejectsInvalidUserID(t *testing.T) {
	response := runAcceptRideHandlerTest("3", "invalid-user-id", true)

	if response.Code != http.StatusUnauthorized {
		t.Errorf(
			"expected status %d, got %d; body: %s",
			http.StatusUnauthorized,
			response.Code,
			response.Body.String(),
		)
	}
}

func TestAcceptRideRejectsInvalidRideID(t *testing.T) {
	response := runAcceptRideHandlerTest("invalid", uint(8), true)

	if response.Code != http.StatusBadRequest {
		t.Errorf(
			"expected status %d, got %d; body: %s",
			http.StatusBadRequest,
			response.Code,
			response.Body.String(),
		)
	}
}
