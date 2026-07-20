package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func setupChannelCapacitySelectionTest(t *testing.T, channels []*Channel, channelIDs []int) {
	t.Helper()
	oldMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true

	channelSyncLock.Lock()
	oldGroupChannels := group2model2channels
	oldChannels := channelsIDM
	group2model2channels = map[string]map[string][]int{
		"default": {"test-model": channelIDs},
	}
	channelsIDM = make(map[int]*Channel, len(channels))
	for _, channel := range channels {
		channelsIDM[channel.Id] = channel
	}
	channelSyncLock.Unlock()

	t.Cleanup(func() {
		channelSyncLock.Lock()
		group2model2channels = oldGroupChannels
		channelsIDM = oldChannels
		channelSyncLock.Unlock()
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
	})
}

func TestGetRandomSatisfiedChannelExcludingPreservesCachedCandidates(t *testing.T) {
	first := &Channel{Id: 1, Priority: common.GetPointer(int64(10)), Weight: common.GetPointer(uint(100))}
	second := &Channel{Id: 2, Priority: common.GetPointer(int64(10)), Weight: common.GetPointer(uint(1))}
	channelIDs := []int{first.Id, second.Id}
	setupChannelCapacitySelectionTest(t, []*Channel{first, second}, channelIDs)

	selected, err := GetRandomSatisfiedChannelExcluding("default", "test-model", 0, "", map[int]struct{}{first.Id: {}})
	require.NoError(t, err)
	require.Equal(t, second.Id, selected.Id)
	require.Equal(t, []int{first.Id, second.Id}, channelIDs)
}

func TestGetRandomSatisfiedChannelExcludingFallsThroughPriority(t *testing.T) {
	higher := &Channel{Id: 10, Priority: common.GetPointer(int64(10)), Weight: common.GetPointer(uint(100))}
	lower := &Channel{Id: 20, Priority: common.GetPointer(int64(5)), Weight: common.GetPointer(uint(100))}
	setupChannelCapacitySelectionTest(t, []*Channel{higher, lower}, []int{higher.Id, lower.Id})

	selected, err := GetRandomSatisfiedChannelExcluding("default", "test-model", 0, "", map[int]struct{}{higher.Id: {}})
	require.NoError(t, err)
	require.Equal(t, lower.Id, selected.Id)
}

func TestGetRandomSatisfiedChannelExcludingReturnsNilWhenAllExcluded(t *testing.T) {
	channel := &Channel{Id: 1, Priority: common.GetPointer(int64(10)), Weight: common.GetPointer(uint(100))}
	setupChannelCapacitySelectionTest(t, []*Channel{channel}, []int{channel.Id})

	selected, err := GetRandomSatisfiedChannelExcluding("default", "test-model", 0, "", map[int]struct{}{channel.Id: {}})
	require.NoError(t, err)
	require.Nil(t, selected)
}
