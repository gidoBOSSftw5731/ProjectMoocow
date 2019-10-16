package main

import (
	"database/sql"
	"fmt"
	"strings"

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
)

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

	<-make(chan struct{})
}

func errCheck(msg string, err error) {
	if err != nil {
		log.Fatalf("%s: %+v", msg, err)
	}
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
	commandContents := strings.Split(message.Content, " ") // 0 = !command, 2 = first arg, etc

	db := startSQL()

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

	case "chkpin":

		last100, err := discord.ChannelMessages(message.ChannelID, 100, message.ID, "", "")
		if err != nil {
			log.Errorln(err)
			return
		}

		msg, err := discord.ChannelMessage(message.ChannelID, message.ID)
		if err != nil {
			log.Errorln(err)
			return
		}

		err = checkAndPin(last100, db, msg, message.GuildID)
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

		err = checkAndPin(last100, db, reactedMsg, message.GuildID)
		if err != nil {
			log.Errorln(err)
			return
		}

	}

}

func checkAndPin(last100 []*discordgo.Message, db *sql.DB, message *discordgo.Message, serverID string) error {
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
				serverID, msg.ID).Scan(&tmpptr, &tmpptr, &tmpptr)

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
