package app

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"cloud-backend/internal/provisioning/domain"
)

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return time.Date(2026, 5, 4, 20, 0, 0, 0, time.UTC)
}

type fixedIDs struct {
	n int
}

func (g *fixedIDs) NewID() string {
	g.n++
	return "id-test"
}

type provisioningRepo struct {
	edgeNodes map[string]domain.EdgeNode
	pairings  map[string]domain.PairingCode
}

func newProvisioningRepo() *provisioningRepo {
	return &provisioningRepo{edgeNodes: map[string]domain.EdgeNode{}, pairings: map[string]domain.PairingCode{}}
}

func (r *provisioningRepo) RegisterUnassigned(context.Context, domain.UnassignedEdgeNode) (domain.UnassignedEdgeNode, error) {
	return domain.UnassignedEdgeNode{}, nil
}

func (r *provisioningRepo) ListUnassigned(context.Context) ([]domain.UnassignedEdgeNode, error) {
	return nil, nil
}

func (r *provisioningRepo) ListEdgeNodesByRestaurant(_ context.Context, restaurantID string) ([]domain.EdgeNode, error) {
	var out []domain.EdgeNode
	for _, node := range r.edgeNodes {
		if node.RestaurantID == restaurantID {
			out = append(out, node)
		}
	}
	return out, nil
}

func (r *provisioningRepo) UpsertEdgeNode(_ context.Context, v domain.EdgeNode) (domain.EdgeNode, error) {
	r.edgeNodes[v.NodeDeviceID] = v
	return v, nil
}

func (r *provisioningRepo) GetEdgeNode(_ context.Context, nodeDeviceID string) (domain.EdgeNode, error) {
	v, ok := r.edgeNodes[nodeDeviceID]
	if !ok {
		return domain.EdgeNode{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *provisioningRepo) MarkUnassignedAssigned(context.Context, string, string, time.Time) error {
	return nil
}

func (r *provisioningRepo) CreatePairingCode(context.Context, domain.PairingCode) (domain.PairingCode, error) {
	return domain.PairingCode{}, nil
}

func (r *provisioningRepo) RevokeActivePairingCodes(context.Context, string, time.Time) error {
	return nil
}

func (r *provisioningRepo) GetPairingCode(_ context.Context, pairingID string) (domain.PairingCode, error) {
	v, ok := r.pairings[pairingID]
	if !ok {
		return domain.PairingCode{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *provisioningRepo) ConsumePairingCode(context.Context, string, string, time.Time) error {
	return nil
}

func TestAssignmentStatusDoesNotRotateExistingNodeToken(t *testing.T) {
	ctx := context.Background()
	repo := newProvisioningRepo()
	repo.edgeNodes["node-1"] = domain.EdgeNode{
		ID:              "edge-node-1",
		RestaurantID:    "restaurant-1",
		NodeDeviceID:    "node-1",
		DisplayName:     "POS Edge",
		Status:          domain.EdgeNodeAssigned,
		CredentialsHash: secretHash("stable-node-token"),
		CreatedAt:       fixedClock{}.Now(),
		UpdatedAt:       fixedClock{}.Now(),
	}
	service := NewService(repo, nil, fixedClock{}, &fixedIDs{}, "http://cloud.local", nil)

	first, err := service.AssignmentStatus(ctx, "node-1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.AssignmentStatus(ctx, "node-1")
	if err != nil {
		t.Fatal(err)
	}
	node, err := repo.GetEdgeNode(ctx, "node-1")
	if err != nil {
		t.Fatal(err)
	}
	if first.Credentials != nil || second.Credentials != nil {
		t.Fatalf("existing node token must not be re-issued by assignment status, got first=%+v second=%+v", first.Credentials, second.Credentials)
	}
	if node.CredentialsHash != secretHash("stable-node-token") {
		t.Fatalf("assignment status rotated credentials hash")
	}
}

func TestAssignmentStatusIssuesNodeTokenOnceWhenMissing(t *testing.T) {
	ctx := context.Background()
	repo := newProvisioningRepo()
	repo.edgeNodes["node-1"] = domain.EdgeNode{
		ID:           "edge-node-1",
		RestaurantID: "restaurant-1",
		NodeDeviceID: "node-1",
		DisplayName:  "POS Edge",
		Status:       domain.EdgeNodeAssigned,
		CreatedAt:    fixedClock{}.Now(),
		UpdatedAt:    fixedClock{}.Now(),
	}
	service := NewService(repo, nil, fixedClock{}, &fixedIDs{}, "http://cloud.local", nil)

	first, err := service.AssignmentStatus(ctx, "node-1")
	if err != nil {
		t.Fatal(err)
	}
	if first.Credentials == nil || first.Credentials.Token == "" {
		t.Fatalf("expected first assignment status to issue credentials, got %+v", first.Credentials)
	}
	issuedHash := repo.edgeNodes["node-1"].CredentialsHash
	if issuedHash == "" {
		t.Fatal("expected issued token hash to be stored")
	}
	second, err := service.AssignmentStatus(ctx, "node-1")
	if err != nil {
		t.Fatal(err)
	}
	if second.Credentials != nil {
		t.Fatalf("expected already issued token not to be reissued, got %+v", second.Credentials)
	}
	if got := repo.edgeNodes["node-1"].CredentialsHash; got != issuedHash {
		t.Fatalf("expected stored token hash to remain stable, before=%q after=%q", issuedHash, got)
	}
}

func TestAssignmentStatusReturnsPendingForUnknownNode(t *testing.T) {
	service := NewService(newProvisioningRepo(), nil, fixedClock{}, &fixedIDs{}, "http://cloud.local", nil)

	status, err := service.AssignmentStatus(context.Background(), "node-missing")
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		t.Fatal(err)
	}
	if status.Status != "pending" {
		t.Fatalf("expected pending status, got %+v", status)
	}
}

func TestListRestaurantDevicesReturnsOwnedEdgeNodesOnly(t *testing.T) {
	repo := newProvisioningRepo()
	repo.edgeNodes["node-1"] = domain.EdgeNode{NodeDeviceID: "node-1", RestaurantID: "restaurant-1", Status: domain.EdgeNodeAssigned}
	repo.edgeNodes["node-2"] = domain.EdgeNode{NodeDeviceID: "node-2", RestaurantID: "restaurant-2", Status: domain.EdgeNodeAssigned}
	service := NewService(repo, nil, fixedClock{}, &fixedIDs{}, "http://cloud.local", nil)

	devices, err := service.ListRestaurantDevices(context.Background(), "restaurant-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 1 || devices[0].NodeDeviceID != "node-1" {
		t.Fatalf("expected one restaurant-owned node, got %+v", devices)
	}
}

func TestPairingPayloadDecryptsWithDerivedPairingKey(t *testing.T) {
	code := "ABCD2345"
	pairingID := "pairing-1"
	key := pairingKey(code, pairingID)
	keyBytes, err := base64.RawURLEncoding.DecodeString(key)
	if err != nil {
		t.Fatal(err)
	}
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		t.Fatal(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	nonce := []byte("123456789012")
	body, err := json.Marshal(PairingConsumePayload{NodeDeviceID: "edge-node-1", DisplayName: "POS Edge", RequestID: "request-1"})
	if err != nil {
		t.Fatal(err)
	}
	ciphertext := aead.Seal(nil, nonce, body, []byte(pairingID))

	got, err := decryptPairingPayload(key, pairingID, base64.RawURLEncoding.EncodeToString(nonce), base64.RawURLEncoding.EncodeToString(ciphertext))
	if err != nil {
		t.Fatal(err)
	}
	if got.NodeDeviceID != "edge-node-1" || got.RequestID != "request-1" {
		t.Fatalf("unexpected decrypted payload: %+v", got)
	}
}
