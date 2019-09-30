package main

import (
	"bufio"
	"log"
	"obcyproxy/service"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("./app <token>")
	}

	proxy := service.NewObcyService()
	err := proxy.Start(os.Args[1])
	if err != nil {
		panic(err)
	}

	reader := bufio.NewReader(os.Stdin)
	sender := service.NewConsoleSender()
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			log.Fatalf("Read line failed: %s", err)
			return
		}

		lineStr := string(line)
		proxy.ConsoleCommandService().Handle(sender, lineStr)
	}
}
