package commands

import (
	"fmt"
	"github.com/alfredxing/calc/compute"
	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dutil/commandsystem"
	"github.com/jonas747/yagpdb/bot"
	"github.com/jonas747/yagpdb/common"
	"github.com/lunixbochs/vtclean"
	"image"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// var (
// 	EscapeSquenceRegex = regexp.MustCompile(`(\x1b\[|\x9b)[^@-_]*[@-_]|\x1b[@-_]`)
// )

func (p *Plugin) InitBot() {
	bot.CommandSystem.Prefix = p
	bot.CommandSystem.RegisterCommands(GlobalCommands...)
}

func (p *Plugin) GetPrefix(s *discordgo.Session, m *discordgo.MessageCreate) string {
	client, err := bot.RedisPool.Get()
	if err != nil {
		log.Println("Failed redis connection from pool", err)
		return ""
	}
	defer bot.RedisPool.Put(client)

	channel, err := s.State.Channel(m.ChannelID)
	if err != nil {
		log.Println("Failed retrieving channel from state", err)
		return ""
	}

	config := GetConfig(client, channel.GuildID)
	return config.Prefix
}

var GlobalCommands = []commandsystem.CommandHandler{
	&commandsystem.SimpleCommand{
		Name:        "Help",
		Description: "Shows help abut all or one specific command",
		RunInDm:     true,
		Arguments: []*commandsystem.ArgumentDef{
			&commandsystem.ArgumentDef{Name: "command", Type: commandsystem.ArgumentTypeString},
		},
		RunFunc: func(parsed *commandsystem.ParsedCommand, source commandsystem.CommandSource, m *discordgo.MessageCreate) error {
			target := ""
			if parsed.Args[0] != nil {
				target = parsed.Args[0].Str()
			}
			help := bot.CommandSystem.GenerateHelp(target, 0)
			bot.Session.ChannelMessageSend(m.ChannelID, help)
			return nil
		},
	},
	// Status command shows the bot's status, stuff like version, conntected servers, uptime, memory used etc..
	&commandsystem.SimpleCommand{
		Name:        "Status",
		Description: "Shows yagpdb status",
		RunInDm:     true,
		RunFunc: func(cmd *commandsystem.ParsedCommand, source commandsystem.CommandSource, m *discordgo.MessageCreate) error {
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)
			servers := len(bot.Session.State.Guilds)

			uptime := time.Since(bot.Started)

			// Convert to megabytes for ez readings
			allocated := float64(memStats.Alloc) / 1000000
			totalAllocated := float64(memStats.TotalAlloc) / 1000000

			numGoroutines := runtime.NumGoroutine()

			status := fmt.Sprintf("**YAGPDB STATUS** *bot version: %s*\n - Connected Servers: %d\n - Uptime: %s\n - Allocated: %.2fMB\n - Total Allocated: %.2fMB\n - Number of Goroutines: %d\n",
				bot.VERSION, servers, uptime.String(), allocated, totalAllocated, numGoroutines)

			bot.Session.ChannelMessageSend(m.ChannelID, status)

			return nil
		},
	},
	// Some fun commands because why not
	&commandsystem.SimpleCommand{
		Name:         "Reverse",
		Aliases:      []string{"r", "rev"},
		Description:  "Flips stuff",
		RunInDm:      true,
		RequiredArgs: 1,
		Arguments: []*commandsystem.ArgumentDef{
			&commandsystem.ArgumentDef{Name: "What", Description: "To flip", Type: commandsystem.ArgumentTypeString},
		},
		RunFunc: func(cmd *commandsystem.ParsedCommand, source commandsystem.CommandSource, m *discordgo.MessageCreate) error {
			toFlip := cmd.Args[0].Str()

			out := ""
			for _, r := range toFlip {
				out = string(r) + out
			}
			bot.Session.ChannelMessageSend(m.ChannelID, "Flippa: "+out)

			return nil
		},
	},
	&commandsystem.SimpleCommand{
		Name:         "Weather",
		Aliases:      []string{"w"},
		Description:  "Shows the weather somewhere",
		RunInDm:      true,
		RequiredArgs: 1,
		Arguments: []*commandsystem.ArgumentDef{
			&commandsystem.ArgumentDef{Name: "Where", Description: "Where", Type: commandsystem.ArgumentTypeString},
		},
		RunFunc: func(cmd *commandsystem.ParsedCommand, source commandsystem.CommandSource, m *discordgo.MessageCreate) error {
			where := cmd.Args[0].Str()

			req, err := http.NewRequest("GET", "http://wttr.in/"+where, nil)
			if err != nil {
				return err
			}

			req.Header.Set("User-Agent", "curl/7.49.1")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			// remove escape sequences
			unescaped := vtclean.Clean(string(body), false)

			split := strings.Split(string(unescaped), "\n")

			out := "```\n"
			for i := 0; i < 7; i++ {
				if i >= len(split) {
					break
				}
				out += strings.TrimRight(split[i], " ") + "\n"
			}
			out += "\n```"

			_, err = bot.Session.ChannelMessageSend(m.ChannelID, out)
			return err
		},
	},
	&commandsystem.SimpleCommand{
		Name:        "Invite",
		Aliases:     []string{"inv", "i"},
		Description: "Responds with bto invite link",
		RunInDm:     true,
		RunFunc: func(cmd *commandsystem.ParsedCommand, source commandsystem.CommandSource, m *discordgo.MessageCreate) error {
			clientId := bot.Config.ClientID
			link := fmt.Sprintf("https://discordapp.com/oauth2/authorize?client_id=%s&scope=bot&permissions=535948311&response_type=code&redirect_uri=http://yagpdb.xyz/cp/", clientId)
			_, err := bot.Session.ChannelMessageSend(m.ChannelID, "You manage this bot through the control panel interface but heres an invite link incase you just want that\n"+link)
			return err
		},
	},
	&commandsystem.SimpleCommand{
		Name:         "Ascii",
		Aliases:      []string{"asci"},
		Description:  "Converts an image to ascii",
		RunInDm:      true,
		RequiredArgs: 1,
		Arguments: []*commandsystem.ArgumentDef{
			&commandsystem.ArgumentDef{Name: "Where", Description: "Where", Type: commandsystem.ArgumentTypeString},
		},
		RunFunc: func(cmd *commandsystem.ParsedCommand, source commandsystem.CommandSource, m *discordgo.MessageCreate) error {

			resp, err := http.Get(cmd.Args[0].Str())
			if err != nil {
				return err
			}

			img, _, err := image.Decode(resp.Body)
			resp.Body.Close()
			if err != nil {
				return err
			}

			out := Convert2Ascii(ScaleImage(img, 50))
			_, err = bot.Session.ChannelMessageSend(m.ChannelID, "```\n"+string(out)+"\n```")
			return err
		},
	},
	&commandsystem.SimpleCommand{
		Name:         "Calc",
		Aliases:      []string{"c", "calculate"},
		Description:  "Converts an image to ascii",
		RunInDm:      true,
		RequiredArgs: 1,
		Arguments: []*commandsystem.ArgumentDef{
			&commandsystem.ArgumentDef{Name: "What", Description: "What to calculate", Type: commandsystem.ArgumentTypeString},
		},
		RunFunc: func(cmd *commandsystem.ParsedCommand, source commandsystem.CommandSource, m *discordgo.MessageCreate) error {
			result, err := compute.Evaluate(cmd.Args[0].Str())
			if err != nil {
				return err
			}

			bot.Session.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Result: `%f`", result))
			return nil
		},
	},
	&commandsystem.SimpleCommand{
		Name:        "Pastebin",
		Aliases:     []string{"ps", "paste"},
		Description: "Creates a pastebin of the channels last 100 messages",
		RunFunc: func(cmd *commandsystem.ParsedCommand, source commandsystem.CommandSource, m *discordgo.MessageCreate) error {
			id, err := common.CreatePastebinLog(m.ChannelID)
			if err != nil {
				return err
			}
			common.BotSession.ChannelMessageSend(m.ChannelID, fmt.Sprintf("<http://pastebin.com/%s>", id))
			return nil
		},
	},
}