package kube

type Repository interface {
	GetContexts() ([]string, error)
}
