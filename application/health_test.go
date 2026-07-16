package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/platforma-dev/platforma/application"
)

func TestHealthSnapshotIsDetached(t *testing.T) {
	t.Parallel()

	health := application.NewHealth()
	health.RegisterService("worker")
	health.StartApplication()
	health.StartService("worker")
	payload := map[string][]string{"queues": {"default"}}
	if err := health.SetServiceData("worker", payload); err != nil {
		t.Fatalf("set service data: %v", err)
	}
	payload["queues"][0] = "mutated after storage"

	snapshot := health.Snapshot()
	serviceSnapshot := snapshot.Services["worker"]
	if serviceSnapshot == nil || serviceSnapshot.StartedAt == nil {
		t.Fatal("expected started service in snapshot")
	}

	snapshot.StartedAt = time.Time{}
	*serviceSnapshot.StartedAt = time.Time{}
	serviceSnapshot.Status = application.ServiceStatusError
	dataSnapshot, ok := serviceSnapshot.Data.(json.RawMessage)
	if !ok {
		t.Fatalf("expected serialized service data, got %T", serviceSnapshot.Data)
	}
	dataSnapshot[0] = 'x'
	delete(snapshot.Services, "worker")

	freshSnapshot := health.Snapshot()
	freshService := freshSnapshot.Services["worker"]
	if freshSnapshot.StartedAt.IsZero() {
		t.Fatal("mutating snapshot changed application start time")
	}
	if freshService == nil {
		t.Fatal("mutating snapshot changed service map")
	}
	if freshService.Status != application.ServiceStatusStarted {
		t.Fatalf("expected service status %q, got %q", application.ServiceStatusStarted, freshService.Status)
	}
	if freshService.StartedAt == nil || freshService.StartedAt.IsZero() {
		t.Fatal("mutating snapshot changed service start time")
	}

	var freshPayload map[string][]string
	freshData, ok := freshService.Data.(json.RawMessage)
	if !ok {
		t.Fatalf("expected serialized fresh service data, got %T", freshService.Data)
	}
	if err := json.Unmarshal(freshData, &freshPayload); err != nil {
		t.Fatalf("decode fresh service data: %v", err)
	}
	if freshPayload["queues"][0] != "default" {
		t.Fatalf("expected detached service data, got %q", freshPayload["queues"][0])
	}
}

func TestHealthRejectsNonJSONServiceData(t *testing.T) {
	t.Parallel()

	health := application.NewHealth()
	health.RegisterService("worker")

	err := health.SetServiceData("worker", make(chan struct{}))
	if err == nil {
		t.Fatal("expected non-JSON service data to be rejected")
	}

	if health.Snapshot().Services["worker"].Data != nil {
		t.Fatal("rejected service data was stored")
	}
}

func TestSetServiceDataUnknownServiceIsNoop(t *testing.T) {
	t.Parallel()

	health := application.NewHealth()

	if err := health.SetServiceData("unknown", make(chan struct{})); err != nil {
		t.Fatalf("expected unknown service to remain a no-op, got: %v", err)
	}
}

func TestSetServiceDataAllowsReentrantJSONMarshaler(t *testing.T) {
	t.Parallel()

	health := application.NewHealth()
	health.RegisterService("worker")

	result := make(chan error, 1)
	go func() {
		result <- health.SetServiceData("worker", reentrantHealthPayload{health: health})
	}()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("set reentrant service data: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("SetServiceData deadlocked while marshaling reentrant payload")
	}
}

func TestHealthConcurrentUpdatesAndSnapshots(t *testing.T) {
	t.Parallel()

	const (
		goroutineCount = 8
		iterationCount = 500
	)

	health := application.NewHealth()
	health.RegisterService("worker")

	serviceErr := errors.New("service failed")
	start := make(chan struct{})
	errs := make(chan error, goroutineCount)

	var wg sync.WaitGroup
	for goroutineIndex := range goroutineCount {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start

			for iterationIndex := range iterationCount {
				if goroutineIndex%2 == 0 {
					health.RegisterService("auxiliary")
					health.StartApplication()
					health.StartService("worker")
					payload := map[string][]int{"iterations": {iterationIndex}}
					if err := health.SetServiceData("worker", payload); err != nil {
						errs <- fmt.Errorf("set service data: %w", err)

						return
					}
					payload["iterations"][0] = -1
					health.FailService("worker", serviceErr)

					continue
				}

				snapshot := health.Snapshot()
				if _, err := json.Marshal(snapshot); err != nil {
					errs <- fmt.Errorf("marshal health snapshot: %w", err)

					return
				}

				_ = health.String()
			}
		}()
	}

	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}

func TestHealthCheckHandlerConcurrentRequests(t *testing.T) {
	t.Parallel()

	const (
		goroutineCount = 8
		requestCount   = 200
	)

	service := &healthcheckService{}
	app := application.New()
	app.RegisterService("worker", service)
	handler := application.NewHealthCheckHandler(app)

	start := make(chan struct{})
	errs := make(chan error, goroutineCount)

	var wg sync.WaitGroup
	for range goroutineCount {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start

			for range requestCount {
				response := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodGet, "/health", nil)
				handler.ServeHTTP(response, request)

				if response.Code != http.StatusOK {
					errs <- fmt.Errorf("expected status %d, got %d", http.StatusOK, response.Code)

					return
				}

				var snapshot application.Health
				if err := json.Unmarshal(response.Body.Bytes(), &snapshot); err != nil {
					errs <- fmt.Errorf("decode health response: %w", err)

					return
				}

				if snapshot.Services["worker"] == nil {
					errs <- errors.New("health response is missing worker service")

					return
				}
			}
		}()
	}

	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}

type healthcheckService struct {
	checkCount atomic.Int64
}

type reentrantHealthPayload struct {
	health *application.Health
}

func (p reentrantHealthPayload) MarshalJSON() ([]byte, error) {
	_ = p.health.Snapshot()

	return []byte(`{"reentrant":true}`), nil
}

func (s *healthcheckService) Run(context.Context) error {
	return nil
}

func (s *healthcheckService) Healthcheck(context.Context) any {
	return struct {
		Check int64 `json:"check"`
	}{Check: s.checkCount.Add(1)}
}
