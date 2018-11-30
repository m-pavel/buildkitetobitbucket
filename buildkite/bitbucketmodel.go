package buildkite

type Repository struct {
	Name string `json:"name"`
	Type string `json:"type"`
}
type Actor struct {
	Username    string `json:"username"`
	Displayname string `json:"display_name"`
}
type Body struct {
	Repo  Repository `json:"repository"`
	Actor Actor      `json:"actor"`
}
