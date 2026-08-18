package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// GetPayloadLogs returns a paginated, body-free list of captured payload logs.
// Admin-only (registered behind AdminAuth).
func GetPayloadLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	username := c.Query("username")
	modelName := c.Query("model_name")
	requestId := c.Query("request_id")
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	logs, total, err := model.GetPayloadLogs(username, modelName, requestId, startTimestamp, endTimestamp, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
}

// GetPayloadLogDetail returns a single payload log including the full request
// and response bodies. Admin-only.
func GetPayloadLogDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	log, err := model.GetPayloadLogById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, log)
}
