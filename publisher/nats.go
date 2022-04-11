package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"

	"github.com/nats-io/stan.go"
)

func main() {
	sc, err := stan.Connect("test-cluster", "publisher", stan.NatsURL("nats://localhost:4222"))
	if err != nil && err != io.EOF {
		log.Fatalln(err)
	} else {
		log.Println("Connected to Nats-streaming")
	}

	var filename string
	for {
		fmt.Println("Please insert a file name:")
		fmt.Scanf("%s\n", &filename)
		dataFromFile, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Fatalln(err)
		}
		sc.Publish("foo1", dataFromFile)
		println("Data send!")
	}
}
