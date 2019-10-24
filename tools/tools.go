package tools

import (
	"database/sql"
	"fmt"

	"github.com/bwmarrin/discordgo"

	"github.com/jinzhu/configor"

	"github.com/gidoBOSSftw5731/log"
)

//Config is a struct containing the configuration files
type Config struct {
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
}

//StartSQL returns a database
func StartSQL(user, password, ip, port string) *sql.DB {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/pinnerboibot",
		user, password, ip, port))
	if err != nil {
		log.Error("Oh noez, could not connect to database")
		log.Errorf("Error in SQL! %v", err)
	}
	log.Debug("Oi, mysql did thing")
	//defer db.Close()

	return db
}

//Configor returns the config as an object
func Configor(config *Config, path string) {
	configor.Load(config, path)
}

//DiscordSession returns a discord session
func DiscordSession(config Config) (*discordgo.Session, error) {
	discord, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		return nil, err
	}
	err = discord.Open()
	if err != nil {
		return nil, err
	}

	return discord, nil
}
