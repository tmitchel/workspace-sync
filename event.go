package wssync

// Event is the information sent from Local to Remote. It
// includes the type of event, the file affected, and the
// new copy of the file.
type Event struct {
	Name string
	Op   string
	File []byte
}
