package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/pprof"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gidoBOSSftw5731/log"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/jinzhu/configor"

	"github.com/gidoBOSSftw5731/ProjectMoocow/tools"
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
	precision   int    = 10
)

// var allChannelIDs = map[string]string{}
var allChannelIDs []channelInfo

type channelInfo struct {
	ChannelID string
	GuildID   string
}

var helpMenu discordgo.MessageEmbed

var (
	botID string
	//discordToken  = flag.String("token", "", "Discord bot secret")
	//commandPrefix = "📌 "
	//author        = ""
	db *sql.DB
)

func main() {
	configor.Load(&config, "config.yml")

	// Create a new router
	router := mux.NewRouter()

	// Register pprof handlers
	router.HandleFunc("/debug/pprof/", pprof.Index)
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

	router.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	router.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	router.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	router.Handle("/debug/pprof/block", pprof.Handler("block"))

	go http.ListenAndServe("192.110.255.55:56799", router)

	helpMenu = discordgo.MessageEmbed{
		Title:  "PinnerBoi Help",
		Author: &discordgo.MessageEmbedAuthor{},
		Color:  rand.Intn(16777215),
		Description: fmt.Sprintf("Prefix is %v \n All commands are case sensitive \n React to any message with %v to have it be pinned \n Bot made by %v",
			config.Prefix, pinReaction, config.Author),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "help",
				Value: "Returns this message",
			},
			{
				Name:  "site",
				Value: "Returns this channel's site",
			},
			{
				Name:  "chkpin",
				Value: "Checks last 100 messages for new pins. The bot checks the last 25 messages every 5 seconds by default.",
			},
			{
				Name:  "invite",
				Value: "Returns the invite URL",
			},
			{
				Name:  "Where is the code?",
				Value: "This bot's code is accessible at https://github.com/gidoBOSSftw5731/ProjectMoocow \n My personal code can be found at https://imagen.click/git",
			},
		},

		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://imagen.click/i/9e1233.png",
		},
		Timestamp: time.Now().Format(time.RFC3339), // Discord wants ISO8601; RFC3339 is an extension of ISO8601 and should be completely compatible.
	}

	//println(config.Token)

	log.SetCallDepth(4)

	discord, err := discordgo.New("Bot " + config.Token)
	errCheck("error creating discord session", err)
	user, err := discord.User("@me")
	errCheck("error retrieving account", err)

	botID = user.ID
	discord.AddHandler(commandHandler)
	discord.AddHandler(func(discord *discordgo.Session, ready *discordgo.Ready) {
		servers := discord.State.Guilds

		err = discord.UpdateGameStatus(2, fmt.Sprintf("Pin all the things! | %vhelp | Pinning in %v servers!",
			config.Prefix, len(servers)))
		if err != nil {
			log.Errorln("Error attempting to set my status")
		}

		log.Debugf("PinnerBoi has started on %d servers", len(servers))
	})

	db = startSQL()

	err = discord.Open()
	errCheck("Error opening connection to Discord", err)
	defer discord.Close()

	discord.AddHandler(func(discord *discordgo.Session, reaction *discordgo.MessageReactionAdd) {
		if reaction.UserID == botID {
			return
		}

		if reaction.Emoji.Name != pinReaction {
			return
		}

		// get message by ID
		msg, err := discord.ChannelMessage(reaction.ChannelID, reaction.MessageID)
		if err != nil {
			log.Errorln(err)
			return
		}

		// get channel object
		channel, err := discord.Channel(reaction.ChannelID)
		if err != nil {
			log.Errorln(err)
			return
		}

		addMsgToDB(msg, channel.GuildID)
		log.Debugln("Added message to DB via Gateway")

	})

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
	// don't use autochecker anymore, replace with intents and gw
	//autoChecker(s)

}

// autoChecker checks for recent messages frequently.
// This is DEPRECATED due to the Gateway handler.
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
				check(s, &channel, &wg)
			}
		}()

		//wg.Done()
		//wg.Wait()
		time.Sleep(1 * time.Second)
	}

}

func check(s *discordgo.Session, channel *channelInfo, wg *sync.WaitGroup) {
	last25, err := s.ChannelMessages(channel.ChannelID, 25, "", "", "")

	if err != nil {
		if strings.Contains(err.Error(), "HTTP 403 Forbidden, {\"message\": \"Missing Access\", \"code\": 50001}") {
			return
		}
		log.Errorln(err)
		return
	}

	checkAndPin(last25, channel.GuildID)
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

	if len(command) < 2 {
		log.Errorln("No command sent")
		return
	}

	switch strings.Split(command, " ")[1] {
	case "chkpin":
		last100, err := discord.ChannelMessages(message.ChannelID, 100, message.ID, "", "")
		if err != nil {
			log.Errorln(err)
			return
		}

		err = checkAndPin(last100, message.GuildID)
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

		err = checkAndPin(last100, message.GuildID)
		if err != nil {
			log.Errorln(err)
			return
		}

	case "site", "website":
		discord.ChannelMessageSend(message.ChannelID, fmt.Sprintf("https://imagen.click/pinbot/%v/%v", message.GuildID, message.ChannelID))

	case "invite":
		discord.ChannelMessageSend(message.ChannelID,
			"https://discordapp.com/oauth2/authorize?client_id=631313783545004032&permissions=305216&redirect_uri=https%3A%2F%2Fdiscordapp.com%2Foauth2%2Fauthorize%3F%26client_id%3D181965297249550336%26scope%3Dbot&scope=bot")

	case "help", "howto", "aahhhh", "sendhelp", "aid", "ayudame", "yonose", "info":
		resp, err := discord.ChannelMessageSendEmbed(message.ChannelID, &helpMenu)
		if err != nil {
			log.Debugln(resp, err)
			discord.ChannelMessageSend(message.ChannelID, "Internal Error!")
			return
		}

	}

}

func checkAndPin(last100 []*discordgo.Message, serverID string) error {
	for messageIndex := 0; messageIndex < len(last100); messageIndex++ {
		msg := *last100[messageIndex]
		reaction := msg.Reactions

		isValid := tools.CheckIfValid(reaction, pinReaction)
		if !isValid {
			continue
		}

		// at this point the message is known good

		err := addMsgToDB(&msg, serverID)
		if err != nil {
			log.Errorln(err)
			return err
		}
	}
	return nil
}

// addMsgToDB adds a message to the SQL Database and checks for error,
// does NOT verify if the reaction is actually there.
func addMsgToDB(msg *discordgo.Message, serverID string) error {
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
		// Pin already exists, do nothing
	}
	return nil
}

func startSQL() *sql.DB {
	db := tools.StartSQL(config.DB.User, config.DB.Password, config.DB.IP, config.DB.Port)

	return db
}
