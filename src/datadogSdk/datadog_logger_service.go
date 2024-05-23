package datadogSdk

import (
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"os/user"
)

type Logger struct {
	logger   *datadogV2.LogsApi
	hostName *string
	user     *user.User
	baseLog  datadogV2.HTTPLogItem
}

var logger = Logger{}

func Info(message string) error {
	logger.baseLog.AdditionalProperties["levelname"] = "INFO"
	logger.baseLog.Message = message
	_, _, err := datadogV2.NewLogsApi(client).SubmitLog(ctx, []datadogV2.HTTPLogItem{logger.baseLog})
	if err != nil {
		return err
	}
	return nil
}

func Error(message string) error {
	logger.baseLog.AdditionalProperties["levelname"] = "ERROR"
	logger.baseLog.Message = message
	_, _, err := datadogV2.NewLogsApi(client).SubmitLog(ctx, []datadogV2.HTTPLogItem{logger.baseLog})
	if err != nil {
		return err
	}
	return nil
}
