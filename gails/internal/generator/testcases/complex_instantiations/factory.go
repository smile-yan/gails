package main

import "github.com/gailsapp/gails/v3/pkg/application"

type Factory[T any, U any] struct {
	simpleFactory[T]
}

func NewFactory[T any, U any]() *Factory[T, U] {
	return &Factory[T, U]{}
}

func (*Factory[T, U]) GetU() application.Service {
	return application.NewService(new(U))
}

type simpleFactory[T any] struct{}

func (simpleFactory[U]) Get() application.Service {
	return application.NewService(new(U))
}
