package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func runCreateRideHandlerTest(
	body string,
	userID any,
	setUserID bool,
) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/rides",
		strings.NewReader(body),
	)
	context.Request.Header.Set("Content-Type", "application/json")

	if setUserID {
		context.Set("userID", userID)
	}

	CreateRide(context)
	return recorder
}

func TestCreateRideRequiresAuthentication(t *testing.T) {
	response := runCreateRideHandlerTest(`{}`, nil, false)

	if response.Code != http.StatusUnauthorized {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			response.Code,
		)
	}
}

func TestCreateRideRejectsInvalidUserID(t *testing.T) {
	response := runCreateRideHandlerTest(`{}`, "invalid-user-id", true)

	if response.Code != http.StatusUnauthorized {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			response.Code,
		)
	}
}

func TestCreateRideRejectsInvalidJSON(t *testing.T) {
	response := runCreateRideHandlerTest(
		`{"pickup_latitude":`,
		uint(9),
		true,
	)

	if response.Code != http.StatusBadRequest {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			response.Code,
		)
	}
}

func TestCreateRideRejectsInvalidLatitude(t *testing.T) {
	body := `{
		"pickup_latitude": 91,
		"pickup_longitude": 74.857,
		"destination_latitude": 32.735,
		"destination_longitude": 74.865
	}`

	response := runCreateRideHandlerTest(body, uint(9), true)

	if response.Code != http.StatusBadRequest {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			response.Code,
		)
	}
}

func TestCreateRideRejectsInvalidLongitude(t *testing.T) {
	body := `{
		"pickup_latitude": 32.7266,
		"pickup_longitude": 181,
		"destination_latitude": 32.735,
		"destination_longitude": 74.865
	}`

	response := runCreateRideHandlerTest(body, uint(9), true)

	if response.Code != http.StatusBadRequest {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			response.Code,
		)
	}
}
