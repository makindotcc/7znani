package service

import (
	"github.com/bwmarrin/discordgo"
	"log"
	"sort"
	"sync"
	"time"
)

type ObcyService struct {
	obciesMap             map[int]*Obcies
	obciesMutex           *sync.RWMutex
	consoleCommandService *ConsoleCommandService
	discordCommandService *DiscordCommandService
	discordSession        *discordgo.Session
	obcyPool              *ObcyPool
}

func (service *ObcyService) ConsoleCommandService() *ConsoleCommandService {
	return service.consoleCommandService
}

func NewObcyService() *ObcyService {
	service := &ObcyService{
		consoleCommandService: NewConsoleCommandService(),
		discordCommandService: NewDiscordCommandService(),
		obciesMutex:           &sync.RWMutex{},
		obciesMap:             make(map[int]*Obcies, 30),
		obcyPool:              NewObcyPool(),
	}
	service.registerCommand(NewSendCommand(service))
	service.registerCommand(NewChatsCommand(service))
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
		time.Sleep(1 * time.Second)
		go func() {
			for {
				log.Println("--- nowa rozmowa ---")

				// sync counter btw im lazy
				service.obciesMutex.Lock()
				obcies := NewObcies(service)
				service.obciesMutex.Unlock()

				service.AddSession(obcies)

				err = obcies.Connect()
				if err != nil {
					log.Println("Session connect failed:", err)
				}
				service.DeleteSession(obcies)
				time.Sleep(3 * time.Second)
			}
		}()
	}
	return
}

func (service *ObcyService) SessionsForEach(receiver func(session *Obcies)) {
	service.obciesMutex.RLock()

	keys := make([]int, 0, len(service.obciesMap))
	for k := range service.obciesMap {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(keys)))
	for _, session := range keys {
		receiver(service.obciesMap[session])
	}
	service.obciesMutex.RUnlock()
}

func (service *ObcyService) AddSession(pair *Obcies) {
	service.obciesMutex.Lock()
	service.obciesMap[pair.sessionId] = pair
	service.obciesMutex.Unlock()
}

func (service *ObcyService) DeleteSession(pair *Obcies) {
	service.obciesMutex.Lock()
	delete(service.obciesMap, pair.sessionId)
	service.obciesMutex.Unlock()
}

func (service *ObcyService) Session(sessionId int) *Obcies {
	service.obciesMutex.RLock()
	session := service.obciesMap[sessionId]
	service.obciesMutex.RUnlock()

	return session
}

func (service *ObcyService) LogMessage(message string) {
	log.Println(message)

	_, err := service.discordSession.ChannelMessageSend("628228137527934977", message)
	if err != nil {
		log.Println("Sending discord message failed! Reason:", err)
		return
	}
}

func (service *ObcyService) InjectMessage(sender CommandSender, sessionId int, who, message string) {
	pair := service.Session(sessionId)
	if pair == nil {
		sender.SendMessage("nie znaleziono takiej sesji byq")
		return
	}

	var err error
	switch who {
	case "karol":
		if pair.clientTwo != nil {
			err = pair.clientTwo.WriteMessage(message)
		} else {
			sender.SendMessage("karol is nil")
		}
		break
	case "jan":
		if pair.clientOne != nil {
			err = pair.clientOne.WriteMessage(message)
		} else {
			sender.SendMessage("jan is nil")
		}
		break
	case "all":
		if pair.clientOne != nil {
			err = pair.clientOne.WriteMessage(message)
		} else {
			sender.SendMessage("jan is nil")
		}
		if err != nil {
			log.Println("Write message error", err)
			sender.SendMessage("Write message error")
		}
		if pair.clientTwo != nil {
			err = pair.clientTwo.WriteMessage(message)
		} else {
			sender.SendMessage("karol is nil")
		}
		break
	default:
		sender.SendMessage("nieprawidlowe uzycie")
		return
	}
	if err != nil {
		log.Println("Write message error", err)
		sender.SendMessage("Write message error")
	} else {
		sender.SendMessage("Sent message successfully. Receiver: " + who + ", Message: " + message)
	}
}
