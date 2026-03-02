package application

type IDGenerator interface {
	NewID() string
}
