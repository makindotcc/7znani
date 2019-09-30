package service

import (
	"github.com/bwmarrin/discordgo"
	"log"
	"strings"
)

type CommandSender interface {
	SendMessage(message string)
}

type ConsoleSender struct {
}

func NewConsoleSender() *ConsoleSender {
	return &ConsoleSender{}
}

func (sender *ConsoleSender) SendMessage(message string) {
	log.Println("ConsoleSender <", message)
}

type Command interface {
	Name() string
	Handle(sender CommandSender, args []string) (err error)
}

type CommandService interface {
	Find(name string) Command
	Register(command Command)
	Handle(sender CommandSender, text string)
}

type CommandRegistry struct {
	command map[string]Command
}

func (registry *CommandRegistry) Register(command Command) {
	registry.command[command.Name()] = command
}

func (registry *CommandRegistry) Find(name string) Command {
	return registry.command[name]
}

func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		command: make(map[string]Command, 0),
	}
}

type DiscordSender struct {
	session   *discordgo.Session
	ChannelId string
}

func NewDiscordSender(session *discordgo.Session, channelId string) *DiscordSender {
	return &DiscordSender{session: session, ChannelId: channelId}
}

func (sender *DiscordSender) SendMessage(message string) {
	_, err := sender.session.ChannelMessageSend(sender.ChannelId, message)
	if err != nil {
		log.Println("DiscordSender send message failed! Reason:", err)
	}
}

type DiscordCommandService struct {
	commandRegistry *CommandRegistry
	session         *discordgo.Session
}

func NewDiscordCommandService() *DiscordCommandService {
	return &DiscordCommandService{commandRegistry: NewCommandRegistry()}
}

func (service *DiscordCommandService) Attach(session *discordgo.Session) {
	service.session = session
	service.session.AddHandler(func(session *discordgo.Session, message *discordgo.MessageCreate) {
		service.Handle(NewDiscordSender(session, message.ChannelID), message.Content)
	})
}

func (service *DiscordCommandService) Find(name string) Command {
	return service.commandRegistry.Find(name)
}

func (service *DiscordCommandService) Register(command Command) {
	service.commandRegistry.Register(command)
}

func (service *DiscordCommandService) Handle(sender CommandSender, text string) {
	if len(text) < 2 {
		return
	}

	if text[0] != '!' {
		return
	}

	splitted := strings.Split(text[1:], " ")
	name := splitted[0]
	command := service.commandRegistry.Find(name)
	if command == nil {
		sender.SendMessage("Command not found!")
		return
	}

	err := command.Handle(sender, splitted[1:])
	if err != nil {
		sender.SendMessage("Command error occurred.")
		log.Println(err.Error())
	}
}

type ConsoleCommandService struct {
	commandRegistry *CommandRegistry
}

func (service *ConsoleCommandService) Find(name string) Command {
	return service.commandRegistry.Find(name)
}

func (service *ConsoleCommandService) Register(command Command) {
	service.commandRegistry.Register(command)
}

func (service *ConsoleCommandService) Handle(sender CommandSender, text string) {
	splitted := strings.Split(text, " ")
	name := splitted[0]
	command := service.commandRegistry.Find(name)
	if command == nil {
		sender.SendMessage("Command not found!")
		return
	}

	err := command.Handle(sender, splitted[1:])
	if err != nil {
		sender.SendMessage("Command error occurred:")
		sender.SendMessage(err.Error())
	}
}

func NewConsoleCommandService() *ConsoleCommandService {
	return &ConsoleCommandService{
		commandRegistry: NewCommandRegistry(),
	}
}

/* impl */

type SendCommand struct {
	service *ObcyService
}

func NewSendCommand(service *ObcyService) *SendCommand {
	return &SendCommand{service: service}
}

func (command *SendCommand) Name() string {
	return "send"
}

func (command *SendCommand) Handle(sender CommandSender, args []string) (err error) {
	sender.SendMessage("Handling command...")
	if len(args) < 2 {
		sender.SendMessage("Usage: <karol/jan/all> <message...>")
		return
	}
	command.service.InjectMessage(sender, args[0], strings.Join(args[1:], " "))
	return
}
