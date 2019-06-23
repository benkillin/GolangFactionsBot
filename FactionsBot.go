package main

import (
    "os"
    "fmt"
    "github.com/bwmarrin/discordgo"
    "github.com/benkillin/ConfigHelper"
    log "github.com/sirupsen/logrus"
    "time"
)

var (
    configFile = "factionsBotConfig.json"
    botID string // Bot ID
    config *Config
)

// Config represents the application's configuration
type Config struct {
    Token string
    CommandPrefix string
    Logging LoggingConfig
    Guilds []GuildConfig
}

// LoggingConfig configuration as part of the config object.
type LoggingConfig struct {
    Level string
    Format string
    Output string
    Logfile string
}

// GuildConfig represents the configuration of a single instance of this bot on a particular server/guild
type GuildConfig struct {
    GuildID string
    WallsEnabled bool
    WallsCheckTimeout time.Duration
    WallsCheckReminder time.Duration
    WallsCheckChannelID string
    WallsRoleMention string
    Players []PlayerConfig
}

// PlayerConfig represents the players and their scores.
type PlayerConfig struct {
    PlayerID string
    PlayerMention string
    WallChecks int
    LastWallCheck time.Time
}

// our main function
func main() {
    defaultConfig := &Config{
        Token: "",
        CommandPrefix: ".",
        Logging: LoggingConfig {
            Level: "trace",
            Format: "text",
            Output: "stderr",
            Logfile: ""}} // the default config
    config = &Config{} // the running configuration

    err := ConfigHelper.GetConfigWithDefault(configFile, defaultConfig, config)
	if err != nil {
		log.Fatalf("error loading/saving config/default config. %s", err)
    }
    
    setupLogging(config)

    token := config.Token
    
	d, err := discordgo.New("Bot " + token)
    if err != nil {
        log.Fatalf("Failed to create discord session: %s", err)
    }

    bot, err := d.User("@me")
    if err != nil {
        log.Fatalf("Failed to get the bot user/access account: %s", err)
    }

	botID = bot.ID

    d.AddHandler(testCmd)
    d.AddHandler(setCmd)
    d.AddHandler(helpCmd)
    d.AddHandler(clearCmd)
    d.AddHandler(weewooCmd)

    err = d.Open()

    if err != nil {
        log.Fatalf("Error: unable to establish connection to discord: %s", err)
    }

    defer d.Close()

    <-make(chan struct{})
}

// our command handler function
func testCmd(d *discordgo.Session, msg *discordgo.MessageCreate) {
    user := msg.Author
    if user.ID == botID || user.Bot {
        return
    }
    
    log.Debugf("Incoming Message: %+v\n", msg.Message)

    messageIds := make([]string, 0)
    content := msg.Content

    if (content == config.CommandPrefix + "test") {
        d.ChannelTyping(msg.ChannelID)

        err := d.ChannelMessageDelete(msg.ChannelID, msg.ID)
        if err != nil {
            log.Errorf("Error: Unable to delete incoming message: %s", err)
            d.ChannelMessageSend(msg.ChannelID, fmt.Sprintf("Bot error: Unable to delete message: %s", err))
        }

        msgID := sendMsg(d, msg.ChannelID, fmt.Sprintf("Hello, %s, you have initated a test of the self destruct sequence!", msg.Author.Mention()))
        messageIds = append(messageIds, msgID)

        for i := 5; i > 0; i-- {
            time.Sleep(1 * time.Second)
            msgID := sendMsg(d, msg.ChannelID, fmt.Sprintf("%d", i))
            messageIds = append(messageIds, msgID)
        }

        log.Printf("Message IDs to delete: %#v", messageIds)

        err = d.ChannelMessagesBulkDelete(msg.ChannelID, messageIds)
        if err != nil {
            log.Errorf("Error: Unable to delete messages: %s", err)
        }
    }
}

func sendMsg(d *discordgo.Session, channelID string, msg string) (string) {
    sentMessage, err := d.ChannelMessageSend(channelID, msg)
    if err != nil {
        log.Errorf("Unable to send message: %s", err)
        return ""
    }
    return sentMessage.ID
}

func setCmd(d *discordgo.Session, msg *discordgo.MessageCreate) {
}

func helpCmd(d *discordgo.Session, msg *discordgo.MessageCreate) {
}

func clearCmd(d *discordgo.Session, msg *discordgo.MessageCreate) {
}

func weewooCmd(d *discordgo.Session, msg *discordgo.MessageCreate) {
}

/*func Cmd(d *discordgo.Session, msg *discordgo.MessageCreate) {
}*/

func hello() (string) {
	return "Hello, world!"
}

func setupLogging(config *Config) {

    if config.Logging.Format == "text" {
        log.SetFormatter(&log.TextFormatter{})
    } else if config.Logging.Format == "json" {
        log.SetFormatter(&log.JSONFormatter{})
    } else {
        log.Warning("Warning: unknown logging format specified. Allowed options are 'text' and 'json' for config.Logging.Format")
        log.SetFormatter(&log.TextFormatter{})
    }
	
    level, err := log.ParseLevel(config.Logging.Level)
    if err != nil {
        log.Fatalf("Error setting up logging - invalid parse level: %s", err)
    }

    log.SetLevel(level)

    if config.Logging.Output == "file" {
        file, err := os.OpenFile(config.Logging.Logfile, os.O_RDWR, 0644)
        if err != nil {
            log.Fatalf("Error opening log file: %s", err)
        }

        log.SetOutput(file)
    } else if config.Logging.Output == "stdout" {
        log.SetOutput(os.Stdout) // bydefault the package outputs to stderr
    } else if config.Logging.Output == "stderr" {
        // do nothing
    } else {
        log.Warn("Warning: log output option not recognized. Valid options are 'file' 'stdout' 'stderr' for config.Logging.output")
    }
}