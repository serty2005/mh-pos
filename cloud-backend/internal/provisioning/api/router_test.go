package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"cloud-backend/internal/platform/clock"
	"cloud-backend/internal/provisioning/api"
	"cloud-backend/internal/provisioning/app"
	"cloud-backend/internal/provisioning/domain"
)

type fixedIDs struct{}

func (fixedIDs) NewID() string { return "id-test" }

type provisioningRepo struct {
	nodes map[string]domain.EdgeNode
}

func (r provisioningRepo) RegisterUnassigned(context.Context, domain.UnassignedEdgeNode) (domain.UnassignedEdgeNode, error) {
	return domain.UnassignedEdgeNode{}, nil
}

func (r provisioningRepo) ListUnassigned(context.Context) ([]domain.UnassignedEdgeNode, error) {
	return nil, nil
}

func (r provisioningRepo) ListEdgeNodesByRestaurant(_ context.Context, restaurantID string) ([]domain.EdgeNode, error) {
	var out []domain.EdgeNode
	for _, node := range r.nodes {
		if node.RestaurantID == restaurantID {
			out = append(out, node)
		}
	}
	return out, nil
}

func (r provisioningRepo) UpsertEdgeNode(context.Context, domain.EdgeNode) (domain.EdgeNode, error) {
	return domain.EdgeNode{}, nil
}

func (r provisioningRepo) GetEdgeNode(context.Context, string) (domain.EdgeNode, error) {
	return domain.EdgeNode{}, domain.ErrNotFound
}

func (r provisioningRepo) MarkUnassignedAssigned(context.Context, string, string, time.Time) error {
	return nil
}

func (r provisioningRepo) CreatePairingCode(context.Context, domain.PairingCode) (domain.PairingCode, error) {
	return domain.PairingCode{}, nil
}

func (r provisioningRepo) RevokeActivePairingCodes(context.Context, string, time.Time) error {
	return nil
}

func (r provisioningRepo) GetPairingCode(context.Context, string) (domain.PairingCode, error) {
	return domain.PairingCode{}, domain.ErrNotFound
}

func (r provisioningRepo) ConsumePairingCode(context.Context, string, string, time.Time) error {
	return nil
}

func TestListRestaurantDevicesRoute(t *testing.T) {
	router := chi.NewRouter()
	service := app.NewService(provisioningRepo{nodes: map[string]domain.EdgeNode{
		"node-1": {ID: "edge-node-1", RestaurantID: "restaurant-1", NodeDeviceID: "node-1", DisplayName: "POS Edge", Status: domain.EdgeNodeAssigned},
	}}, nil, clock.SystemClock{}, fixedIDs{}, "http://cloud.local", nil)
	router.Route("/api/v1", func(r chi.Router) {
		api.RegisterRoutes(r, service)
	})

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/restaurants/restaurant-1/devices", nil))

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	if body := res.Body.String(); body == "null\n" || body == "[]\n" {
		t.Fatalf("expected owned edge node in response, got %s", body)
	}
}
