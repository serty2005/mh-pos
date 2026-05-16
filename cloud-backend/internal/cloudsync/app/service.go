package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud-backend/internal/cloudsync/contracts"
	"cloud-backend/internal/platform/clock"
)

type Repository interface {
	ReceiveEdgeEvent(context.Context, EdgeEventReceipt) (contracts.EventAck, error)
	ListEdgeEvents(context.Context, EdgeEventListFilter) ([]contracts.EdgeEventView, error)
	UpsertMasterDataPackage(context.Context, contracts.MasterDataPackage) (contracts.MasterDataPackage, error)
	GetMasterDataPackage(context.Context, string, string) (contracts.MasterDataPackage, error)
	AuthenticateNodeToken(context.Context, string, string, string) error
}

type EdgeEventReceipt struct {
	Envelope         contracts.SyncEnvelope
	IdempotencyKey   string
	RawPayload       []byte
	RawPayloadSHA256 string
	CloudReceivedAt  time.Time
}

// EdgeEventListFilter ограничивает журнал incoming Edge events безопасным page-size и restaurant scope.
type EdgeEventListFilter struct {
	RestaurantID string
	DeviceID     string
	EventType    string
	Limit        int
}

type Service struct {
	repo  Repository
	clock clock.Clock
}

func NewService(repo Repository, clock clock.Clock) *Service {
	return &Service{repo: repo, clock: clock}
}

func (s *Service) Receive(ctx context.Context, raw []byte) (contracts.EventAck, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return contracts.EventAck{}, fmt.Errorf("%w: empty body", contracts.ErrInvalidEnvelope)
	}
	var envelope contracts.SyncEnvelope
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&envelope); err != nil {
		return contracts.EventAck{}, fmt.Errorf("%w: %v", contracts.ErrInvalidEnvelope, err)
	}
	key, err := contracts.IdempotencyKey(envelope)
	if err != nil {
		return contracts.EventAck{}, err
	}
	sum := sha256.Sum256(raw)
	receivedAt := s.clock.Now().UTC()
	return s.repo.ReceiveEdgeEvent(ctx, EdgeEventReceipt{
		Envelope:         envelope,
		IdempotencyKey:   key,
		RawPayload:       append([]byte(nil), raw...),
		RawPayloadSHA256: hex.EncodeToString(sum[:]),
		CloudReceivedAt:  receivedAt,
	})
}

// ListEdgeEvents возвращает последние принятые Edge events без raw payload.
func (s *Service) ListEdgeEvents(ctx context.Context, filter EdgeEventListFilter) ([]contracts.EdgeEventView, error) {
	filter.RestaurantID = strings.TrimSpace(filter.RestaurantID)
	filter.DeviceID = strings.TrimSpace(filter.DeviceID)
	filter.EventType = strings.TrimSpace(filter.EventType)
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 50
	}
	return s.repo.ListEdgeEvents(ctx, filter)
}

// ReceiveBatch принимает batch SyncEnvelope и возвращает item-level ACK decisions.
func (s *Service) ReceiveBatch(ctx context.Context, raws [][]byte) contracts.BatchEventAck {
	items := make([]contracts.BatchEventAckItem, 0, len(raws))
	allAccepted := true
	for i, raw := range raws {
		ack, err := s.Receive(ctx, raw)
		if err == nil {
			items = append(items, contracts.BatchEventAckItem{
				Index:  i,
				Status: contracts.BatchItemAccepted,
				Ack:    &ack,
			})
			continue
		}
		allAccepted = false
		item := contracts.BatchEventAckItem{
			Index: i,
			Error: err.Error(),
		}
		switch {
		case errors.Is(err, contracts.ErrInvalidEnvelope):
			item.Status = contracts.BatchItemRejected
			item.ErrorCode = "INVALID_ENVELOPE"
		case errors.Is(err, contracts.ErrPayloadConflict):
			item.Status = contracts.BatchItemRejected
			item.ErrorCode = "PAYLOAD_CONFLICT"
		default:
			item.Status = contracts.BatchItemRetryable
			item.ErrorCode = "INTERNAL"
		}
		items = append(items, item)
	}
	status := "accepted"
	if !allAccepted {
		status = "partial"
	}
	return contracts.BatchEventAck{
		Status: status,
		Items:  items,
	}
}

// AuthenticateNodeToken проверяет node_token для exchange без раскрытия секрета в логах/ответах.
func (s *Service) AuthenticateNodeToken(ctx context.Context, nodeDeviceID, restaurantID, token string) error {
	nodeDeviceID = strings.TrimSpace(nodeDeviceID)
	restaurantID = strings.TrimSpace(restaurantID)
	token = strings.TrimSpace(token)
	if nodeDeviceID == "" || restaurantID == "" || token == "" {
		return contracts.ErrSyncUnauthorized
	}
	return s.repo.AuthenticateNodeToken(ctx, nodeDeviceID, restaurantID, token)
}

// Exchange выполняет единый Cloud-Edge цикл: preflight stream revisions, прием Edge events, выдача Cloud packages.
func (s *Service) Exchange(ctx context.Context, req contracts.SyncExchangeRequest) (contracts.SyncExchangeResponse, error) {
	if err := contracts.ValidateSyncExchangeRequest(req); err != nil {
		return contracts.SyncExchangeResponse{}, err
	}
	req.NodeDeviceID = strings.TrimSpace(req.NodeDeviceID)
	req.RestaurantID = strings.TrimSpace(req.RestaurantID)
	streamPackages := make(map[string]contracts.MasterDataPackage, len(req.Streams))
	streamResults := make([]contracts.SyncExchangeStreamResult, 0, len(req.Streams))
	for _, stream := range req.Streams {
		stream.StreamName = strings.TrimSpace(stream.StreamName)
		stream.CheckpointToken = strings.TrimSpace(stream.CheckpointToken)
		pkg, err := s.GetMasterDataPackage(ctx, stream.StreamName, req.NodeDeviceID)
		if errors.Is(err, contracts.ErrNotFound) {
			streamResults = append(streamResults, contracts.SyncExchangeStreamResult{
				StreamName: stream.StreamName,
				Status:     contracts.SyncExchangeStreamNotFound,
				ErrorCode:  "SYNC_PACKAGE_NOT_FOUND",
				MessageKey: "errors.sync.packageNotFound",
			})
			continue
		}
		if err != nil {
			return contracts.SyncExchangeResponse{}, err
		}
		if stream.LastCloudVersion > pkg.CloudVersion {
			return contracts.SyncExchangeResponse{}, fmt.Errorf("%w: stream %s edge=%d cloud=%d", contracts.ErrSyncRevisionAhead, stream.StreamName, stream.LastCloudVersion, pkg.CloudVersion)
		}
		if stream.LastCloudVersion == pkg.CloudVersion && stream.CheckpointToken != "" && pkg.CheckpointToken != "" && stream.CheckpointToken != pkg.CheckpointToken {
			return contracts.SyncExchangeResponse{}, fmt.Errorf("%w: stream %s", contracts.ErrSyncCheckpointConflict, stream.StreamName)
		}
		status := contracts.SyncExchangeStreamUpToDate
		if stream.LastCloudVersion < pkg.CloudVersion {
			status = contracts.SyncExchangeStreamChanged
			streamPackages[stream.StreamName] = pkg
		}
		streamResults = append(streamResults, contracts.SyncExchangeStreamResult{
			StreamName:      stream.StreamName,
			Status:          status,
			CloudVersion:    pkg.CloudVersion,
			CheckpointToken: pkg.CheckpointToken,
		})
	}

	edgeAcks := make([]contracts.SyncExchangeEdgeAck, 0, len(req.EdgeEvents))
	allAccepted := true
	for _, item := range req.EdgeEvents {
		ack, err := s.receiveExchangeEdgeEvent(ctx, req, item)
		if err == nil {
			edgeAcks = append(edgeAcks, contracts.SyncExchangeEdgeAck{
				ClientItemID: item.ClientItemID,
				Status:       contracts.BatchItemAccepted,
				Ack:          &ack,
			})
			continue
		}
		allAccepted = false
		edgeAcks = append(edgeAcks, exchangeAckError(item.ClientItemID, err))
	}
	cloudPackages := make([]contracts.SyncExchangeCloudPackage, 0, len(streamPackages))
	for _, stream := range req.Streams {
		pkg, ok := streamPackages[strings.TrimSpace(stream.StreamName)]
		if !ok {
			continue
		}
		cloudPackages = append(cloudPackages, masterPackageToExchange(pkg, req.NodeDeviceID))
	}
	status := contracts.SyncExchangeStatusAccepted
	if !allAccepted {
		status = contracts.SyncExchangeStatusPartial
	}
	return contracts.SyncExchangeResponse{
		ProtocolVersion: contracts.SyncExchangeProtocolVersion,
		Status:          status,
		EdgeAcks:        edgeAcks,
		CloudPackages:   cloudPackages,
		StreamResults:   streamResults,
	}, nil
}

// UpsertMasterDataPackage сохраняет Cloud-authored master/reference/configuration payload для Edge import.
func (s *Service) UpsertMasterDataPackage(ctx context.Context, v contracts.MasterDataPackage) (contracts.MasterDataPackage, error) {
	now := s.clock.Now().UTC()
	v.StreamName = strings.TrimSpace(v.StreamName)
	v.NodeDeviceID = strings.TrimSpace(v.NodeDeviceID)
	v.RestaurantID = strings.TrimSpace(v.RestaurantID)
	v.SyncMode = contracts.NormalizeSyncMode(v.SyncMode)
	v.FullSnapshotReason = strings.TrimSpace(strings.ToLower(v.FullSnapshotReason))
	v.PayloadJSON = bytes.TrimSpace(v.PayloadJSON)
	if err := contracts.ValidateMasterDataPackage(v); err != nil {
		return contracts.MasterDataPackage{}, err
	}
	if v.CloudUpdatedAt != nil {
		updated := v.CloudUpdatedAt.UTC()
		v.CloudUpdatedAt = &updated
	}
	v.CreatedAt = now
	v.UpdatedAt = now
	return s.repo.UpsertMasterDataPackage(ctx, v)
}

// GetMasterDataPackage возвращает Cloud-authored package для запрошенных stream/node.
func (s *Service) GetMasterDataPackage(ctx context.Context, streamName, nodeDeviceID string) (contracts.MasterDataPackage, error) {
	streamName = strings.TrimSpace(streamName)
	nodeDeviceID = strings.TrimSpace(nodeDeviceID)
	if err := contracts.ValidateMasterDataStream(streamName); err != nil {
		return contracts.MasterDataPackage{}, err
	}
	return s.repo.GetMasterDataPackage(ctx, streamName, nodeDeviceID)
}

func exchangeAckError(clientItemID string, err error) contracts.SyncExchangeEdgeAck {
	item := contracts.SyncExchangeEdgeAck{
		ClientItemID: strings.TrimSpace(clientItemID),
		Details:      map[string]string{},
	}
	switch {
	case errors.Is(err, contracts.ErrInvalidEnvelope):
		item.Status = contracts.BatchItemRejected
		item.ErrorCode = "INVALID_ENVELOPE"
		item.MessageKey = "errors.sync.invalidEnvelope"
	case errors.Is(err, contracts.ErrPayloadConflict):
		item.Status = contracts.BatchItemRejected
		item.ErrorCode = "PAYLOAD_CONFLICT"
		item.MessageKey = "errors.sync.payloadConflict"
	default:
		item.Status = contracts.BatchItemRetryable
		item.ErrorCode = "INTERNAL"
		item.MessageKey = "errors.server"
	}
	return item
}

func (s *Service) receiveExchangeEdgeEvent(ctx context.Context, req contracts.SyncExchangeRequest, item contracts.SyncExchangeEdgeEvent) (contracts.EventAck, error) {
	var envelope contracts.SyncEnvelope
	dec := json.NewDecoder(bytes.NewReader(bytes.TrimSpace(item.Payload)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&envelope); err != nil {
		return contracts.EventAck{}, fmt.Errorf("%w: %v", contracts.ErrInvalidEnvelope, err)
	}
	if err := contracts.ValidateEnvelope(envelope); err != nil {
		return contracts.EventAck{}, err
	}
	if envelope.RestaurantID == nil || strings.TrimSpace(*envelope.RestaurantID) != req.RestaurantID {
		return contracts.EventAck{}, fmt.Errorf("%w: envelope restaurant_id does not match exchange restaurant_id", contracts.ErrInvalidEnvelope)
	}
	if strings.TrimSpace(envelope.NodeDeviceID) != "" && strings.TrimSpace(envelope.NodeDeviceID) != req.NodeDeviceID {
		return contracts.EventAck{}, fmt.Errorf("%w: envelope node_device_id does not match exchange node_device_id", contracts.ErrInvalidEnvelope)
	}
	if strings.TrimSpace(envelope.DeviceID) != req.NodeDeviceID {
		return contracts.EventAck{}, fmt.Errorf("%w: envelope device_id does not match exchange node_device_id", contracts.ErrInvalidEnvelope)
	}
	return s.Receive(ctx, item.Payload)
}

func masterPackageToExchange(pkg contracts.MasterDataPackage, nodeDeviceID string) contracts.SyncExchangeCloudPackage {
	if strings.TrimSpace(pkg.NodeDeviceID) == "" {
		pkg.NodeDeviceID = strings.TrimSpace(nodeDeviceID)
	}
	out := contracts.SyncExchangeCloudPackage{
		StreamName:         pkg.StreamName,
		NodeDeviceID:       pkg.NodeDeviceID,
		RestaurantID:       pkg.RestaurantID,
		SyncMode:           pkg.SyncMode,
		FullSnapshotReason: pkg.FullSnapshotReason,
		CloudVersion:       pkg.CloudVersion,
		CheckpointToken:    pkg.CheckpointToken,
		PayloadJSON:        append([]byte(nil), pkg.PayloadJSON...),
	}
	if pkg.CloudUpdatedAt != nil {
		out.CloudUpdatedAt = pkg.CloudUpdatedAt.UTC().Format(time.RFC3339)
	}
	return out
}
