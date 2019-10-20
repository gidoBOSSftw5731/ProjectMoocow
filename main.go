package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gidoBOSSftw5731/log"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/configor"
)

var config = struct {
	APPName string `default:"PinnerBoi"`
	Author  string `default:"gidoBOSSftw5731#6422"`

	Prefix string `default:"📌 "`
	Token  string `required:"true"`

	DB struct {
		User     string `default:"pinnerboi"`
		Password string `required:"true" env:"DBPassword"`
		Port     string `default:"3306"`
		IP       string `default:"127.0.0.1"`
	}
}{}

const (
	pinReaction string = "📌"
	precision   int    = 5
)

//var allChannelIDs = map[string]string{}
var allChannelIDs []channelInfo

type channelInfo struct {
	ChannelID string
	GuildID   string
}

var (
	botID string
	//discordToken  = flag.String("token", "", "Discord bot secret")
	//commandPrefix = "📌 "
	//author        = ""
)

func main() {
	configor.Load(&config, "config.yml")

	//println(config.Token)

	log.SetCallDepth(4)

	discord, err := discordgo.New("Bot " + config.Token)
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

	go collectAllChannelIDs(discord)

	<-make(chan struct{})
}

func errCheck(msg string, err error) {
	if err != nil {
		log.Fatalf("%s: %+v", msg, err)
	}
}

func collectAllChannelIDs(s *discordgo.Session) {
	for _, guild := range s.State.Guilds {
		channels, err := s.GuildChannels(guild.ID)
		if err != nil {
			log.Errorln(err)
			continue
		}
		for _, c := range channels {
			// Check if channel is a guild text channel and not a voice or DM channel
			if c.Type != discordgo.ChannelTypeGuildText {
				continue
			}

			//allChannelIDs = append(allChannelIDs, c.ID)
			//allChannelIDs[c.ID] = c.GuildID
			allChannelIDs = append(allChannelIDs, channelInfo{ChannelID: c.ID, GuildID: c.GuildID})
		}
	}

	//log.Tracef("All channel IDs: \n%v", allChannelIDs)
	autoChecker(s)

}

func autoChecker(s *discordgo.Session) {

	iterations := make(map[channelInfo]int)
	for _, channel := range allChannelIDs {
		id, err := strconv.Atoi(channel.ChannelID)
		if err != nil {
			log.Errorln(err)
			continue
		}
		iterations[channel] = id % precision
	}

	//log.Traceln(iterations)

	db := startSQL()

	for {
		now := time.Now().Unix() % int64(precision)
		var current []channelInfo

		for channel := range iterations {
			if int64(iterations[channel]) == now {
				current = append(current, channel)
			}
		}

		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			for _, channel := range current {

				wg.Add(1)
				check(s, &channel, db, &wg)
			}
		}()

		//wg.Done()
		//wg.Wait()
		time.Sleep(1 * time.Second)
	}

}

func check(s *discordgo.Session, channel *channelInfo, db *sql.DB, wg *sync.WaitGroup) {
	last25, err := s.ChannelMessages(channel.ChannelID, 25, "", "", "")
	if err != nil {
		log.Errorln(err)
		return
	}
	checkAndPin(last25, db, channel.GuildID)
	//wg.Done()
}

func commandHandler(discord *discordgo.Session, message *discordgo.MessageCreate) {
	if message.Content == "" || len(message.Content) < len(config.Prefix) {
		return
	}
	if message.Content[:len(config.Prefix)] != config.Prefix ||
		len(strings.Split(message.Content, config.Prefix)) < 2 {
		return
	}

	log.Debugln("prefix found")

	command := strings.Split(message.Content, config.Prefix)[1]
	commandContents := strings.Split(message.Content, " ") // 0 = !, 1 = command, 2 = first arg, etc

	db := startSQL()

	if len(command) < 2 {
		log.Errorln("No command sent")
		return
	}

	switch strings.Split(command, " ")[1] {
	case "returnreacts":

		reactedMsg, err := discord.ChannelMessage(message.ChannelID, commandContents[2])
		if err != nil {
			return
		}

		fmt.Println("Reactions:")

		for _, reaction := range reactedMsg.Reactions {
			fmt.Println(reaction.Emoji.Name)
		}

		//discord.ChannelMessageSend(message.ChannelID, message.Reactions[1])

	// in-joke, not functional
	case "alcegf":
		discord.ChannelMessageSend(message.ChannelID, "HE NEEDS YOU <@613131233852915733>")
	case "chkpin":

		last100, err := discord.ChannelMessages(message.ChannelID, 100, message.ID, "", "")
		if err != nil {
			log.Errorln(err)
			return
		}

		err = checkAndPin(last100, db, message.GuildID)
		if err != nil {
			log.Errorln(err)
			return
		}

	case "updatepin":
		reactedMsg, err := discord.ChannelMessage(message.ChannelID, commandContents[2])
		if err != nil {
			return
		}

		last100, err := discord.ChannelMessages(reactedMsg.ChannelID, 100, "", "", reactedMsg.ID)
		if err != nil {
			log.Errorln(err)
			return
		}

		err = checkAndPin(last100, db, message.GuildID)
		if err != nil {
			log.Errorln(err)
			return
		}

	}

}

func pinsWithInfo(serverID, channelID string, discord *discordgo.Session) ([]*discordgo.Message, error) {
	var messages []*discordgo.Message

	db := startSQL()

	rows, err := db.Query("SELECT messageid FROM pinnedmessages WHERE serverid=? AND channelid=?", serverID, channelID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var messageid string
		rows.Scan(&messageid)
		message, err := discord.ChannelMessage(channelID, messageid)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	if rows.Err() != nil {
		return nil, err
	}

	return messages, err
}

//Webpage is a function that returns an HTML file as a string to be sent to a user.
func Webpage(channelID, serverID string, discord *discordgo.Session) (string, error) {
	var output string

	t := template.New("pins")
	t, err := t.ParseFiles("templates/template.html")
	if err != nil {
		return output, err
	}

	messages, err := pinsWithInfo(serverID, channelID, discord)
	if err != nil {
		return output, err
	}

	messagetmpl, err := messageTemplater(messages)
	if err != nil {
		return output, err
	}

	var buf bytes.Buffer

	t.Execute(&buf, messagetmpl)
	output = fmt.Sprintln(buf)

	return output, err
}

func messageTemplater(messages []*discordgo.Message) (string, error) {
	var output string

	for _, message := range messages {
		t := template.New("message")
		t, err := t.ParseFiles("temlates/messagetmpl.html")
		if err != nil {
			return output, err
		}

		var buf bytes.Buffer

		t.Execute(&buf, message)

		output += fmt.Sprintln(buf)
		output += "\n"
	}

	return output, nil
}

func checkAndPin(last100 []*discordgo.Message, db *sql.DB, serverID string) error {
	for messageIndex := 0; messageIndex < len(last100); messageIndex++ {
		msg := *last100[messageIndex]
		reaction := msg.Reactions
		for reactionIndex := 0; reactionIndex < len(reaction); reactionIndex++ {
			if reaction[reactionIndex].Emoji.Name != pinReaction {
				continue
			}
			//At this point, we know the command was "pinned"

			var tmpptr string //throwaway var

			err := db.QueryRow("SELECT * FROM pinnedmessages WHERE channelid=? AND messageid=?",
				msg.ChannelID, msg.ID).Scan(&tmpptr, &tmpptr, &tmpptr)

			switch {
			case err == sql.ErrNoRows:
				log.Debug("New pin, adding..")
				_, err := db.Exec("INSERT INTO pinnedmessages VALUES(?, ?, ?)",
					serverID, msg.ChannelID, msg.ID)
				if err != nil {
					log.Error(err)
					return err
				}
				log.Debug("Added pin info to table")
			case err != nil:
				log.Error(err)
				return err
			default:
				continue
			}

		}
	}
	return nil
}

func startSQL() *sql.DB {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/pinnerboibot",
		config.DB.User, config.DB.Password, config.DB.IP, config.DB.Port))
	if err != nil {
		log.Error("Oh noez, could not connect to database")
		log.Errorf("Error in SQL! %v", err)
	}
	log.Debug("Oi, mysql did thing")
	//defer db.Close()

	return db
}
