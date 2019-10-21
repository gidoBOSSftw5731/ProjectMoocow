package web

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"

	"../tools"
	"github.com/bwmarrin/discordgo"
)

//MsgStruct is a struct to hold all essential information for the template
type MsgStruct struct {
	Author  string
	Content string
	GID     string
	CID     string
	ID      string
	Time    string
}

//SQLInfo is a struct that contains info to connect to SQL
type SQLInfo struct {
	User     string
	Password string
	IP       string
	Port     string
}

//Webpage is a function that returns an HTML file as a string to be sent to a user.
func Webpage(channelID, serverID string, discord *discordgo.Session, sql SQLInfo) (string, error) {
	var output string

	file, err := ioutil.ReadFile("templates/template.html")
	if err != nil {
		return "", err
	}

	t := template.New("pins")
	t, err = t.Parse(string(file))
	if err != nil {
		return output, err
	}

	messages, err := pinsWithInfo(serverID, channelID, discord, sql)
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

	return output, nil
}

func messageTemplater(messages []*discordgo.Message) (string, error) {
	var output string

	for _, Message := range messages {
		msg := msgToStruct(Message)

		file, err := ioutil.ReadFile("templates/messagetmpl.html")
		if err != nil {
			return "", err
		}

		t := template.New("message")
		t, err = t.Parse(string(file))
		if err != nil {
			return output, err
		}

		var buf bytes.Buffer

		err = t.Execute(&buf, msg)
		if err != nil {
			return output, err
		}

		//log.Traceln(buf)
		output += fmt.Sprintln(buf)
		output += "\n"
	}

	return output, nil
}

func pinsWithInfo(serverID, channelID string, discord *discordgo.Session, sql SQLInfo) ([]*discordgo.Message, error) {
	var messages []*discordgo.Message

	db := tools.StartSQL(sql.User, sql.Password, sql.IP, sql.Port)

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

	return messages, nil
}

func msgToStruct(message *discordgo.Message) MsgStruct {
	return MsgStruct{
		message.Author.Username,
		message.Content,
		message.GuildID,
		message.ChannelID,
		message.ID,
		string(message.Timestamp)}
}
