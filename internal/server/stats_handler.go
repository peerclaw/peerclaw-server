package server

import (
	"net/http"
	"sort"
	"time"

	"github.com/peerclaw/peerclaw-server/internal/registry"
)

// DashboardStats holds aggregated data for the dashboard overview.
type DashboardStats struct {
	RegisteredAgents int            `json:"registered_agents"`
	ConnectedAgents  int            `json:"connected_agents"`
	Bridges          []BridgeStat   `json:"bridges"`
	Health           HealthStatus   `json:"health"`
	RecentAgents     []AgentSummary `json:"recent_agents"`
}

// BridgeStat reports the status of a protocol bridge.
type BridgeStat struct {
	Protocol  string `json:"protocol"`
	Available bool   `json:"available"`
	TaskCount int64  `json:"task_count"`
}

// HealthStatus reports overall system health.
type HealthStatus struct {
	Status     string            `json:"status"`
	Components map[string]string `json:"components"`
}

// AgentSummary is a lightweight view of an agent for the dashboard.
type AgentSummary struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Status        string   `json:"status"`
	Protocols     []string `json:"protocols"`
	LastHeartbeat string   `json:"last_heartbeat"`
}

// Bridge adapters expose count methods via these interfaces.
type taskCounter interface {
	TaskCount() int64
}

type sessionCounter interface {
	SessionCount() int64
}

type runCounter interface {
	RunCount() int64
}

func (s *HTTPServer) handleDashboardStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	stats := DashboardStats{
		Health: HealthStatus{
			Status:     "ok",
			Components: make(map[string]string),
		},
	}

	// Registered agents count + recent agents.
	if s.store != nil {
		result, err := s.store.List(ctx, registry.ListFilter{PageSize: 100})
		if err != nil {
			stats.Health.Components["database"] = "error"
			stats.Health.Status = "degraded"
		} else {
			stats.Health.Components["database"] = "ok"
			stats.RegisteredAgents = result.TotalCount

			// Sort by RegisteredAt descending, take top 5.
			agents := result.Agents
			sort.Slice(agents, func(i, j int) bool {
				return agents[i].RegisteredAt.After(agents[j].RegisteredAt)
			})
			limit := 5
			if len(agents) < limit {
				limit = len(agents)
			}
			for _, a := range agents[:limit] {
				protocols := make([]string, len(a.Protocols))
				for i, p := range a.Protocols {
					protocols[i] = string(p)
				}
				stats.RecentAgents = append(stats.RecentAgents, AgentSummary{
					ID:            a.ID,
					Name:          a.Name,
					Status:        string(a.Status),
					Protocols:     protocols,
					LastHeartbeat: a.LastHeartbeat.Format(time.RFC3339),
				})
			}
		}
	}

	// Connected agents (signaling hub).
	if s.sigHub != nil {
		stats.ConnectedAgents = s.sigHub.ConnectedAgents()
		stats.Health.Components["signaling"] = "ok"
	}

	// Bridge stats with task counts via type assertion.
	if s.bridges != nil {
		for _, info := range s.bridges.ListBridges() {
			bs := BridgeStat{
				Protocol:  info.Protocol,
				Available: info.Available,
			}
			if b, err := s.bridges.GetBridge(info.Protocol); err == nil {
				switch v := b.(type) {
				case taskCounter:
					bs.TaskCount = v.TaskCount()
				case sessionCounter:
					bs.TaskCount = v.SessionCount()
				case runCounter:
					bs.TaskCount = v.RunCount()
				}
			}
			stats.Bridges = append(stats.Bridges, bs)
		}
	}

	if stats.RecentAgents == nil {
		stats.RecentAgents = []AgentSummary{}
	}
	if stats.Bridges == nil {
		stats.Bridges = []BridgeStat{}
	}

	s.jsonResponse(w, http.StatusOK, stats)
}
