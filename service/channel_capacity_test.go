package service

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newCapacityTestContext(requestID string) *gin.Context {
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(common.RequestIdKey, requestID)
	return ctx
}

func newCapacityTestChannel(id, concurrency, rpm, tpm int) *model.Channel {
	channel := &model.Channel{Id: id}
	channel.SetOtherSettings(dto.ChannelOtherSettings{
		ConcurrencyLimit: concurrency,
		RPMLimit:         rpm,
		TPMLimit:         tpm,
	})
	return channel
}

func resetCapacityTestState(t *testing.T) {
	t.Helper()
	oldRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	channelCapacityMemory.Lock()
	channelCapacityMemory.channels = map[int]*channelCapacityState{}
	channelCapacityMemory.Unlock()
	t.Cleanup(func() {
		common.RedisEnabled = oldRedisEnabled
	})
}

func TestChannelCapacityConcurrencyReleases(t *testing.T) {
	resetCapacityTestState(t)
	channel := newCapacityTestChannel(1, 1, 0, 0)
	first := newCapacityTestContext("concurrency-1")
	_, saturated, err := TryAcquireChannelCapacity(first, channel, 1)
	require.NoError(t, err)
	require.Empty(t, saturated)

	second := newCapacityTestContext("concurrency-2")
	_, saturated, err = TryAcquireChannelCapacity(second, channel, 1)
	require.NoError(t, err)
	require.Equal(t, []string{"concurrency"}, saturated)

	ReleaseChannelCapacity(first)
	_, saturated, err = TryAcquireChannelCapacity(second, channel, 1)
	require.NoError(t, err)
	require.Empty(t, saturated)
}

func TestChannelCapacityRPMCountsReleasedRequests(t *testing.T) {
	resetCapacityTestState(t)
	channel := newCapacityTestChannel(2, 1, 1, 0)
	first := newCapacityTestContext("rpm-1")
	_, saturated, err := TryAcquireChannelCapacity(first, channel, 1)
	require.NoError(t, err)
	require.Empty(t, saturated)
	ReleaseChannelCapacity(first)

	second := newCapacityTestContext("rpm-2")
	_, saturated, err = TryAcquireChannelCapacity(second, channel, 1)
	require.NoError(t, err)
	require.Equal(t, []string{"rpm"}, saturated)
}

func TestChannelCapacityTPMReconcilesActualUsage(t *testing.T) {
	resetCapacityTestState(t)
	channel := newCapacityTestChannel(3, 1, 0, 100)
	first := newCapacityTestContext("tpm-1")
	_, saturated, err := TryAcquireChannelCapacity(first, channel, 80)
	require.NoError(t, err)
	require.Empty(t, saturated)
	ReconcileChannelCapacity(first, 30)
	ReleaseChannelCapacity(first)

	second := newCapacityTestContext("tpm-2")
	_, saturated, err = TryAcquireChannelCapacity(second, channel, 70)
	require.NoError(t, err)
	require.Empty(t, saturated, fmt.Sprintf("actual usage reconciliation should leave 70 tokens available: %v", saturated))
	ReleaseChannelCapacity(second)

	third := newCapacityTestContext("tpm-3")
	_, saturated, err = TryAcquireChannelCapacity(third, channel, 1)
	require.NoError(t, err)
	require.Equal(t, []string{"tpm"}, saturated)
}
