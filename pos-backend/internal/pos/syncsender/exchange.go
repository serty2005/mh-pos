package syncsender

import "pos-backend/internal/pos/domain"

const (
	SyncExchangeProtocolVersion = domain.SyncExchangeProtocolVersion
	SyncExchangeStatusAccepted  = domain.SyncExchangeStatusAccepted
	SyncExchangeStatusPartial   = domain.SyncExchangeStatusPartial
)

type SyncExchangeState = domain.SyncExchangeState
type SyncExchangeRequest = domain.SyncExchangeRequest
type SyncExchangeEdgeEvent = domain.SyncExchangeEdgeEvent
type SyncExchangeStreamRequest = domain.SyncExchangeStreamRequest
type CloudPackage = domain.CloudPackage

type SyncExchangeResponse struct {
	Status        string
	EdgeAcks      []BatchSendResult
	CloudPackages []CloudPackage
}
