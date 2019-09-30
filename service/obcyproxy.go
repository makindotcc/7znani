package service

import (
	"github.com/bwmarrin/discordgo"
	"log"
	"time"
)

type ObcyService struct {
	obcies                *Obcies
	consoleCommandService *ConsoleCommandService
	discordCommandService *DiscordCommandService
	discordSession        *discordgo.Session
}

func (service *ObcyService) ConsoleCommandService() *ConsoleCommandService {
	return service.consoleCommandService
}

func NewObcyService() *ObcyService {
	service := &ObcyService{
		consoleCommandService: NewConsoleCommandService(),
		discordCommandService: NewDiscordCommandService(),
	}
	service.registerCommand(NewSendCommand(service))
	return service
}

func (service *ObcyService) registerCommand(command Command) {
	service.consoleCommandService.Register(command)
	service.discordCommandService.Register(command)
}

func (service *ObcyService) Start(discordToken string) (err error) {
	service.discordSession, err = discordgo.New("Bot " + discordToken)
	if err != nil {
		return
	}

	err = service.discordSession.Open()
	if err != nil {
		return
	}
	service.discordCommandService.Attach(service.discordSession)

	for i := 0; i < 1; i++ {
		go func() {
			for {
				time.Sleep(3 * time.Second)
				service.LogMessage("\n``--- nowa rozmowa ---``\n")

				service.obcies = NewObcies(service)
				err = service.obcies.Connect()
				if err != nil {
					continue
				}
			}
		}()
	}
	return
}

func (service *ObcyService) LogMessage(message string) {
	log.Println(message)

	_, err := service.discordSession.ChannelMessageSend("628228137527934977", message)
	if err != nil {
		log.Println("Sending discord message failed! Reason:", err)
		return
	}
}

func (service *ObcyService) InjectMessage(sender CommandSender, who, message string) {
	if service.obcies != nil {
		var err error
		switch who {
		case "karol":
			if service.obcies.clientTwo != nil {
				err = service.obcies.clientTwo.WriteMessage(message)
			} else {
				sender.SendMessage("karol is nil")
			}
			break
		case "jan":
			if service.obcies.clientOne != nil {
				err = service.obcies.clientOne.WriteMessage(message)
			} else {
				sender.SendMessage("jan is nil")
			}
			break
		case "all":
			if service.obcies.clientOne != nil {
				err = service.obcies.clientOne.WriteMessage(message)
			} else {
				sender.SendMessage("jan is nil")
			}
			if err != nil {
				log.Println("Write message error", err)
				sender.SendMessage("Write message error")
			}
			if service.obcies.clientTwo != nil {
				err = service.obcies.clientTwo.WriteMessage(message)
			} else {
				sender.SendMessage("karol is nil")
			}
			break
		}
		if err != nil {
			log.Println("Write message error", err)
			sender.SendMessage("Write message error")
		} else {
			sender.SendMessage("Sent message successfully. Receiver: " + who + ", Message:" + message)
		}
	} else {
		sender.SendMessage("Obcies not found!")
	}
}
