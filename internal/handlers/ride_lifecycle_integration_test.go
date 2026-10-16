package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"SaarthiRide/internal/models"

	"github.com/gin-gonic/gin"
)

func callRideLifecycleIntegrationHandler(
	handler gin.HandlerFunc,
	rideID uint,
	userID uint,
) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	c.Request = httptest.NewRequest(
		http.MethodPatch,
		fmt.Sprintf("/api/rides/%d", rideID),
		nil,
	)
	c.Params = gin.Params{
		{Key: "id", Value: strconv.FormatUint(uint64(rideID), 10)},
	}
	c.Set("userID", userID)

	handler(c)
	return recorder
}

func TestStartRideIntegration_Success(t *testing.T) {
	s := newAcceptRideTestDB(t)

	driver := s.createUser(t, "driver")
	passenger := s.createUser(t, "passenger")
	vehicle := s.createVehicle(t, driver.ID, false)

	driverID, vehicleID := driver.ID, vehicle.ID
	ride := s.createRide(
		t, passenger.ID, "accepted", &driverID, &vehicleID,
	)

	response := callRideLifecycleIntegrationHandler(
		StartRide, ride.ID, driver.ID,
	)

	if response.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s",
			response.Code, response.Body.String())
	}

	var actual models.Ride
	if err := s.db.First(&actual, ride.ID).Error; err != nil {
		t.Fatal(err)
	}
	if actual.Status != "in_progress" {
		t.Errorf("Expected in_progress, got %q", actual.Status)
	}
}

func TestStartRideIntegration_WrongDriver(t *testing.T) {
	s := newAcceptRideTestDB(t)

	assignedDriver := s.createUser(t, "driver")
	otherDriver := s.createUser(t, "driver")
	passenger := s.createUser(t, "passenger")
	vehicle := s.createVehicle(t, assignedDriver.ID, false)

	driverID, vehicleID := assignedDriver.ID, vehicle.ID
	ride := s.createRide(
		t, passenger.ID, "accepted", &driverID, &vehicleID,
	)

	response := callRideLifecycleIntegrationHandler(
		StartRide, ride.ID, otherDriver.ID,
	)

	if response.Code != http.StatusConflict {
		t.Fatalf("Expected 409, got %d: %s",
			response.Code, response.Body.String())
	}

	var actual models.Ride
	if err := s.db.First(&actual, ride.ID).Error; err != nil {
		t.Fatal(err)
	}
	if actual.Status != "accepted" {
		t.Errorf("Ride should remain accepted, got %q", actual.Status)
	}
}

func TestCompleteRideIntegration_Success(t *testing.T) {
	s := newAcceptRideTestDB(t)

	driver := s.createUser(t, "driver")
	passenger := s.createUser(t, "passenger")
	vehicle := s.createVehicle(t, driver.ID, false)

	driverID, vehicleID := driver.ID, vehicle.ID
	ride := s.createRide(
		t, passenger.ID, "in_progress", &driverID, &vehicleID,
	)

	response := callRideLifecycleIntegrationHandler(
		CompleteRide, ride.ID, driver.ID,
	)

	if response.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s",
			response.Code, response.Body.String())
	}

	var actualRide models.Ride
	if err := s.db.First(&actualRide, ride.ID).Error; err != nil {
		t.Fatal(err)
	}
	if actualRide.Status != "completed" {
		t.Errorf("Expected completed, got %q", actualRide.Status)
	}

	var actualVehicle models.Vehicle
	if err := s.db.First(&actualVehicle, vehicle.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !actualVehicle.IsAvailable {
		t.Error("Expected vehicle to be available after completion")
	}
}

func TestCancelRideIntegration_PassengerCancelsAcceptedRide(t *testing.T) {
	s := newAcceptRideTestDB(t)

	driver := s.createUser(t, "driver")
	passenger := s.createUser(t, "passenger")
	vehicle := s.createVehicle(t, driver.ID, false)

	driverID, vehicleID := driver.ID, vehicle.ID
	ride := s.createRide(
		t, passenger.ID, "accepted", &driverID, &vehicleID,
	)

	response := callRideLifecycleIntegrationHandler(
		CancelRide, ride.ID, passenger.ID,
	)

	if response.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s",
			response.Code, response.Body.String())
	}

	var actualRide models.Ride
	if err := s.db.First(&actualRide, ride.ID).Error; err != nil {
		t.Fatal(err)
	}
	if actualRide.Status != "cancelled" {
		t.Errorf("Expected cancelled, got %q", actualRide.Status)
	}

	var actualVehicle models.Vehicle
	if err := s.db.First(&actualVehicle, vehicle.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !actualVehicle.IsAvailable {
		t.Error("Expected vehicle to be available after cancellation")
	}
}

func TestCancelRideIntegration_OtherPassengerForbidden(t *testing.T) {
	s := newAcceptRideTestDB(t)

	driver := s.createUser(t, "driver")
	owner := s.createUser(t, "passenger")
	otherPassenger := s.createUser(t, "passenger")
	vehicle := s.createVehicle(t, driver.ID, false)

	driverID, vehicleID := driver.ID, vehicle.ID
	ride := s.createRide(
		t, owner.ID, "accepted", &driverID, &vehicleID,
	)

	response := callRideLifecycleIntegrationHandler(
		CancelRide, ride.ID, otherPassenger.ID,
	)

	if response.Code != http.StatusForbidden {
		t.Fatalf("Expected 403, got %d: %s",
			response.Code, response.Body.String())
	}

	var actual models.Ride
	if err := s.db.First(&actual, ride.ID).Error; err != nil {
		t.Fatal(err)
	}
	if actual.Status != "accepted" {
		t.Errorf("Ride should remain accepted, got %q", actual.Status)
	}
}

func TestCancelRideIntegration_CannotCancelInProgressRide(t *testing.T) {
	s := newAcceptRideTestDB(t)

	driver := s.createUser(t, "driver")
	passenger := s.createUser(t, "passenger")
	vehicle := s.createVehicle(t, driver.ID, false)

	driverID, vehicleID := driver.ID, vehicle.ID
	ride := s.createRide(
		t, passenger.ID, "in_progress", &driverID, &vehicleID,
	)

	response := callRideLifecycleIntegrationHandler(
		CancelRide, ride.ID, passenger.ID,
	)

	if response.Code != http.StatusConflict {
		t.Fatalf("Expected 409, got %d: %s",
			response.Code, response.Body.String())
	}

	var actual models.Ride
	if err := s.db.First(&actual, ride.ID).Error; err != nil {
		t.Fatal(err)
	}
	if actual.Status != "in_progress" {
		t.Errorf("Ride should remain in_progress, got %q", actual.Status)
	}
}
