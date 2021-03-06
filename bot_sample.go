// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"log"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mattermost/mattermost-server/v5/model"

	tb "gopkg.in/tucnak/telebot.v2"
)

const (
	SAMPLE_NAME = "Mattermost Bot Sample"

	USER_EMAIL    = "bot@example.com"
	USER_PASSWORD = "Password1!"
	USER_NAME     = "samplebot"
	USER_FIRST    = "Sample"
	USER_LAST     = "Bot"

	TEAM_NAME        = "test"
	CHANNEL_LOG_NAME = "test10"
)

var client *model.Client4
var webSocketClient *model.WebSocketClient

var botUser *model.User
var botTeam *model.Team
var debuggingChannel *model.Channel

//Telegram

var telegramClient *tb.Bot

// Documentation for the Go driver can be found
// at https://godoc.org/github.com/mattermost/platform/model#Client
func main() {
	println(SAMPLE_NAME)

	SetupGracefulShutdown()

	client = model.NewAPIv4Client("http://api.ghasedakplatform.ir:8065")

	// Lets test to see if the mattermost server is up and running
	MakeSureServerIsRunning()

	// lets attempt to login to the Mattermost server as the bot user
	// This will set the token required for all future calls
	// You can get this token with client.AuthToken
	LoginAsTheBotUser()

	// If the bot user doesn't have the correct information lets update his profile
	UpdateTheBotUserIfNeeded()

	// Lets find our bot team
	FindBotTeam()

	// This is an important step.  Lets make sure we use the botTeam
	// for all future web service requests that require a team.
	//client.SetTeamId(botTeam.Id)

	// Lets create a bot channel for logging debug messages into
	CreateBotDebuggingChannelIfNeeded()
	SendMsgToDebuggingChannel("_"+SAMPLE_NAME+" has **started** running_", "")

	//// Lets start listening to some channels via the websocket!
	//webSocketClient, err := model.NewWebSocketClient4("ws://api.ghasedakplatform.ir:8065", client.AuthToken)
	//if err != nil {
	//	println("We failed to connect to the web socket")
	//	PrintError(err)
	//}
	//
	//webSocketClient.Listen()
	//
	//go func() {
	//	for resp := range webSocketClient.EventChannel {
	//		HandleWebSocketResponse(resp)
	//	}
	//}()

	telegram()

	// You can block forever with
	select {}
}

var cache sync.Map

//blocking
func telegram() {
	//Telegram
	TelegramLogin()
	telegramClient.Handle(tb.OnChannelPost, func(m *tb.Message) {
		channelName := m.SenderChat.Username
		matterId, ok := cache.Load(channelName)
		if ok {
			println("===========channelname", channelName)
			SendMsgToChannel(m.Text, matterId.(string))
		} else {
			println("the channel is not register: ", channelName)
		}
	})
	telegramManager()
	telegramClient.Start()
}

func telegramManager() {
	telegramClient.Handle("/addchannel", func(m *tb.Message) {
		command := m.Text
		command = strings.ReplaceAll(command, "/addchannel ", "")
		command = strings.TrimSpace(command)
		channels := strings.Split(command, ",")
		if len(channels) != 3 {
			_, err := telegramClient.Send(m.Sender, "???????? ???? ???????? ?????? ???????? ???????? :\n"+
				"telegram-channel,mattermost-channel,mattermost-team")
			if err != nil {
				log.Println(err)
			}
		} else {
			telegramChannel := channels[0]
			mattermostChannel := channels[1]
			mattermostTeam := channels[2]
			matterChannelId, err := getMattermostChannelId(mattermostTeam, mattermostChannel)
			if err != nil {
				log.Println("error in getting channeId", err)
				_, err := telegramClient.Send(m.Sender, "?????? ???? ?????????? ?????????? ??????????")
				if err != nil {
					log.Println(err)
				}
				return
			}

			cache.Store(telegramChannel, matterChannelId)

			log.Println(telegramChannel, matterChannelId)
			_, err = telegramClient.Send(m.Sender, "???? ???????????? ?????????? ????")
			if err != nil {
				log.Println(err)
			}
		}

	})

	telegramClient.Handle("/removechannel", func(m *tb.Message) {
		_, err := telegramClient.Send(m.Sender, "Add Channel!")
		if err != nil {
			log.Println(err)
		}
	})

	telegramClient.Handle(tb.OnText, func(m *tb.Message) {
		_, err := telegramClient.Send(m.Sender, "???????? ?????????? ?????????? ???? ???????? ????????")
		if err != nil {
			log.Println(err)
		}
	})
}

func getMattermostChannelId(teamName, channelName string) (string, error) {
	team, resp := client.GetTeamByName(teamName, "")
	if resp.Error != nil {
		println("We failed to get the initial load")
		println("or we do not appear to be a member of the team '" + teamName + "'")
		return "", resp.Error
	}

	rchannel, resp := client.GetChannelByName(channelName, team.Id, "")
	if resp.Error != nil {
		println("We failed to get the channels")
		return "", resp.Error
	}

	return rchannel.Id, nil

}

func TelegramLogin() {
	var err error
	telegramClient, err = tb.NewBot(tb.Settings{
		Token:  "1959795220:AAHjj73SSkOJuA99Zq370MTOZoV_udhbLfk",
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatal(err)
		return
	}
}

func MakeSureServerIsRunning() {
	if props, resp := client.GetOldClientConfig(""); resp.Error != nil {
		println("There was a problem pinging the Mattermost server.  Are you sure it's running?")
		PrintError(resp.Error)
		os.Exit(1)
	} else {
		println("Server detected and is running version " + props["Version"])
	}
}

func LoginAsTheBotUser() {
	if user, resp := client.Login(USER_EMAIL, USER_PASSWORD); resp.Error != nil {
		println("There was a problem logging into the Mattermost server.  Are you sure ran the setup steps from the README.md?")
		PrintError(resp.Error)
		os.Exit(1)
	} else {
		botUser = user
	}
}

func UpdateTheBotUserIfNeeded() {
	if botUser.FirstName != USER_FIRST || botUser.LastName != USER_LAST || botUser.Username != USER_NAME {
		botUser.FirstName = USER_FIRST
		botUser.LastName = USER_LAST
		botUser.Username = USER_NAME

		if user, resp := client.UpdateUser(botUser); resp.Error != nil {
			println("We failed to update the Sample Bot user")
			PrintError(resp.Error)
			os.Exit(1)
		} else {
			botUser = user
			println("Looks like this might be the first run so we've updated the bots account settings")
		}
	}
}

func FindBotTeam() {
	if team, resp := client.GetTeamByName(TEAM_NAME, ""); resp.Error != nil {
		println("We failed to get the initial load")
		println("or we do not appear to be a member of the team '" + TEAM_NAME + "'")
		PrintError(resp.Error)
		os.Exit(1)
	} else {
		botTeam = team
	}
}

func CreateBotDebuggingChannelIfNeeded() {
	if rchannel, resp := client.GetChannelByName(CHANNEL_LOG_NAME, botTeam.Id, ""); resp.Error != nil {
		println("We failed to get the channels")
		PrintError(resp.Error)
	} else {
		debuggingChannel = rchannel
		return
	}

	// Looks like we need to create the logging channel
	channel := &model.Channel{}
	channel.Name = CHANNEL_LOG_NAME
	channel.DisplayName = "Debugging For Sample Bot"
	channel.Purpose = "This is used as a test channel for logging bot debug messages"
	channel.Type = model.CHANNEL_OPEN
	channel.TeamId = botTeam.Id
	if rchannel, resp := client.CreateChannel(channel); resp.Error != nil {
		println("We failed to create the channel " + CHANNEL_LOG_NAME)
		PrintError(resp.Error)
	} else {
		debuggingChannel = rchannel
		println("Looks like this might be the first run so we've created the channel " + CHANNEL_LOG_NAME)
	}
}

func SendMsgToChannel(msg, channelId string) {
	post := &model.Post{}
	post.ChannelId = channelId
	post.Message = msg

	post.RootId = ""

	if _, resp := client.CreatePost(post); resp.Error != nil {
		println("We failed to send a message to the logging channel")
		PrintError(resp.Error)
	}
}

func SendMsgToDebuggingChannel(msg string, replyToId string) {
	post := &model.Post{}
	post.ChannelId = debuggingChannel.Id
	post.Message = msg

	post.RootId = replyToId

	if _, resp := client.CreatePost(post); resp.Error != nil {
		println("We failed to send a message to the logging channel")
		PrintError(resp.Error)
	}
}

func HandleWebSocketResponse(event *model.WebSocketEvent) {
	HandleMsgFromDebuggingChannel(event)
}

func HandleMsgFromDebuggingChannel(event *model.WebSocketEvent) {
	// If this isn't the debugging channel then lets ingore it
	if event.Broadcast.ChannelId != debuggingChannel.Id {
		return
	}

	// Lets only reponded to messaged posted events
	if event.Event != model.WEBSOCKET_EVENT_POSTED {
		return
	}

	println("responding to debugging channel msg")

	post := model.PostFromJson(strings.NewReader(event.Data["post"].(string)))
	if post != nil {

		// ignore my events
		if post.UserId == botUser.Id {
			return
		}

		// if you see any word matching 'alive' then respond
		if matched, _ := regexp.MatchString(`(?:^|\W)alive(?:$|\W)`, post.Message); matched {
			SendMsgToDebuggingChannel("Yes I'm running", post.Id)
			return
		}

		// if you see any word matching 'up' then respond
		if matched, _ := regexp.MatchString(`(?:^|\W)up(?:$|\W)`, post.Message); matched {
			SendMsgToDebuggingChannel("Yes I'm running", post.Id)
			return
		}

		// if you see any word matching 'running' then respond
		if matched, _ := regexp.MatchString(`(?:^|\W)running(?:$|\W)`, post.Message); matched {
			SendMsgToDebuggingChannel("Yes I'm running", post.Id)
			return
		}

		// if you see any word matching 'hello' then respond
		if matched, _ := regexp.MatchString(`(?:^|\W)hello(?:$|\W)`, post.Message); matched {
			SendMsgToDebuggingChannel("Yes I'm running", post.Id)
			return
		}
	}

	SendMsgToDebuggingChannel("I did not understand you!", post.Id)
}

func PrintError(err *model.AppError) {
	println("\tError Details:")
	println("\t\t" + err.Message)
	println("\t\t" + err.Id)
	println("\t\t" + err.DetailedError)
}

func SetupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			if webSocketClient != nil {
				webSocketClient.Close()
			}

			SendMsgToDebuggingChannel("_"+SAMPLE_NAME+" has **stopped** running_", "")
			os.Exit(0)
		}
	}()
}
