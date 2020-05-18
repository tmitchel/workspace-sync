package wssync

type Event struct {
	Name string
	Op   string
	File []byte
}
