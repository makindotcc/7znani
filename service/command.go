package service

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"strconv"
	"strings"
	"time"
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
	Author    string
}

func NewDiscordSender(session *discordgo.Session, channelId string, author string) *DiscordSender {
	return &DiscordSender{session: session, ChannelId: channelId, Author: author}
}

func (sender *DiscordSender) SendMessage(message string) {
	_, err := sender.session.ChannelMessageSend(sender.ChannelId, message)
	if err != nil {
		log.Println("DiscordSender send message failed! Reason:", err)
	}
}

func (sender *DiscordSender) EditMessage(messageId string, message string) {
	_, err := sender.session.ChannelMessageEdit(sender.ChannelId, messageId, message)
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
		service.Handle(NewDiscordSender(session, message.ChannelID, message.Author.ID), message.Content)
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

	discordSender := sender.(*DiscordSender)
	if discordSender.Author == "524346987865178123" {
		sender.SendMessage("spierdalaj")
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
	if len(args) < 3 {
		sender.SendMessage("Usage: <session_id> <karol/jan/all> <message...>")
		return
	}

	arg0 := args[0]
	sessionId, err := strconv.Atoi(arg0)
	if err != nil {
		sender.SendMessage("cyumbale session id masz wpisac a nie " + arg0)
		return
	}

	command.service.InjectMessage(sender, sessionId, args[1], strings.Join(args[2:], " "))
	return
}

type ChatsCommand struct {
	service         *ObcyService
	updateMessageId string
}

func NewChatsCommand(service *ObcyService) *ChatsCommand {
	return &ChatsCommand{service: service}
}

func (command *ChatsCommand) Name() string {
	return "chats"
}

func (command *ChatsCommand) Handle(sender CommandSender, args []string) (err error) {
	minLength := 1
	if len(args) >= 1 {
		minLength, err = strconv.Atoi(args[0])
		if err != nil {
			sender.SendMessage("masz podac numer d-.-b")
			return
		}
	}

	discordSender := sender.(*DiscordSender)
	if discordSender != nil {
		message, err := discordSender.session.ChannelMessageSend(discordSender.ChannelId,
			command.generateMessage(minLength))
		if err != nil {
			return err
		}
		command.updateMessageId = message.ID
		go func() {
			const maxEdits = 500
			edits := 0
			for {
				time.Sleep(1 * time.Second)
				if command.updateMessageId != message.ID {
					return
				}

				discordSender.EditMessage(command.updateMessageId, command.generateMessage(minLength))
				edits++
				if edits > maxEdits {
					return
				}
			}
		}()
	} else {
		sender.SendMessage(command.generateMessage(minLength))
	}

	return
}

func (command *ChatsCommand) generateMessage(minLength int) string {
	builder := strings.Builder{}
	command.service.SessionsForEach(func(session *Obcies) {
		session.chatMutex.RLock()
		if len(session.chatHistory) >= minLength {
			builder.WriteString(fmt.Sprintf("``Chat (id: %d):``\n", session.sessionId))
			for _, message := range session.chatHistory {
				builder.WriteString(message)
				builder.WriteByte('\n')
			}
		}

		session.chatMutex.RUnlock()
	})

	if builder.Len() == 0 {
		builder.WriteString("Brak historii czatu spelniajacych podane wymaganie")
	}

	return builder.String()
}
