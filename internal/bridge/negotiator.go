package bridge

import (
	"fmt"
	"log/slog"

	"github.com/peerclaw/peerclaw-core/agentcard"
)

// Negotiator selects the optimal protocol path between two agents.
type Negotiator struct {
	manager *Manager
	logger  *slog.Logger
}

// NegotiateResult holds the outcome of protocol negotiation.
type NegotiateResult struct {
	Protocol         string
	SourceAdapter    ProtocolBridge
	TargetAdapter    ProtocolBridge
	NeedsTranslation bool
}

// NewNegotiator creates a new protocol negotiator.
func NewNegotiator(manager *Manager, logger *slog.Logger) *Negotiator {
	if logger == nil {
		logger = slog.Default()
	}
	return &Negotiator{
		manager: manager,
		logger:  logger,
	}
}

// protocolPriority defines the default preference order for bridging.
var protocolPriority = []string{"a2a", "mcp", "acp"}

// Negotiate determines the best protocol path between source and target agents.
// It follows these rules:
//  1. If both agents share a common protocol, use it directly (no translation).
//  2. If no common protocol exists, find the best translation path.
//  3. Priority: same protocol direct > a2a bridge > mcp bridge > acp bridge.
func (n *Negotiator) Negotiate(sourceCard, targetCard *agentcard.Card) (*NegotiateResult, error) {
	if sourceCard == nil || targetCard == nil {
		return nil, fmt.Errorf("both source and target cards are required")
	}

	sourceProtos := protocolSet(sourceCard)
	targetProtos := protocolSet(targetCard)

	// 1. Find common protocols (direct connection, no translation needed).
	for _, p := range protocolPriority {
		if sourceProtos[p] && targetProtos[p] {
			adapter, err := n.manager.GetBridge(p)
			if err != nil {
				continue
			}
			n.logger.Debug("negotiated direct protocol",
				"protocol", p,
				"source", sourceCard.ID,
				"target", targetCard.ID,
			)
			return &NegotiateResult{
				Protocol:         p,
				SourceAdapter:    adapter,
				TargetAdapter:    adapter,
				NeedsTranslation: false,
			}, nil
		}
	}

	// 2. No common protocol — find best bridge path.
	// Source protocol → translate → target protocol.
	for _, srcProto := range protocolPriority {
		if !sourceProtos[srcProto] {
			continue
		}
		srcBridge, err := n.manager.GetBridge(srcProto)
		if err != nil {
			continue
		}

		for _, tgtProto := range protocolPriority {
			if !targetProtos[tgtProto] {
				continue
			}
			tgtBridge, err := n.manager.GetBridge(tgtProto)
			if err != nil {
				continue
			}

			n.logger.Debug("negotiated translated protocol path",
				"source_proto", srcProto,
				"target_proto", tgtProto,
				"source", sourceCard.ID,
				"target", targetCard.ID,
			)
			return &NegotiateResult{
				Protocol:         tgtProto,
				SourceAdapter:    srcBridge,
				TargetAdapter:    tgtBridge,
				NeedsTranslation: true,
			}, nil
		}
	}

	return nil, fmt.Errorf("no compatible protocol path between %s and %s", sourceCard.ID, targetCard.ID)
}

func protocolSet(card *agentcard.Card) map[string]bool {
	set := make(map[string]bool, len(card.Protocols))
	for _, p := range card.Protocols {
		set[string(p)] = true
	}
	return set
}
