package buildkite

const (
	EventPing          = "ping"
	EventBuildRunning  = "build.running"
	EventBuildFinished = "build.finished"
)

type Event struct {
	Event        string       `json:"event"`
	Build        Build        `json:"build"`
	Pipeline     Pipeline     `json:"pipeline"`
	Service      Service      `json:"service"`
	Organization Organization `json:"organization"`
	Sender       Person       `json:"sender"`
}
type Service struct {
}
type Organization struct {
}
type Person struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Created string `json:"created_at"`
}
type Build struct {
	ID        string `json:"id"`
	Number    int    `json:"number"`
	Source    string `json:"source"`
	State     string `json:"state"`
	Message   string `json:"message"`
	Creator   Person `json:"creator"`
	Created   string `json:"created_at"`
	Scheduled string `json:"scheduled_at"`
	Started   string `json:"started_at"`
	Finished  string `json:"finished_at"`
}

type Pipeline struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Url  string `json:"url"`
	Slug string `json:"slug"`
}
