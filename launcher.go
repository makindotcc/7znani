package main

import (
	"bufio"
	"log"
	"obcyproxy/discord"
	"obcyproxy/service"
	"os"
)

func main() {
	if len(os.Args) < 6 {
		log.Fatal("./app <token> <webhookid> <webhooktoken> <channelid> <ip>")
	}

	proxy := service.NewObcyService(discord.NewWebhookExecutorConfig(os.Args[2], os.Args[3]), os.Args[4], os.Args[5])
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
