package middleware

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestSelectChannelByCapacitySkipsSaturatedWeightedCandidate(t *testing.T) {
	primary := &model.Channel{Id: 1}
	fallback := &model.Channel{Id: 2}
	result := selectChannelByCapacity(
		primary,
		true,
		func(channel *model.Channel) ([]string, error) {
			if channel.Id == primary.Id {
				return []string{"rpm"}, nil
			}
			return nil, nil
		},
		func(excluded map[int]struct{}) (*model.Channel, error) {
			require.Contains(t, excluded, primary.Id)
			return fallback, nil
		},
	)
	require.Equal(t, fallback, result.Channel)
	require.NoError(t, result.CapacityErr)
	require.NoError(t, result.SelectionErr)
}

func TestSelectChannelByCapacityFallsBackFromSaturatedHigherPriority(t *testing.T) {
	higherPriority := &model.Channel{Id: 10}
	lowerPriority := &model.Channel{Id: 20}
	result := selectChannelByCapacity(
		higherPriority,
		true,
		func(channel *model.Channel) ([]string, error) {
			if channel.Id == higherPriority.Id {
				return []string{"concurrency"}, nil
			}
			return nil, nil
		},
		func(map[int]struct{}) (*model.Channel, error) {
			return lowerPriority, nil
		},
	)
	require.Equal(t, lowerPriority, result.Channel)
}

func TestSelectChannelByCapacityReturnsExhaustedWhenAllCandidatesSaturated(t *testing.T) {
	first := &model.Channel{Id: 1}
	second := &model.Channel{Id: 2}
	selectionCount := 0
	result := selectChannelByCapacity(
		first,
		true,
		func(*model.Channel) ([]string, error) {
			return []string{"tpm"}, nil
		},
		func(map[int]struct{}) (*model.Channel, error) {
			selectionCount++
			if selectionCount == 1 {
				return second, nil
			}
			return nil, nil
		},
	)
	require.Nil(t, result.Channel)
	require.Equal(t, []string{"tpm"}, result.Saturated)
}

func TestSelectChannelByCapacityAffinitySaturationFallsBack(t *testing.T) {
	preferred := &model.Channel{Id: 7}
	fallback := &model.Channel{Id: 8}
	result := selectChannelByCapacity(
		preferred,
		true,
		func(channel *model.Channel) ([]string, error) {
			if channel.Id == preferred.Id {
				return []string{"concurrency"}, nil
			}
			return nil, nil
		},
		func(map[int]struct{}) (*model.Channel, error) {
			return fallback, nil
		},
	)
	require.Equal(t, fallback, result.Channel)
	require.NotEqual(t, preferred.Id, result.Channel.Id)
}
