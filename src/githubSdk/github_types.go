package githubSdk

type FrontendTemplate struct {
	name        string
	description string
	path        string
	removePath  string
	gitIgnore   []string
}

var (
	Vue3 = FrontendTemplate{
		name:        "vue3",
		description: "",
		path:        "embed/vue3-frontend",
		removePath:  "embed/vue3-frontend",
		gitIgnore:   []string{"node_modules"},
	}
)

type BackendTemplate struct {
	name        string
	description string
	path        string
	removePath  string
	gitIgnore   []string
}

var (
	NodeExpressMainApi = BackendTemplate{
		name:        "nodeExpressMainApi",
		description: "",
		path:        "embed/node-express-main-api",
		removePath:  "embed/node-express-main-api",
		gitIgnore:   []string{"node_modules"},
	}
)
