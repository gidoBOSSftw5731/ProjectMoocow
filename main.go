package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/gidoBOSSftw5731/log"
)

var (
	botID         string
	discordToken  = flag.String("token", "", "Discord bot secret")
	commandPrefix = "📌 "
	author        = ""
)

func main() {
	flag.Parse()
	log.SetCallDepth(4)

	discord, err := discordgo.New("Bot " + *discordToken)
	errCheck("error creating discord session", err)
	user, err := discord.User("@me")
	errCheck("error retrieving account", err)

	botID = user.ID
	discord.AddHandler(commandHandler)
	discord.AddHandler(func(discord *discordgo.Session, ready *discordgo.Ready) {
		err = discord.UpdateStatus(2, "Pin all the things!")
		if err != nil {
			log.Errorln("Error attempting to set my status")
		}
		servers := discord.State.Guilds
		log.Debugf("PinnerBoi has started on %d servers", len(servers))
	})

	err = discord.Open()
	errCheck("Error opening connection to Discord", err)
	defer discord.Close()

	<-make(chan struct{})
}

func errCheck(msg string, err error) {
	if err != nil {
		log.Fatalf("%s: %+v", msg, err)
	}
}

func commandHandler(discord *discordgo.Session, message *discordgo.MessageCreate) {
	if message.Content[:len(commandPrefix)] != commandPrefix ||
		len(strings.Split(message.Content, commandPrefix)) < 2 {
		return
	}

	command := strings.Split(message.Content, commandPrefix)[1]
	commandContents := strings.Split(message.Content, " ") // 0 = !command, 2 = first arg, etc

	switch strings.Split(command, " ")[0] {
	case "returnreacts":
		reactedMsg, err := discord.ChannelMessage(message.ChannelID, commandContents[2])
		if err != nil {
			return
		}

		fmt.Println("Reactions:")

		for _, reaction := range reactedMsg.Reactions {
			fmt.Println(reaction)
		}
		//discord.ChannelMessages()
		//discord.ChannelMessageSend(message.ChannelID, message.Reactions[1])
	}

}
