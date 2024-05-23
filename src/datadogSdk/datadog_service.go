package datadogSdk

import (
	"context"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"os"
	"os/user"
)

var ctx context.Context
var client *datadog.APIClient

func init() {
	ctx = context.WithValue(
		context.Background(),
		datadog.ContextAPIKeys,
		map[string]datadog.APIKey{"apiKeyAuth": {Key: "a576b66cfc29bdf5811cd89746278d5b"}},
	)
	configuration := datadog.NewConfiguration()
	client = datadog.NewAPIClient(configuration)

	hostname, _ := os.Hostname()
	usr, err := user.Current()
	if err != nil {
		usr = &user.User{}
	}
	logger.user = usr
	logger.hostName = &hostname
	logger.logger = datadogV2.NewLogsApi(client)
	logger.baseLog = datadogV2.HTTPLogItem{
		Hostname:             logger.hostName,
		Ddsource:             datadog.PtrString("FYC"),
		AdditionalProperties: map[string]string{"username": logger.user.Username, "name": logger.user.Name},
	}
}
