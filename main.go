package main

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Token Variables used for command line parameters
var (
	Token string
)

func init() {
	Token = os.Getenv("TOKEN")
}

func main() {

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}
	status, err := ioutil.ReadFile("status.text")
	if err != nil {
		println("Could not read status.text file, " + err.Error())
	}
	err = dg.UpdateGameStatus(0, string(status))
	if err != nil {
		return
	}
	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	err = dg.Close()
	if err != nil {
		println(err)
	}
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}
	// If the message is "ping" reply with "Pong!"
	if m.Content == "e!ping" {
		var _, err = s.ChannelMessageSendReply(
			m.ChannelID,
			strconv.FormatInt(s.HeartbeatLatency().Milliseconds(), 10)+"ms",
			m.Reference())
		if err != nil {
			println(err.Error())
		}
	}
	if strings.HasPrefix(m.Content, "e!status") && m.Author.ID == "861733561463603240" {
		args := strings.Split(m.Content, " ")
		args = append(args[:0], args[1:]...)
		err := s.UpdateGameStatus(0, strings.Join(args, " "))
		if err != nil {
			err := SendWithSelfDelete(s, m.ChannelID, "Failed to update status: "+err.Error())
			if err != nil {
				return
			}
		}
		err = ioutil.WriteFile("status.text", []byte(strings.Join(args, " ")), fs.FileMode(0777))
		if err != nil {
			err := SendWithSelfDelete(s, m.ChannelID, "Failed to write to file: "+err.Error())
			if err != nil {
				return
			}
		}
		_, err = s.ChannelMessageSendReply(m.ChannelID, "Changed status to "+strings.Join(args, " "), m.Reference())
		if err != nil {
			return
		}
	}
	if strings.HasPrefix(m.Content, "e!clean") {
		p, err := s.UserChannelPermissions(m.Author.ID, m.ChannelID)
		if err != nil {
			fmt.Println(err.Error())
		}

		if !(p&discordgo.PermissionManageMessages == discordgo.PermissionManageMessages) {
			err := SendWithSelfDelete(s, m.ChannelID, "You don't have permission to run that command!")
			if err != nil {
				return
			}
		}
		args := strings.Split(m.Content, " ")
		var count int
		if len(args) >= 2 {
			var err error
			count, err = strconv.Atoi(strings.Split(m.Content, " ")[1])
			if err != nil {
				return
			}
		} else {
			err := SendWithSelfDelete(s, m.ChannelID, "This command requires a count, e.g. `e!clean 10`")
			if err != nil {
				return
			}
			return
		}

		workingMessage, err := s.ChannelMessageSend(m.ChannelID, "Cleaning channel...")
		if err != nil {
			println(err.Error())
		}
		requestsNeeded := count / 100
		for i := 1; i < requestsNeeded; i++ {
			println(count / requestsNeeded)
			messages, err := s.ChannelMessages(m.ChannelID, count, m.ID, "", "")
			var messageIDs = make([]string, 0)
			messageIDs = append(messageIDs, m.ID)
			for _, message := range messages {
				messageIDs = append(messageIDs, message.ID)
			}
			if err != nil {
				println(err.Error())
			}
			err = s.ChannelMessagesBulkDelete(
				m.ChannelID,
				messageIDs)
			if err != nil {
				println(err.Error())
			}
		}
		messages, err := s.ChannelMessages(m.ChannelID, count%100, m.ID, "", "")
		var messageIDs = make([]string, 0)
		messageIDs = append(messageIDs, m.ID)
		for _, message := range messages {
			messageIDs = append(messageIDs, message.ID)
		}
		if err != nil {
			println(err.Error())
		}
		err = s.ChannelMessagesBulkDelete(
			m.ChannelID,
			messageIDs)
		if err != nil {
			println(err.Error())
		}
		err = s.ChannelMessageDelete(workingMessage.ChannelID, workingMessage.ID)
		if err != nil {
			return
		}
		err = SendWithSelfDelete(s, m.ChannelID, "Cleaned channel!")
		if err != nil {
			return
		}
	}
}
func SendWithSelfDelete(ds *discordgo.Session, channelId, message string) error {
	m, err := ds.ChannelMessageSend(channelId, message)
	if err != nil {
		return err
	}

	go func(ch, id string, session *discordgo.Session) {
		<-time.After(10 * time.Second)
		_ = ds.ChannelMessageDelete(channelId, m.ID)
	}(channelId, m.ID, ds)
	return nil
}
