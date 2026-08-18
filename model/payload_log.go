package model

import (
	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// PayloadLog stores the full request and response bodies of a relay call. Rows
// are only ever written when the platform-wide switch common.PayloadLogEnabled
// is ON; with the switch OFF (default) no payload is captured or persisted, so
// the platform keeps only billing/ops metadata (the Log table).
type PayloadLog struct {
	Id           int    `json:"id"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint;index"`
	UserId       int    `json:"user_id" gorm:"index"`
	Username     string `json:"username" gorm:"index;default:''"`
	TokenName    string `json:"token_name" gorm:"default:''"`
	ModelName    string `json:"model_name" gorm:"index;default:''"`
	ChannelId    int    `json:"channel_id" gorm:"index;default:0"`
	RequestId    string `json:"request_id" gorm:"type:varchar(64);index;default:''"`
	Ip           string `json:"ip" gorm:"default:''"`
	StatusCode   int    `json:"status_code" gorm:"default:0"`
	DurationMs   int64  `json:"duration_ms" gorm:"default:0"`
	RequestBody  string `json:"request_body,omitempty" gorm:"type:text"`
	ResponseBody string `json:"response_body,omitempty" gorm:"type:text"`
}

func (PayloadLog) TableName() string {
	return "payload_logs"
}

// payloadLogListColumns excludes the two body columns so the list view stays
// light; full bodies are only loaded on demand via GetPayloadLogById.
const payloadLogListColumns = "id, created_at, user_id, username, token_name, model_name, channel_id, request_id, ip, status_code, duration_ms"

// RecordPayloadLog persists a captured payload. Errors are swallowed with a log
// line: payload logging must never affect the live relay request.
func RecordPayloadLog(log *PayloadLog) {
	if log == nil {
		return
	}
	if err := LOG_DB.Create(log).Error; err != nil {
		common.SysLog("failed to record payload log: " + err.Error())
	}
}

// GetPayloadLogs returns a page of payload logs WITHOUT the request/response
// bodies. Use GetPayloadLogById to fetch a single row with the full bodies.
func GetPayloadLogs(username, modelName, requestId string, startTimestamp, endTimestamp int64, startIdx, pageSize int) (logs []*PayloadLog, total int64, err error) {
	tx := LOG_DB.Model(&PayloadLog{})
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	if requestId != "" {
		tx = tx.Where("request_id = ?", requestId)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if err = tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err = tx.Select(payloadLogListColumns).Order("id desc").Limit(pageSize).Offset(startIdx).Find(&logs).Error
	return logs, total, err
}

func GetPayloadLogById(id int) (*PayloadLog, error) {
	if id == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var log PayloadLog
	if err := LOG_DB.Where("id = ?", id).First(&log).Error; err != nil {
		return nil, err
	}
	return &log, nil
}
