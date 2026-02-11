package application

// Domain describes a domain module that exposes its repository.
type Domain interface {
	GetRepository() any
}
