package main

// An exported but internal model.
//
//gails:internal
type InternalModel struct {
	Field string
}

// An exported but internal service.
//
//gails:internal
type InternalService struct{}

func (InternalService) Method(InternalModel) {}
