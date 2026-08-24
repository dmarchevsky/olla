package routing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thushan/olla/internal/config"
	"github.com/thushan/olla/internal/core/constants"
	"github.com/thushan/olla/internal/core/domain"
)

func TestDiscoveryStrategy_NilDiscoveryService(t *testing.T) {
	ctx := context.Background()
	testLogger := createTestLogger()

	healthyEndpoints := []*domain.Endpoint{
		{Name: "ep1", URLString: "http://ep1", Status: domain.StatusHealthy},
	}
	modelEndpoints := []string{"http://ep2"} // model only on an endpoint not currently healthy

	strategy := NewDiscoveryStrategy(nil, config.ModelRoutingStrategyOptions{
		DiscoveryRefreshOnMiss: true,
	}, testLogger)

	result, decision, err := strategy.GetRoutableEndpoints(ctx, "test-model", healthyEndpoints, modelEndpoints)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Equal(t, "rejected", string(decision.Action))
	assert.Equal(t, constants.RoutingReasonDiscoveryServiceUnavailable, decision.Reason)
}
