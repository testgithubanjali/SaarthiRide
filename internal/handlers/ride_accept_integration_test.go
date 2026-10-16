package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"SaarthiRide/internal/config"
	"SaarthiRide/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

type acceptRideTestDB struct {
	db       *gorm.DB
	users    []uint
	vehicles []uint
	rides    []uint
}

func newAcceptRideTestDB(t *testing.T) *acceptRideTestDB {
	t.Helper()

	if os.Getenv("SAARTHIRIDE_INTEGRATION_TESTS") != "1" {
		t.Skip("Set SAARTHIRIDE_INTEGRATION_TESTS=1 to run database integration tests")
	}

	// Load connection settings from the project's root .env file.
	// Never print or log these settings.
	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatalf("Could not load ../../.env; configure database environment variables: %v", err)
	}

	// Force the test database name, even if .env points to development.
	t.Setenv("DB_NAME", "saarthiride_test")

	for _, key := range []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_SSLMODE"} {
		if os.Getenv(key) == "" {
			t.Fatalf("Required database environment variable %s is empty", key)
		}
	}

	db, err := config.ConnectDatabase()
	if err != nil {
		t.Fatalf("Could not connect to test database: %v", err)
	}

	var databaseName string
	if err := db.Raw("SELECT current_database()").Scan(&databaseName).Error; err != nil {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
		t.Fatalf("Could not verify database name: %v", err)
	}

	if databaseName != "saarthiride_test" {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
		t.Fatalf("Refusing to test against database %q", databaseName)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Could not obtain database connection: %v", err)
	}

	state := &acceptRideTestDB{db: db}

	// Remove only records created by this test run.
	t.Cleanup(func() {
		if len(state.rides) > 0 {
			if err := db.Where("id IN ?", state.rides).
				Delete(&models.Ride{}).Error; err != nil {
				t.Errorf("Could not clean up test rides: %v", err)
			}
		}

		if len(state.vehicles) > 0 {
			if err := db.Where("id IN ?", state.vehicles).
				Delete(&models.Vehicle{}).Error; err != nil {
				t.Errorf("Could not clean up test vehicles: %v", err)
			}
		}

		if len(state.users) > 0 {
			if err := db.Where("id IN ?", state.users).
				Delete(&models.User{}).Error; err != nil {
				t.Errorf("Could not clean up test users: %v", err)
			}
		}

		_ = sqlDB.Close()
	})

	if err := db.AutoMigrate(
		&models.User{},
		&models.Vehicle{},
		&models.Ride{},
	); err != nil {
		t.Fatalf("Could not migrate test database: %v", err)
	}

	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_rides_unique_active_vehicle
		ON public.rides (vehicle_id)
		WHERE vehicle_id IS NOT NULL
		  AND status IN ('accepted', 'in_progress');
	`).Error; err != nil {
		t.Fatalf("Could not create active vehicle index: %v", err)
	}

	return state
}

func (s *acceptRideTestDB) createUser(t *testing.T, role string) models.User {
	t.Helper()

	user := models.User{
		Name:         "Integration " + role,
		Phone:        fmt.Sprintf("it-%s-%d", role, time.Now().UnixNano()),
		PasswordHash: "test-only-hash",
		Role:         role,
	}

	if err := s.db.Create(&user).Error; err != nil {
		t.Fatalf("Could not create test user: %v", err)
	}

	s.users = append(s.users, user.ID)
	return user
}

func (s *acceptRideTestDB) createVehicle(
	t *testing.T,
	driverID uint,
	available bool,
) models.Vehicle {
	t.Helper()

	vehicle := models.Vehicle{
		DriverID:           driverID,
		RegistrationNumber: fmt.Sprintf("IT-%d", time.Now().UnixNano()),
		Model:              "Test Vehicle",
		Capacity:           3,
		IsAvailable:        available,
	}

	if err := s.db.Create(&vehicle).Error; err != nil {
		t.Fatalf("Could not create test vehicle: %v", err)
	}

	s.vehicles = append(s.vehicles, vehicle.ID)
	return vehicle
}

func (s *acceptRideTestDB) createRide(
	t *testing.T,
	passengerID uint,
	status string,
	driverID *uint,
	vehicleID *uint,
) models.Ride {
	t.Helper()

	ride := models.Ride{
		PassengerID:          passengerID,
		DriverID:             driverID,
		VehicleID:            vehicleID,
		PickupLatitude:       32.7266,
		PickupLongitude:      74.857,
		DestinationLatitude:  32.735,
		DestinationLongitude: 74.865,
		Status:               status,
	}

	if err := s.db.Create(&ride).Error; err != nil {
		t.Fatalf("Could not create test ride: %v", err)
	}

	s.rides = append(s.rides, ride.ID)
	return ride
}

func callAcceptRideIntegrationHandler(
	rideID uint,
	userID uint,
) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	c.Request = httptest.NewRequest(
		http.MethodPatch,
		fmt.Sprintf("/api/rides/%d/accept", rideID),
		nil,
	)
	c.Params = gin.Params{
		{Key: "id", Value: strconv.FormatUint(uint64(rideID), 10)},
	}
	c.Set("userID", userID)

	AcceptRide(c)
	return recorder
}

func TestAcceptRideIntegration_PassengerForbidden(t *testing.T) {
	s := newAcceptRideTestDB(t)

	passenger := s.createUser(t, "passenger")
	ride := s.createRide(t, passenger.ID, "pending", nil, nil)

	response := callAcceptRideIntegrationHandler(ride.ID, passenger.ID)

	if response.Code != http.StatusForbidden {
		t.Fatalf("Expected 403, got %d: %s", response.Code, response.Body.String())
	}

	var actual models.Ride
	if err := s.db.First(&actual, ride.ID).Error; err != nil {
		t.Fatalf("Could not reload ride: %v", err)
	}
	if actual.Status != "pending" {
		t.Errorf("Expected ride to remain pending, got %q", actual.Status)
	}
}

func TestAcceptRideIntegration_NoAvailableVehicle(t *testing.T) {
	s := newAcceptRideTestDB(t)

	driver := s.createUser(t, "driver")
	passenger := s.createUser(t, "passenger")
	ride := s.createRide(t, passenger.ID, "pending", nil, nil)

	response := callAcceptRideIntegrationHandler(ride.ID, driver.ID)

	if response.Code != http.StatusConflict {
		t.Fatalf("Expected 409, got %d: %s", response.Code, response.Body.String())
	}

	var actual models.Ride
	if err := s.db.First(&actual, ride.ID).Error; err != nil {
		t.Fatalf("Could not reload ride: %v", err)
	}
	if actual.Status != "pending" || actual.DriverID != nil {
		t.Errorf("Ride should remain pending and unassigned: %+v", actual)
	}
}

func TestAcceptRideIntegration_Success(t *testing.T) {
	s := newAcceptRideTestDB(t)

	driver := s.createUser(t, "driver")
	passenger := s.createUser(t, "passenger")
	vehicle := s.createVehicle(t, driver.ID, true)
	ride := s.createRide(t, passenger.ID, "pending", nil, nil)

	response := callAcceptRideIntegrationHandler(ride.ID, driver.ID)

	if response.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", response.Code, response.Body.String())
	}

	var actualRide models.Ride
	if err := s.db.First(&actualRide, ride.ID).Error; err != nil {
		t.Fatalf("Could not reload ride: %v", err)
	}

	if actualRide.Status != "accepted" {
		t.Errorf("Expected accepted status, got %q", actualRide.Status)
	}
	if actualRide.DriverID == nil || *actualRide.DriverID != driver.ID {
		t.Errorf("Expected driver %d, got %v", driver.ID, actualRide.DriverID)
	}
	if actualRide.VehicleID == nil || *actualRide.VehicleID != vehicle.ID {
		t.Errorf("Expected vehicle %d, got %v", vehicle.ID, actualRide.VehicleID)
	}

	var actualVehicle models.Vehicle
	if err := s.db.First(&actualVehicle, vehicle.ID).Error; err != nil {
		t.Fatalf("Could not reload vehicle: %v", err)
	}
	if actualVehicle.IsAvailable {
		t.Error("Expected assigned vehicle to become unavailable")
	}
}

func TestAcceptRideIntegration_ExcludesVehicleWithActiveRide(t *testing.T) {
	s := newAcceptRideTestDB(t)

	driver := s.createUser(t, "driver")
	passenger1 := s.createUser(t, "passenger")
	passenger2 := s.createUser(t, "passenger")
	vehicle := s.createVehicle(t, driver.ID, true)

	driverID := driver.ID
	vehicleID := vehicle.ID

	// Simulate a stale availability flag: an active ride already uses
	// this vehicle, even though is_available is true.
	s.createRide(t, passenger1.ID, "accepted", &driverID, &vehicleID)
	pendingRide := s.createRide(t, passenger2.ID, "pending", nil, nil)

	response := callAcceptRideIntegrationHandler(pendingRide.ID, driver.ID)

	if response.Code != http.StatusConflict {
		t.Fatalf("Expected 409, got %d: %s", response.Code, response.Body.String())
	}

	var actual models.Ride
	if err := s.db.First(&actual, pendingRide.ID).Error; err != nil {
		t.Fatalf("Could not reload pending ride: %v", err)
	}
	if actual.Status != "pending" || actual.DriverID != nil || actual.VehicleID != nil {
		t.Errorf("Ride should remain unassigned: %+v", actual)
	}
}

func TestAcceptRideIntegration_UniqueIndexRejectsDuplicateActiveVehicle(t *testing.T) {
	s := newAcceptRideTestDB(t)

	driver := s.createUser(t, "driver")
	passenger1 := s.createUser(t, "passenger")
	passenger2 := s.createUser(t, "passenger")
	vehicle := s.createVehicle(t, driver.ID, false)

	driverID := driver.ID
	vehicleID := vehicle.ID

	s.createRide(t, passenger1.ID, "accepted", &driverID, &vehicleID)

	duplicate := models.Ride{
		PassengerID:          passenger2.ID,
		DriverID:             &driverID,
		VehicleID:            &vehicleID,
		PickupLatitude:       32.7266,
		PickupLongitude:      74.857,
		DestinationLatitude:  32.735,
		DestinationLongitude: 74.865,
		Status:               "in_progress",
	}

	err := s.db.Create(&duplicate).Error
	if err == nil {
		s.rides = append(s.rides, duplicate.ID)
		t.Fatal("Expected PostgreSQL to reject duplicate active vehicle assignment")
	}
}
