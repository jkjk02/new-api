package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

const channelCapacityLeaseContextKey = "channel_capacity_lease"

type ChannelCapacityLease struct {
	ChannelID       int
	RequestID       string
	EstimatedTokens int64
	ConcurrencyHeld bool
	released        bool
}

type channelCapacityReservation struct {
	createdAt time.Time
	tokens    int64
}

type channelCapacityState struct {
	active       int
	reservations map[string]channelCapacityReservation
}

var channelCapacityMemory = struct {
	sync.Mutex
	channels map[int]*channelCapacityState
}{channels: map[int]*channelCapacityState{}}

var channelCapacityAcquireScript = `
local activeKey, rpmKey, tpmWindowKey, tpmValueKey = KEYS[1], KEYS[2], KEYS[3], KEYS[4]
local now, cutoff, requestId = tonumber(ARGV[1]), tonumber(ARGV[2]), ARGV[3]
local concurrencyLimit, rpmLimit, tpmLimit, estimated = tonumber(ARGV[4]), tonumber(ARGV[5]), tonumber(ARGV[6]), tonumber(ARGV[7])
local expired = redis.call('ZRANGEBYSCORE', tpmWindowKey, '-inf', cutoff)
if #expired > 0 then redis.call('HDEL', tpmValueKey, unpack(expired)) end
redis.call('ZREMRANGEBYSCORE', tpmWindowKey, '-inf', cutoff)
redis.call('ZREMRANGEBYSCORE', rpmKey, '-inf', cutoff)
local active = tonumber(redis.call('GET', activeKey) or '0')
local rpm = tonumber(redis.call('ZCARD', rpmKey) or '0')
local tpm = 0
local values = redis.call('HVALS', tpmValueKey)
for _, value in ipairs(values) do tpm = tpm + tonumber(value) end
local concurrencyFull = concurrencyLimit > 0 and active >= concurrencyLimit
local rpmFull = rpmLimit > 0 and rpm >= rpmLimit
local tpmFull = tpmLimit > 0 and (tpm + estimated) > tpmLimit
if concurrencyFull or rpmFull or tpmFull then
  return {0, concurrencyFull and 1 or 0, rpmFull and 1 or 0, tpmFull and 1 or 0}
end
if concurrencyLimit > 0 then redis.call('INCR', activeKey); redis.call('EXPIRE', activeKey, 120) end
if rpmLimit > 0 then redis.call('ZADD', rpmKey, now, requestId); redis.call('EXPIRE', rpmKey, 120) end
if tpmLimit > 0 then redis.call('ZADD', tpmWindowKey, now, requestId); redis.call('HSET', tpmValueKey, requestId, estimated); redis.call('EXPIRE', tpmWindowKey, 120); redis.call('EXPIRE', tpmValueKey, 120) end
return {1, 0, 0, 0}
`

func TryAcquireChannelCapacity(c *gin.Context, channel *model.Channel, estimatedTokens int64) (*ChannelCapacityLease, []string, error) {
	settings := channel.GetOtherSettings()
	if settings.ConcurrencyLimit == 0 && settings.RPMLimit == 0 && settings.TPMLimit == 0 {
		return &ChannelCapacityLease{ChannelID: channel.Id}, nil, nil
	}
	if estimatedTokens < 1 {
		estimatedTokens = 1
	}
	requestID := common.GetContextKeyString(c, common.RequestIdKey)
	if requestID == "" {
		requestID = common.NewRequestId()
	}
	lease := &ChannelCapacityLease{
		ChannelID:       channel.Id,
		RequestID:       requestID,
		EstimatedTokens: estimatedTokens,
		ConcurrencyHeld: settings.ConcurrencyLimit > 0,
	}
	if common.RedisEnabled && common.RDB != nil {
		now := time.Now().UnixMilli()
		prefix := fmt.Sprintf("channel_capacity:%d", channel.Id)
		result, err := common.RDB.Eval(context.Background(), channelCapacityAcquireScript,
			[]string{prefix + ":active", prefix + ":rpm", prefix + ":tpm_window", prefix + ":tpm_values"},
			now, now-time.Minute.Milliseconds(), requestID, settings.ConcurrencyLimit, settings.RPMLimit, settings.TPMLimit, estimatedTokens,
		).Int64Slice()
		if err != nil {
			return nil, nil, err
		}
		if len(result) > 0 && result[0] == 1 {
			c.Set(channelCapacityLeaseContextKey, lease)
			return lease, nil, nil
		}
		return nil, saturatedDimensions(result), nil
	}
	return acquireChannelCapacityMemory(c, channel.Id, settings.ConcurrencyLimit, settings.RPMLimit, settings.TPMLimit, lease)
}

func acquireChannelCapacityMemory(c *gin.Context, channelID, concurrencyLimit, rpmLimit, tpmLimit int, lease *ChannelCapacityLease) (*ChannelCapacityLease, []string, error) {
	channelCapacityMemory.Lock()
	defer channelCapacityMemory.Unlock()
	now := time.Now()
	state := channelCapacityMemory.channels[channelID]
	if state == nil {
		state = &channelCapacityState{reservations: map[string]channelCapacityReservation{}}
		channelCapacityMemory.channels[channelID] = state
	}
	for id, reservation := range state.reservations {
		if now.Sub(reservation.createdAt) >= time.Minute {
			delete(state.reservations, id)
		}
	}
	var tokens int64
	for _, reservation := range state.reservations {
		tokens += reservation.tokens
	}
	dimensions := make([]string, 0, 3)
	if concurrencyLimit > 0 && state.active >= concurrencyLimit {
		dimensions = append(dimensions, "concurrency")
	}
	if rpmLimit > 0 && len(state.reservations) >= rpmLimit {
		dimensions = append(dimensions, "rpm")
	}
	if tpmLimit > 0 && tokens+lease.EstimatedTokens > int64(tpmLimit) {
		dimensions = append(dimensions, "tpm")
	}
	if len(dimensions) > 0 {
		return nil, dimensions, nil
	}
	if concurrencyLimit > 0 {
		state.active++
	}
	if rpmLimit > 0 || tpmLimit > 0 {
		state.reservations[lease.RequestID] = channelCapacityReservation{createdAt: now, tokens: lease.EstimatedTokens}
	}
	c.Set(channelCapacityLeaseContextKey, lease)
	return lease, nil, nil
}

func saturatedDimensions(result []int64) []string {
	dimensions := make([]string, 0, 3)
	if len(result) > 1 && result[1] == 1 {
		dimensions = append(dimensions, "concurrency")
	}
	if len(result) > 2 && result[2] == 1 {
		dimensions = append(dimensions, "rpm")
	}
	if len(result) > 3 && result[3] == 1 {
		dimensions = append(dimensions, "tpm")
	}
	return dimensions
}

func ReconcileChannelCapacity(c *gin.Context, actualTokens int) {
	value, exists := c.Get(channelCapacityLeaseContextKey)
	if !exists {
		return
	}
	lease, ok := value.(*ChannelCapacityLease)
	if !ok || lease.RequestID == "" || actualTokens <= 0 {
		return
	}
	if common.RedisEnabled && common.RDB != nil {
		key := fmt.Sprintf("channel_capacity:%d:tpm_values", lease.ChannelID)
		windowKey := fmt.Sprintf("channel_capacity:%d:tpm_window", lease.ChannelID)
		common.RDB.Eval(context.Background(), `if redis.call('ZSCORE', KEYS[1], ARGV[1]) then return redis.call('HSET', KEYS[2], ARGV[1], ARGV[2]) end return 0`, []string{windowKey, key}, lease.RequestID, actualTokens)
		return
	}
	channelCapacityMemory.Lock()
	defer channelCapacityMemory.Unlock()
	if state := channelCapacityMemory.channels[lease.ChannelID]; state != nil {
		if reservation, found := state.reservations[lease.RequestID]; found {
			reservation.tokens = int64(actualTokens)
			state.reservations[lease.RequestID] = reservation
		}
	}
}

func ReleaseChannelCapacity(c *gin.Context) {
	value, exists := c.Get(channelCapacityLeaseContextKey)
	if !exists {
		return
	}
	lease, ok := value.(*ChannelCapacityLease)
	if !ok || lease.released {
		return
	}
	lease.released = true
	if !lease.ConcurrencyHeld {
		return
	}
	if common.RedisEnabled && common.RDB != nil {
		key := fmt.Sprintf("channel_capacity:%d:active", lease.ChannelID)
		common.RDB.Eval(context.Background(), `local v=tonumber(redis.call('GET',KEYS[1]) or '0'); if v>0 then return redis.call('DECR',KEYS[1]) end; return 0`, []string{key})
		return
	}
	channelCapacityMemory.Lock()
	defer channelCapacityMemory.Unlock()
	if state := channelCapacityMemory.channels[lease.ChannelID]; state != nil && state.active > 0 {
		state.active--
	}
}
