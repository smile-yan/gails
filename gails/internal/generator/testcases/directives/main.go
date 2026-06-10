//gails:inject console.log("Hello everywhere!");
//gails:inject **:console.log("Hello everywhere again!");
//gails:inject *c:console.log("Hello Classes!");
//gails:inject *i:console.log("Hello Interfaces!");
//gails:inject j*:console.log("Hello JS!");
//gails:inject jc:console.log("Hello JS Classes!");
//gails:inject ji:console.log("Hello JS Interfaces!");
//gails:inject t*:console.log("Hello TS!");
//gails:inject tc:console.log("Hello TS Classes!");
//gails:inject ti:console.log("Hello TS Interfaces!");
package main

import (
	"log"

	"github.com/gailsapp/gails/v3/internal/generator/testcases/directives/otherpackage"
	"github.com/gailsapp/gails/v3/pkg/application"
)

type IgnoredType struct {
	Field int
}

//gails:inject j*:/**
//gails:inject j*: * @param {string} arg
//gails:inject j*: * @returns {Promise<void>}
//gails:inject j*: */
//gails:inject j*:export async function CustomMethod(arg) {
//gails:inject t*:export async function CustomMethod(arg: string): Promise<void> {
//gails:inject     await InternalMethod("Hello " + arg + "!");
//gails:inject }
type Service struct{}

func (*Service) VisibleMethod(otherpackage.Dummy) {}

//gails:ignore
func (*Service) IgnoredMethod(IgnoredType) {}

//gails:internal
func (*Service) InternalMethod(string) {}

func main() {
	app := application.New(application.Options{
		Services: []application.Service{
			application.NewService(&Service{}),
			application.NewService(&unexportedService{}),
			application.NewService(&InternalService{}),
		},
	})

	app.Window.New()

	err := app.Run()

	if err != nil {
		log.Fatal(err)
	}

}
