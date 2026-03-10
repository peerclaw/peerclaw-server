package server

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-server/internal/registry"
)

// PublicAgentProfile is the sanitized public view of an agent.
type PublicAgentProfile struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	Description      string              `json:"description,omitempty"`
	Version          string              `json:"version,omitempty"`
	PublicKey        string              `json:"public_key,omitempty"`
	Capabilities     []string            `json:"capabilities,omitempty"`
	Skills           []agentcard.Skill   `json:"skills,omitempty"`
	Protocols        []string            `json:"protocols,omitempty"`
	Status           agentcard.AgentStatus `json:"status"`
	Tags             []string            `json:"tags,omitempty"`
	Verified         bool                `json:"verified"`
	VerifiedAt       *time.Time          `json:"verified_at,omitempty"`
	Trusted          bool                `json:"trusted"`
	ReputationScore  float64             `json:"reputation_score"`
	ReputationEvents int64               `json:"reputation_events"`
	PlaygroundEnabled bool                `json:"playground_enabled"`
	TotalCalls       int64               `json:"total_calls"`
	EndpointURL      string              `json:"endpoint_url,omitempty"` // Only if public_endpoint=true
	RegisteredAt     time.Time           `json:"registered_at"`
	ReviewSummary    *reviewSummaryJSON   `json:"review_summary,omitempty"`
	Categories       []string            `json:"categories,omitempty"`
}

// reviewSummaryJSON is the JSON representation of a review summary.
type reviewSummaryJSON struct {
	AverageRating float64 `json:"average_rating"`
	TotalReviews  int     `json:"total_reviews"`
	Distribution  [5]int  `json:"distribution"`
}

// DirectoryResponse is the response for the public directory endpoint.
type DirectoryResponse struct {
	Agents        []PublicAgentProfile `json:"agents"`
	NextPageToken string               `json:"next_page_token,omitempty"`
	TotalCount    int                  `json:"total_count"`
}

// toPublicProfile converts an internal Card to a sanitized public profile.
func toPublicProfile(card *agentcard.Card) PublicAgentProfile {
	protocols := make([]string, len(card.Protocols))
	for i, p := range card.Protocols {
		protocols[i] = string(p)
	}

	profile := PublicAgentProfile{
		ID:              card.ID,
		Name:            card.Name,
		Description:     card.Description,
		Version:         card.Version,
		PublicKey:       card.PublicKey,
		Capabilities:    card.Capabilities,
		Skills:          card.Skills,
		Protocols:       protocols,
		Status:          card.Status,
		Tags:            card.PeerClaw.Tags,
		ReputationScore: card.PeerClaw.ReputationScore,
		RegisteredAt:    card.RegisteredAt,
	}

	// Only expose endpoint URL if the owner opted in.
	if card.PeerClaw.PublicEndpoint {
		profile.EndpointURL = card.Endpoint.URL
	}

	return profile
}

// handleDirectory handles GET /api/v1/directory — public agent directory.
func (s *HTTPServer) handleDirectory(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := registry.ListFilter{
		Protocol:   q.Get("protocol"),
		Capability: q.Get("capability"),
		Search:     q.Get("search"),
		PageToken:  q.Get("page_token"),
		SortBy:     q.Get("sort"),
	}

	if q.Get("status") != "" {
		filter.Status = agentcard.AgentStatus(q.Get("status"))
	}
	if q.Get("verified") == "true" {
		filter.Verified = true
	}
	if ms := q.Get("min_score"); ms != "" {
		if score, err := strconv.ParseFloat(ms, 64); err == nil {
			filter.MinScore = score
		}
	}
	if ps := q.Get("page_size"); ps != "" {
		if size, err := strconv.Atoi(ps); err == nil {
			filter.PageSize = size
		}
	}

	if q.Get("category") != "" {
		filter.Category = q.Get("category")
	}

	if q.Get("playground_only") == "true" {
		filter.PlaygroundOnly = true
	}
	// Always filter out private agents in public directory.
	filter.PublicOnly = true

	sortByPopular := filter.SortBy == "popular"
	if filter.SortBy == "" {
		filter.SortBy = "reputation"
	}
	// For popular sorting, fetch from DB with default order; we'll re-sort after enrichment.
	if sortByPopular {
		filter.SortBy = "reputation"
	}

	result, err := s.registry.ListAgents(r.Context(), filter)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build a map of agent call counts for popular sort.
	callCounts := make(map[string]int64)
	if sortByPopular && s.invocation != nil {
		since := time.Now().AddDate(0, 0, -7) // last 7 days
		topAgents, err := s.invocation.TopAgents(r.Context(), since, 200)
		if err == nil {
			for _, a := range topAgents {
				callCounts[a.AgentID] = a.TotalCalls
			}
		}
	}

	profiles := make([]PublicAgentProfile, 0, len(result.Agents))
	for _, card := range result.Agents {
		p := toPublicProfile(card)
		if flags, err := s.registry.GetAccessFlags(r.Context(), card.ID); err == nil && flags != nil {
			p.PlaygroundEnabled = flags.PlaygroundEnabled
		}
		// Enrich with reputation data from the engine if available.
		if s.reputation != nil {
			score, _ := s.reputation.GetScore(r.Context(), card.ID)
			p.ReputationScore = score
		}
		// Enrich with call count.
		if count, ok := callCounts[card.ID]; ok {
			p.TotalCalls = count
		}
		profiles = append(profiles, p)
	}

	// Sort by popularity (call count in last 7 days, descending).
	if sortByPopular {
		sort.Slice(profiles, func(i, j int) bool {
			return profiles[i].TotalCalls > profiles[j].TotalCalls
		})
	}

	s.jsonResponse(w, http.StatusOK, DirectoryResponse{
		Agents:        profiles,
		NextPageToken: result.NextPageToken,
		TotalCount:    result.TotalCount,
	})
}

// handlePublicProfile handles GET /api/v1/directory/{id} — single agent public profile.
func (s *HTTPServer) handlePublicProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	card, err := s.registry.GetAgent(r.Context(), id)
	if err != nil {
		s.jsonError(w, "agent not found", http.StatusNotFound)
		return
	}

	// Check visibility.
	flags, _ := s.registry.GetAccessFlags(r.Context(), id)
	if flags != nil && flags.Visibility == "private" {
		s.jsonError(w, "agent not found", http.StatusNotFound)
		return
	}

	profile := toPublicProfile(card)

	if flags != nil {
		profile.PlaygroundEnabled = flags.PlaygroundEnabled
	}

	// Enrich with live reputation data.
	if s.reputation != nil {
		score, _ := s.reputation.GetScore(r.Context(), card.ID)
		profile.ReputationScore = score
		// Trusted badge: verified + high reputation.
		profile.Trusted = profile.Verified && score > 0.8
	}

	// Enrich with review summary.
	if s.reviewService != nil {
		summary, err := s.reviewService.GetSummary(r.Context(), card.ID)
		if err == nil && summary != nil {
			profile.ReviewSummary = &reviewSummaryJSON{
				AverageRating: summary.AverageRating,
				TotalReviews:  summary.TotalReviews,
				Distribution:  summary.Distribution,
			}
		}
	}

	s.jsonResponse(w, http.StatusOK, profile)
}

// handleReputationHistory handles GET /api/v1/directory/{id}/reputation — reputation event history.
func (s *HTTPServer) handleReputationHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if s.reputation == nil {
		s.jsonError(w, "reputation engine not available", http.StatusServiceUnavailable)
		return
	}

	limit := 50
	if ls := r.URL.Query().Get("limit"); ls != "" {
		if l, err := strconv.Atoi(ls); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}

	events, err := s.reputation.GetHistory(r.Context(), id, limit)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if events == nil {
		s.jsonResponse(w, http.StatusOK, map[string]any{"events": []struct{}{}})
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{"events": events})
}

// handleVerifyEndpoint handles POST /api/v1/agents/{id}/verify — initiate endpoint verification.
func (s *HTTPServer) handleVerifyEndpoint(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if s.verificationChallenger == nil {
		s.jsonError(w, "verification not available", http.StatusServiceUnavailable)
		return
	}

	// Get the agent to verify.
	card, err := s.registry.GetAgent(r.Context(), id)
	if err != nil {
		s.jsonError(w, "agent not found", http.StatusNotFound)
		return
	}

	if card.Endpoint.URL == "" {
		s.jsonError(w, "agent has no endpoint URL", http.StatusBadRequest)
		return
	}
	if card.PublicKey == "" {
		s.jsonError(w, "agent has no public key", http.StatusBadRequest)
		return
	}

	result, err := s.verificationChallenger.InitiateChallenge(r.Context(), id, card.Endpoint.URL, card.PublicKey)
	if err != nil {
		// Record verification failure.
		if s.reputation != nil {
			_ = s.reputation.RecordEvent(r.Context(), id, "verification_fail", err.Error())
		}
		s.jsonError(w, "verification failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Mark agent as verified and record success event.
	if s.reputation != nil {
		_ = s.reputation.SetVerified(r.Context(), id)
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"status":    "verified",
		"challenge": result.Challenge,
	})
}
