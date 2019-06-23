package main

import (
    "os"
    "fmt"
    "github.com/bwmarrin/discordgo"
    "github.com/benkillin/ConfigHelper"
    log "github.com/sirupsen/logrus"
    "time"
    "strings"
)

var (
    configFile = "factionsBotConfig.json"
    defaultConfigFile = "factionsBotConfig.default.json" // this file gets overwritten every run with the current default config
    botID string // Bot ID
    config *Config
)

// Config represents the application's configuration
type Config struct {
    Token string
    CommandPrefix string
    Logging LoggingConfig
    Guilds map[string]GuildConfig
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
    GuildName string
    WallsEnabled bool
    WallsCheckTimeout time.Duration
    WallsCheckReminder time.Duration
    WallsCheckChannelID string
    WallsRoleMention string
    Players map[string]PlayerConfig
}

// PlayerConfig represents the players and their scores.
type PlayerConfig struct {
    PlayerString string
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
            Logfile: ""},
            Guilds: map[string]GuildConfig{
                "123456789012345678": GuildConfig{
                    GuildName: "DerpGuild",
                    WallsEnabled: false,
                    WallsCheckTimeout: 15*time.Minute,
                    WallsCheckReminder: 5*time.Minute,
                    WallsCheckChannelID: "#123456789012345678",
                    WallsRoleMention: "@&123456789012345678",
                    Players: map[string]PlayerConfig{
                        "123456789012345678": PlayerConfig{
                            PlayerString: "Derp#1234",
                            PlayerMention: "@123456789012345678",
                            WallChecks: 0,
                            LastWallCheck: time.Time{}}}}}} // the default config
    config = &Config{} // the running configuration

    // This is debug code basically to keep the default json file updated which is checked into git.
    os.Remove(defaultConfigFile)
    ConfigHelper.GetConfigWithDefault(defaultConfigFile, defaultConfig, &Config{})
    
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

    d.AddHandler(messageHandler)

    err = d.Open()

    if err != nil {
        log.Fatalf("Error: unable to establish connection to discord: %s", err)
    }

    defer d.Close()

    <-make(chan struct{})
}

// our command handler function
func messageHandler(d *discordgo.Session, msg *discordgo.MessageCreate) {
    user := msg.Author
    if user.ID == botID || user.Bot {
        return
    }

    content := msg.Content

    splitContent := strings.Split(content, " ")

    switch splitContent[0]{
    case config.CommandPrefix + "test":
        testCmd(d, msg, msg.ChannelID)
    case config.CommandPrefix + "set":
        setCmd(d ,msg, msg.ChannelID)
    case config.CommandPrefix + "clear":
        clearCmd(d, msg, msg.ChannelID)
    case config.CommandPrefix + "weewoo":
        weewooCmd(d, msg, msg.ChannelID)
    case config.CommandPrefix + "help":
        helpCmd(d, msg, msg.ChannelID)
    }
}

func testCmd(d *discordgo.Session, msg *discordgo.MessageCreate, channelID string) {
    log.Debugf("Incoming TEST Message: %+v\n", msg.Message)
    messageIds := make([]string, 0)
    log.Debugf("Mention of author: %s; String of author: %s; author ID: %s", msg.Author.Mention(), msg.Author.String(), msg.Author.ID)

    deleteMsg(d, msg.ChannelID, msg.ID)
    
    msgID := sendMsg(d, msg.ChannelID, fmt.Sprintf("Hello, %s, you have initated a test of the self destruct sequence!", msg.Author.Mention()))
    messageIds = append(messageIds, msgID)

    for i := 5; i > 0; i-- {
        msgID := sendMsg(d, msg.ChannelID, fmt.Sprintf("%d", i))
        messageIds = append(messageIds, msgID)
        time.Sleep(1500 * time.Millisecond) // it seems if it is 1 second or faster then discord itself will throttle.
    }

    time.Sleep(3 * time.Second)

    err := d.ChannelMessagesBulkDelete(msg.ChannelID, messageIds)
    if err != nil {
        log.Errorf("Error: Unable to delete messages: %s", err)
    }
}

func setCmd(d *discordgo.Session, msg *discordgo.MessageCreate, channelID string) {
    sendTempMsg(d, channelID, "Settings command handler! TODO: this handler!", 5*time.Second)
}

func helpCmd(d *discordgo.Session, msg *discordgo.MessageCreate, channelID string) {
    sendTempMsg(d, channelID, "Help command handler! TODO: this handler!", 5*time.Second)
}

func clearCmd(d *discordgo.Session, msg *discordgo.MessageCreate, channelID string) {
    sendTempMsg(d, channelID, "Clear command handler! TODO: this handler!", 5*time.Second)
}

func weewooCmd(d *discordgo.Session, msg *discordgo.MessageCreate, channelID string) {
    sendTempMsg(d, channelID, "Weewoo command handler! TODO: this handler!", 5*time.Second)
}

/*func Cmd(d *discordgo.Session, msg *discordgo.MessageCreate, channelID string) {
}*/

func hello() (string) {
	return "Hello, world!"
}

func sendMsg(d *discordgo.Session, channelID string, msg string) (string) {
    err := d.ChannelTyping(channelID)
    if err != nil {
        log.Errorf("Unable to send typing notification: %s", err)
    }

    sentMessage, err := d.ChannelMessageSend(channelID, msg)
    if err != nil {
        log.Errorf("Unable to send message: %s", err)
        return ""
    }

    return sentMessage.ID
}

func deleteMsg(d *discordgo.Session, channelID string, messageID string) (error) {
    err := d.ChannelMessageDelete(channelID, messageID)
    if err != nil {
        log.Errorf("Error: Unable to delete incoming message: %s", err)
        return err
    }

    return nil
}

func sendTempMsg(d *discordgo.Session, channelID string, msg string, timeout time.Duration) {
    go func() {
        messageID := sendMsg(d, channelID, msg)
        time.Sleep(timeout)
        d.ChannelMessageDelete(channelID, messageID)
    }()
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