package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"

	"github.com/bwmarrin/discordgo"
	"github.com/psheets/ddgquery"
	"github.com/psheets/detbot/query"
)

// Variables used for command line parameters

// Token stores the app token to run bot as

var db *gorm.DB

var err error

type Configuration struct {
	Token   string
}

type MessageStoreMessage struct {
	gorm.Model
	MessageID     string
	DiscordUserID string
	GuildID       string
}
type User struct {
	gorm.Model
	DiscordUserID string
	Nicknames     []Nickname `gorm:"ForeignKey:DiscordUserID"`
}

type Nickname struct {
	gorm.Model
	Nickname      string
	GuildID       string
	DiscordUserID string
}

type ChannelProp struct {
	gorm.Model
	ChannelID string
	Ticker    bool `gorm:"default:false"`
}

func NewMessageEmbed() *discordgo.MessageEmbed {
	var ef = new(discordgo.MessageEmbedFooter)
	ef.Text = "DisGoBot"

	var me = new(discordgo.MessageEmbed)
	me.Footer = ef
	me.Type = "Rich"
	me.Color = 3447003
	return me
}

var configuration Configuration

func init() {

	// Get env var for disco token
	configuration.Token = os.Getenv("DISCO_TOKEN")
}

func main() {

	db, err = gorm.Open("sqlite3", "data/bot.db")
	if err != nil {
		log.Println("Database Error: ", err)
	}
	defer db.Close()
	db.AutoMigrate(&MessageStoreMessage{})
	db.AutoMigrate(&User{})
	db.AutoMigrate(&Nickname{})
	db.AutoMigrate(&ChannelProp{})

	dg, err := discordgo.New("Bot " + configuration.Token)
	if err != nil {
		log.Println("error creating Discord session,", err)
		return
	}
	// Close connection when server stops.
	defer dg.Close()

	// Register the func as a callback for events.
	dg.AddHandler(messageCreate)
	dg.AddHandler(guildMemberChunk)
	dg.AddHandler(ready)
	dg.AddHandler(guildCreate)
	dg.AddHandler(guildMemberUpdate)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		log.Println("error opening connection,", err)
		return
	}

	//server health check endpoint
	http.HandleFunc("/health", // or whatever url you wish
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "OK")
		})

	//run http server for health check
	go func() {
		log.Println("Serving on 5000")
		log.Fatal(http.ListenAndServe(":5000", nil))
	}()

	log.Println("Server startup finished!")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt, os.Kill)

	// Wait here until CTRL-C or other term signal is received.
	<-sc

	log.Println("Disco connection Closed.")
	log.Println("Server is stopping!")
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	chn, _ := s.Channel(m.ChannelID)

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	author, _ := s.User(m.Author.ID)
	if strings.HasPrefix(m.Content, "!") {
		cmds := strings.Split(m.Content, " ")
		switch cmds[0] {

		case "!set":
			perms, _ := s.State.UserChannelPermissions(m.Author.ID, chn.ID)
			if len(cmds) > 1 && perms&0x00000010 != 0 {
				var chp ChannelProp
				err := db.FirstOrCreate(&chp, ChannelProp{ChannelID: chn.ID}).Error
				if err != nil {
					log.Println(err)

				}
				switch cmds[1] {

				case "ticker":
					if len(cmds) > 2 {
						switch cmds[2] {
						case "on":

							err := db.Model(&chp).Updates(ChannelProp{Ticker: true}).Error
							if err != nil {
								log.Println(err)
							}
							s.ChannelMessageSend(chn.ID, "OK "+author.Mention()+", Ticker was set to ON.")
						case "off":
							err := db.Exec("UPDATE channel_props SET ticker = ? WHERE channel_id = ?", false, chn.ID).Error
							if err != nil {
								log.Println(err)
							}
							var params discordgo.ChannelEdit
							params.Topic = " "
							s.ChannelEditComplex(chn.ID, &params)
							s.ChannelMessageSend(chn.ID, "OK "+author.Mention()+", Ticker was set to OFF.")

						default:
							s.ChannelMessageSend(chn.ID, "ticker <on|off>")
						}
					}
				}
			} else {
				s.ChannelMessageSend(chn.ID, author.Mention()+" No.")
			}

		case "!choose":
			mg := NewMessageEmbed()
			mg.Title = "Choice Bot"
			var mef = new(discordgo.MessageEmbedField)
			//mg.Description = "Here is what was choosen"
			q := strings.Join(cmds[1:], " ")

			if strings.Contains(q, " or ") {
				result := strings.Split(q, " or ")

				mef.Name = string(result[rand.Intn(len(result))])
				mef.Value = "So it has been Written!"
				mef.Inline = false
				mg.Fields = append(mg.Fields[:], mef)
				s.ChannelMessageSendEmbed(m.ChannelID, mg)
			} else if strings.Contains(q, " OR ") {
				result := strings.Split(q, " OR ")

				mef.Name = string(result[rand.Intn(len(result))])
				mef.Value = "So it has been Written!"
				mef.Inline = false
				mg.Fields = append(mg.Fields[:], mef)
				s.ChannelMessageSendEmbed(m.ChannelID, mg)
			} else {
				var mef = new(discordgo.MessageEmbedField)
				mef.Name = "Error"
				mef.Value = "Optons must be split by ' OR '"
				mef.Inline = false
				mg.Fields = append(mg.Fields[:], mef)
				s.ChannelMessageSendEmbed(m.ChannelID, mg)
			}

		case "!crypt", "!btc", "!BTC", "!crypto":
			currents := []string{"BTC", "LTC", "ETH"}

			mg := NewMessageEmbed()
			mg.Title = "Crypto Price"
			mg.URL = "https://www.coinbase.com/charts"
			mg.Description = "Crypto Price Powered by CoinBase"

			for _, v := range currents[:] {
				var mef = new(discordgo.MessageEmbedField)
				mef.Name = v
				mef.Value = fmt.Sprintf("$%s", query.GetCrypt(v))
				mg.Fields = append(mg.Fields[:], mef)

			}
			s.ChannelMessageSendEmbed(m.ChannelID, mg)

		case "!seen":
			mg := NewMessageEmbed()
			var mef = new(discordgo.MessageEmbedField)
			var rs string
			if len(cmds) > 1 {
				if len(m.Mentions) > 0 {
					rs = m.Mentions[0].ID
				} else if cmds[1] != "" {
					var n Nickname
					db.Where("nickname = ?", cmds[1]).First(&n)
					rs = n.DiscordUserID
				}

				if rs == "" {
					mef.Name = "Not Seen"
					mef.Value = "Sorry **" + string(cmds[1]) + "** has never been seen here."
					mg.Fields = append(mg.Fields[:], mef)
				} else {
					ruser, _ := s.User(rs)
					var msm MessageStoreMessage
					if db.Where("discord_user_id = ? AND guild_id = ?", rs, chn.GuildID).First(&msm).RecordNotFound() {
						mef.Name = "Lurker"
						mef.Value = ruser.Username + " has just been watching from afar."
						mg.Fields = append(mg.Fields[:], mef)
					} else {

						db.Where("discord_user_id = ? AND guild_id = ?", rs, chn.GuildID).First(&msm)
						msg, _ := s.ChannelMessage(m.ChannelID, msm.MessageID)
						t1, _ := time.Parse("2006-01-02T15:04:05Z07:00", string(msg.Timestamp))
						//mef.Name = string(q + " @ " + t1.Format("Jan-02-2006 3:04PM"))
						dur := time.Since(t1)
						mef.Name = "Seen " + fmt.Sprintf("%s", dur-(dur%time.Second)) + " ago"
						mef.Value = "**" + string(ruser.Username) + ":**\n" + " \"" + msg.Content + "\""
						//aut.Name = ms[q].Author.Username
						//aut.IconURL = ms[q].Author.Avatar
						mg.Fields = append(mg.Fields[:], mef)
					}
				}
			} else {
				mef.Name = "Error"
				mef.Value = "Please provide a search"
				mg.Fields = append(mg.Fields[:], mef)
			}

			s.ChannelMessageSendEmbed(m.ChannelID, mg)

		case "!help":
			mg := NewMessageEmbed()
			mg.Title = "DisGoBot"
			//mg.URL = sq
			mg.Description = "Available Commands"

			var mef = new(discordgo.MessageEmbedField)
			mef.Name = "!choose <this> or <that>"
			mef.Value = "Chooses between options split by ' OR '"
			mef.Inline = false
			mg.Fields = append(mg.Fields[:], mef)

			mef = new(discordgo.MessageEmbedField)
			mef.Name = "!crypt | !crypto | !btc"
			mef.Value = "Responds with current BTC, ETH, and LTC prices."
			mef.Inline = false
			mg.Fields = append(mg.Fields[:], mef)

			mef = new(discordgo.MessageEmbedField)
			mef.Name = "!g <search terms>"
			mef.Value = "Responds with top three search results from DuckDuckGo."
			mef.Inline = false
			mg.Fields = append(mg.Fields[:], mef)

			mef = new(discordgo.MessageEmbedField)
			mef.Name = "!news <source>"
			mef.Value = "Responds with top three headlines from source. \n drudge | hacker"
			mef.Inline = false
			mg.Fields = append(mg.Fields[:], mef)

			mef = new(discordgo.MessageEmbedField)
			mef.Name = "!seen <User Name>"
			mef.Value = "Responds with time since user was last seen and their last comment."
			mef.Inline = false
			mg.Fields = append(mg.Fields[:], mef)

			s.ChannelMessageSendEmbed(m.ChannelID, mg)

		case "!g":

			q := strings.Join(cmds[1:], " ")
			r, sq := ddgquery.Query(q, 3)

			mg := NewMessageEmbed()
			mg.Title = "Search Results for: " + q
			mg.URL = sq
			mg.Description = "Search Powered by DuckDuckGo"

			for _, v := range r[:] {
				var mef = new(discordgo.MessageEmbedField)
				mef.Name = v.Title
				mef.Value = fmt.Sprintf("[%s](%s)", v.Info, v.Ref)
				mef.Inline = false
				mg.Fields = append(mg.Fields[:], mef)
			}

			s.ChannelMessageSendEmbed(m.ChannelID, mg)

		case "!news":
			if len(cmds) > 1 {

				switch cmds[1] {

				case "drudge":

					r := query.DrudgeQuery(3)

					mg := NewMessageEmbed()
					mg.Title = "Drudge Report"
					mg.URL = "http://www.drudgereport.com/"

					for _, v := range r[:] {
						var mef = new(discordgo.MessageEmbedField)
						mef.Name = v.Title
						//mef.Value = log.Sprintf("[%s](%s)", "link", v.Ref)
						mef.Value = v.Ref
						mef.Inline = false
						mg.Fields = append(mg.Fields[:], mef)
					}

					s.ChannelMessageSendEmbed(m.ChannelID, mg)

				case "hacker":
					var aresponse []int

					type Post struct {
						ID    int    `json:"id"`
						Title string `json:"title"`
						Ptype string `json:"type"`
						URL   string `json:"url"`
					}

					url := "https://hacker-news.firebaseio.com/v0/topstories.json"
					resp, err := http.Get(url)
					if err != nil {
						// handle error
					}
					defer resp.Body.Close()
					body, err := ioutil.ReadAll(resp.Body)

					json.Unmarshal(body, &aresponse)

					mg := NewMessageEmbed()
					mg.Title = "Hacker News"
					mg.URL = "https://news.ycombinator.com/"

					for i := 0; i < 3; i++ {
						url = "https://hacker-news.firebaseio.com/v0/item/" + strconv.Itoa(aresponse[i]) + ".json"

						resp, err = http.Get(url)
						if err != nil {
							log.Println(err)
						}
						var post Post
						body, err = ioutil.ReadAll(resp.Body)
						json.Unmarshal(body, &post)

						var mef = new(discordgo.MessageEmbedField)
						mef.Name = post.Title

						mef.Inline = false

						if post.Ptype != "story" || post.Ptype == "" {
							mef.Value = "https://news.ycombinator.com/item?id=" + strconv.Itoa(aresponse[i])
						} else {
							mef.Value = post.URL
						}
						mg.Fields = append(mg.Fields[:], mef)
					}
					s.ChannelMessageSendEmbed(m.ChannelID, mg)
				}
			}

		default:
			mg := NewMessageEmbed()
			mg.Title = "Det-Bot"
			//mg.URL = sq
			mg.Description = "Invalid Command"

			var mef = new(discordgo.MessageEmbedField)
			mef.Name = cmds[0] + " " + "is not a valid command!"
			mef.Value = "Please use !help to get a list of all avalible commands."
			mef.Inline = false
			mg.Fields = append(mg.Fields[:], mef)

			s.ChannelMessageSendEmbed(m.ChannelID, mg)
		}
	}

	var msm MessageStoreMessage
	db.FirstOrCreate(&msm, MessageStoreMessage{GuildID: chn.GuildID, DiscordUserID: m.Author.ID})
	db.Model(&msm).Updates(MessageStoreMessage{MessageID: m.ID})
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateGameStatus(0, "Det-Bot | !help")

	//run ticker update watcher
	go tickerUpdate(s)
	log.Println("Ticker update watcher started.")
}

func guildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	for _, g := range s.State.Guilds {
		err := s.RequestGuildMembers(g.ID, "", 0, false)
		if err != nil {
			log.Println(err)
		}
	}
}

func guildMemberUpdate(s *discordgo.Session, event *discordgo.GuildMemberUpdate) {
	for _, g := range s.State.Guilds {
		err := s.RequestGuildMembers(g.ID, "", 0, false)
		if err != nil {
			log.Println(err)
		}
	}
}

func guildMemberChunk(s *discordgo.Session, event *discordgo.GuildMembersChunk) {
	for _, m := range event.Members {
		var usr User
		err := db.FirstOrCreate(&usr, User{DiscordUserID: m.User.ID}).Error
		if err != nil {
			log.Println(err)
		}
		var newnick Nickname
		err = db.FirstOrCreate(&newnick, Nickname{Nickname: m.User.Username, GuildID: event.GuildID, DiscordUserID: m.User.ID}).Error
		if err != nil {
			log.Println(err)
		}
		if m.Nick != "" {
			var nick Nickname
			err = db.FirstOrCreate(&nick, Nickname{Nickname: m.Nick, GuildID: event.GuildID, DiscordUserID: m.User.ID}).Error
			if err != nil {
				log.Println(err)
			}
		} else {

		}
	}
}
func tickerUpdate(s *discordgo.Session) {
	var params discordgo.ChannelEdit
	for {
		var chanprops []ChannelProp
		db.Where("ticker = ?", true).Find(&chanprops)
		for _, chp := range chanprops {
			params.Topic = fmt.Sprintf("BTC $%s | LTC $%s | ETH $%s", query.GetCrypt("BTC"), query.GetCrypt("LTC"), query.GetCrypt("ETH"))
			s.ChannelEditComplex(chp.ChannelID, &params)
		}
		time.Sleep(5 * time.Second)
	}
}
