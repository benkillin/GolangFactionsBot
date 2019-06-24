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
    TimerLoopTimeout time.Duration
    Logging LoggingConfig
    Guilds map[string]*GuildConfig
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
    WallsLastChecked time.Time
    WallsCheckChannelID string
    WallsRoleMention string
    WallReminders int
    ReminderMessages []string
    LastReminder time.Time
    Players map[string]*PlayerConfig
}

// PlayerConfig represents the players and their scores.
type PlayerConfig struct {
    PlayerString string
    PlayerUsername string
    PlayerMention string
    WallChecks int
    LastWallCheck time.Time
}

// our main function
func main() {
    defaultConfig := &Config{
        Token: "",
        CommandPrefix: ".",
        TimerLoopTimeout: 5 * time.Second,
        Logging: LoggingConfig {
            Level: "trace",
            Format: "text",
            Output: "stderr",
            Logfile: ""},
        Guilds: map[string]*GuildConfig{
            "123456789012345678": &GuildConfig{
                GuildName: "DerpGuild",
                WallsEnabled: false,
                WallsCheckTimeout: 45*time.Minute,
                WallsCheckReminder: 30*time.Minute,
                WallsCheckChannelID: "#123456789012345678",
                WallsRoleMention: "@&123456789012345678",
                Players: map[string]*PlayerConfig{
                    "123456789012345678": &PlayerConfig{
                        PlayerString: "Derp#1234",
                        PlayerUsername: "asdfasdfasdf",
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
    log.Infof("Created discord object.")

    bot, err := d.User("@me")
    if err != nil {
        log.Fatalf("Failed to get the bot user/access account: %s", err)
    }
    log.Infof("Obtained self user.")

	botID = bot.ID
    d.AddHandler(messageHandler)

    err = d.Open()
    if err != nil {
        log.Fatalf("Error: unable to establish connection to discord: %s", err)
    }
    log.Infof("Successfully opened discord connection.")

    defer d.Close()

    // goroutine for looping through guilds and checking last checked time
    go func() {
        for {
            for guildID := range config.Guilds {
                if config.Guilds[guildID].WallsEnabled {
                    lastCheckedPlusTimeout := config.Guilds[guildID].WallsLastChecked.Add(config.Guilds[guildID].WallsCheckTimeout)
                    if time.Now().After(lastCheckedPlusTimeout) {
                        if config.Guilds[guildID].WallReminders == 0 {
                            config.Guilds[guildID].WallReminders = 1

                            reminderID := sendMsg(d, config.Guilds[guildID].WallsCheckChannelID, 
                                fmt.Sprintf("It's time to check walls! Time last checked %s", config.Guilds[guildID].WallsLastChecked))
                            config.Guilds[guildID].ReminderMessages = append(config.Guilds[guildID].ReminderMessages, reminderID)
                            config.Guilds[guildID].LastReminder = time.Now()
                        } else {
                            lastReminderPlusReminderInterval := config.Guilds[guildID].LastReminder.Add(config.Guilds[guildID].WallsCheckReminder)

                            if time.Now().After(lastReminderPlusReminderInterval) {
                                config.Guilds[guildID].WallReminders++
                                durationSinceLastChecked := time.Now().Sub(config.Guilds[guildID].WallsLastChecked)
                                msg := fmt.Sprintf("<@&%s>, reminder to check walls! They have still not been checked! It has been %s since the last check!", 
                                    config.Guilds[guildID].WallsRoleMention, durationSinceLastChecked.Round(time.Second))
                                reminderID := sendMsg(d, config.Guilds[guildID].WallsCheckChannelID, msg)
                                clearReminderMessages(d, guildID)
                                config.Guilds[guildID].ReminderMessages = append(config.Guilds[guildID].ReminderMessages, reminderID)
                                config.Guilds[guildID].WallsCheckReminder++
                                config.Guilds[guildID].LastReminder = time.Now()
                            }
                        }

                        ConfigHelper.SaveConfig(configFile, config)
                    }
                }
            }

            time.Sleep(config.TimerLoopTimeout)
        }
    }()

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
        testCmd(d, msg.ChannelID, msg, splitContent)
    case config.CommandPrefix + "set":
        setCmd(d, msg.ChannelID, msg, splitContent)
    case config.CommandPrefix + "clear":
        clearCmd(d, msg.ChannelID, msg, splitContent)
    case config.CommandPrefix + "weewoo":
        weewooCmd(d, msg.ChannelID, msg, splitContent)
    case config.CommandPrefix + "help":
        helpCmd(d, msg.ChannelID, msg, splitContent)
    }
}

func setCmd(d *discordgo.Session, channelID string, msg *discordgo.MessageCreate, splitMessage []string) {
    deleteMsg(d, msg.ChannelID, msg.ID)

    if len(splitMessage) > 1 {
        log.Debugf("Incoming settings message: %+v", msg.Message)

        checkGuild(d, channelID, msg.GuildID)

        subcommand := splitMessage[1]

        switch subcommand {
        case "walls":
            if len(splitMessage) > 2 {
                changed := false

                switch splitMessage[2] {
                case "on":
                    config.Guilds[msg.GuildID].WallsEnabled = true
                    changed = true
                    sendTempMsg(d, channelID, fmt.Sprintf("Wall checks are now enabled!"), 5 * time.Second)

                case "off":
                    config.Guilds[msg.GuildID].WallsEnabled = false
                    changed = true

                    sendTempMsg(d, channelID, fmt.Sprintf("Wall checks are now disabled."), 5 * time.Second)

                case "channel":
                    if len(splitMessage) > 3 {
                        wallsChannel := splitMessage[3]
                        wallsChannelID := strings.Replace(wallsChannel, "<", "", -1)
                        wallsChannelID = strings.Replace(wallsChannelID, ">", "", -1)
                        wallsChannelID = strings.Replace(wallsChannelID, "#", "", -1)

                        _, err := d.Channel(wallsChannelID)
                        if err != nil {
                            log.Errorf("Invalid channel specified while setting wall checks channel: %s", err)
                            sendTempMsg(d, channelID, fmt.Sprintf("Invalid channel specified: %s", err), 10*time.Second)
                        } else {
                            config.Guilds[msg.GuildID].WallsCheckChannelID = wallsChannelID
                            sendTempMsg(d, channelID, fmt.Sprintf("Set channel to send reminders to <#%s>", wallsChannelID), 5*time.Second)
                            changed = true
                        }
                    } else {
                        sendTempMsg(d, channelID, "usage: " + config.CommandPrefix + "set walls channel #channelNameForWallChecks", 10*time.Second)
                    }

                case "role":
                    if len(splitMessage) > 3 {
                        if len(msg.MentionRoles) > 0 {
                            mentionRole := msg.MentionRoles[0]
                            config.Guilds[msg.GuildID].WallsRoleMention = mentionRole
                            changed = true
                        } else {
                            sendTempMsg(d, channelID, "Error - invalid/no role specified", 10*time.Second)
                        }
                    } else { 
                        sendTempMsg(d, channelID, "usage: " + config.CommandPrefix + "set walls role @roleForWallCheckRemidners", 10*time.Second)
                    }

                case "timeout":
                    changed = true

                case "reminder": 
                    changed = true

                default:
                    sendCurrentWallsSettings(d, channelID, msg)
                }

                if changed {
                    ConfigHelper.SaveConfig(configFile, config)
                }
            } else {
                sendCurrentWallsSettings(d, channelID, msg)
            }
        }
    } else {
        helpCmd(d, channelID, msg, splitMessage)
    }
}

func helpCmd(d *discordgo.Session, channelID string, msg *discordgo.MessageCreate, splitMessage []string) {
    sendTempMsg(d, channelID, "Help command handler! TODO: this handler!", 5*time.Second)
}

func clearCmd(d *discordgo.Session, channelID string, msg *discordgo.MessageCreate, splitMessage []string) {
    deleteMsg(d, msg.ChannelID, msg.ID)
    log.Debugf("Incoming clear message: %+v", msg.Message)
    checkGuild(d, channelID, msg.GuildID)
    checkPlayer(d, channelID, msg.GuildID, msg.Author.ID)
    
    config.Guilds[msg.GuildID].WallsLastChecked = time.Now()
    config.Guilds[msg.GuildID].WallReminders = 0
    config.Guilds[msg.GuildID].Players[msg.Author.ID].WallChecks++

    timeTookSinceLastWallCheck := time.Now().Sub(config.Guilds[msg.GuildID].Players[msg.Author.ID].LastWallCheck)
    config.Guilds[msg.GuildID].Players[msg.Author.ID].LastWallCheck = time.Now()

    sendMsg(d, config.Guilds[msg.GuildID].WallsCheckChannelID, 
        fmt.Sprintf("Thanks, %s, the walls have been marked clear! Your current score is %d. Time took: %s.",
            config.Guilds[msg.GuildID].Players[msg.Author.ID].PlayerMention,
            config.Guilds[msg.GuildID].Players[msg.Author.ID].WallChecks,
            timeTookSinceLastWallCheck.Round(time.Second)))

    go func() {
        clearReminderMessages(d, msg.GuildID)
    } ()

    ConfigHelper.SaveConfig(configFile, config)
}

func weewooCmd(d *discordgo.Session, channelID string, msg *discordgo.MessageCreate, splitMessage []string) {
    deleteMsg(d, msg.ChannelID, msg.ID)
    log.Debugf("Incoming clear message: %+v", msg.Message)
    checkGuild(d, channelID, msg.GuildID)

    sendMsg(d, config.Guilds[msg.GuildID].WallsCheckChannelID,
        fmt.Sprintf("WEEWOO!!! WEEWOO!!!! WE ARE BEING RAIDED!!!! PLEASE GET ON AND HELP DEFEND THE BASE!!!"))
    
    go func() {
        for i := 0; i < 3; i++ {
            sendTempMsg(d, config.Guilds[msg.GuildID].WallsCheckChannelID,
                fmt.Sprintf("<@&%s> WE ARE BEING RAIDED!", config.Guilds[msg.GuildID].WallsRoleMention),
                30 * time.Second)
            time.Sleep(500 * time.Millisecond)
        }
    }()
}

func testCmd(d *discordgo.Session, channelID string, msg *discordgo.MessageCreate, splitMessage []string) {
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

/*func Cmd(d *discordgo.Session, msg *discordgo.MessageCreate, channelID string) {
}*/

func checkGuild(d *discordgo.Session, channelID string, GuildID string) {
    guild, err := d.Guild(GuildID)
    if err != nil {
        log.Errorf("Error obtaining guild: %s", err)
        sendMsg(d, channelID, fmt.Sprintf("Error obtaining guild: %s", err))
        return
    }

    if _, ok := config.Guilds[GuildID]; !ok {
        players := make(map[string]*PlayerConfig)
        config.Guilds[GuildID] = &GuildConfig{
            GuildName: guild.Name,
            WallsCheckChannelID: "",
            WallsCheckReminder: 30*time.Minute,
            WallsCheckTimeout: 45*time.Minute,
            WallsEnabled: false,
            WallsRoleMention: "",
            WallReminders: 0,
            Players: players}
    } else {
        if guild.Name != config.Guilds[GuildID].GuildName {
            config.Guilds[GuildID].GuildName = guild.Name
        } 
    }

    ConfigHelper.SaveConfig(configFile, config)
}

func checkPlayer(d *discordgo.Session, channelID string, GuildID string, authorID string) {
    checkGuild(d, channelID, GuildID)
    player, err := d.User(authorID)
    if err != nil {
        log.Errorf("Error obtaining user information: %s", err)
        sendMsg(d, channelID, fmt.Sprintf("Error obtaining user information: %s", err))
        return
    }

    if _, ok := config.Guilds[GuildID].Players[player.ID]; !ok {
        config.Guilds[GuildID].Players[player.ID] = &PlayerConfig {
            PlayerString: player.String(),
            PlayerUsername: player.Username,
            PlayerMention: player.Mention(),
            WallChecks: 0,
            LastWallCheck: time.Time{}}
    } else {
        if player.Username != config.Guilds[GuildID].Players[authorID].PlayerString {
            config.Guilds[GuildID].Players[authorID].PlayerString = player.String()
            config.Guilds[GuildID].Players[authorID].PlayerUsername = player.Username
            config.Guilds[GuildID].Players[authorID].PlayerMention = player.Mention()
        }
    }
}

func sendCurrentWallsSettings(d *discordgo.Session, channelID string, msg *discordgo.MessageCreate) {
    embed := NewEmbed().
        SetTitle("Walls settings").
        SetDescription("Current walls settings").
        AddField("Guild Name", config.Guilds[msg.GuildID].GuildName).
        AddField("Checks enabled", fmt.Sprintf("%t", config.Guilds[msg.GuildID].WallsEnabled)).
        AddField("Role to mention", "<@&" + config.Guilds[msg.GuildID].WallsRoleMention + ">").
        AddField("Check channel", "<#" + config.Guilds[msg.GuildID].WallsCheckChannelID + ">").
        AddField("Walls check reminder", fmt.Sprintf("%s", config.Guilds[msg.GuildID].WallsCheckReminder)).
        AddField("Walls check interval", fmt.Sprintf("%s", config.Guilds[msg.GuildID].WallsCheckTimeout)).
        AddField("Walls last checked", fmt.Sprintf("%s", config.Guilds[msg.GuildID].WallsLastChecked)).
        MessageEmbed
    _, err := d.ChannelMessageSendEmbed(channelID, embed)

    if err != nil {
        log.Errorf("Error sending message1: %s", err)
    }
}

func clearReminderMessages(d *discordgo.Session, GuildID string) {
    for i := 0; i < len(config.Guilds[GuildID].ReminderMessages); i++ {
        messageID := config.Guilds[GuildID].ReminderMessages[i]
        deleteMsg(d, config.Guilds[GuildID].WallsCheckChannelID, messageID)
        time.Sleep(1500 * time.Millisecond)
    }
    config.Guilds[GuildID].ReminderMessages = config.Guilds[GuildID].ReminderMessages[:0]
    ConfigHelper.SaveConfig(configFile, config)
}

func hello() (string) {
	return "Hello, world!"
}

// send a message including a typing notification.
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

// delete a message
func deleteMsg(d *discordgo.Session, channelID string, messageID string) (error) {
    err := d.ChannelMessageDelete(channelID, messageID)
    if err != nil {
        log.Errorf("Error: Unable to delete incoming message: %s", err)
        return err
    }

    return nil
}

// send a self deleting message "this message will self destruct in 5..." :)
func sendTempMsg(d *discordgo.Session, channelID string, msg string, timeout time.Duration) {
    go func() {
        messageID := sendMsg(d, channelID, msg)
        time.Sleep(timeout)
        d.ChannelMessageDelete(channelID, messageID)
    }()
}

// set up the logger.
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

func remove(s []string, i int) []string {
    s[len(s)-1], s[i] = s[i], s[len(s)-1]
    return s[:len(s)-1]
}























// from https://gist.github.com/Necroforger/8b0b70b1a69fa7828b8ad6387ebb3835

//Embed ...
type Embed struct {
	*discordgo.MessageEmbed
}

// Constants for message embed character limits
const (
	EmbedLimitTitle       = 256
	EmbedLimitDescription = 2048
	EmbedLimitFieldValue  = 1024
	EmbedLimitFieldName   = 256
	EmbedLimitField       = 25
	EmbedLimitFooter      = 2048
	EmbedLimit            = 4000
)

//NewEmbed returns a new embed object
func NewEmbed() *Embed {
	return &Embed{&discordgo.MessageEmbed{}}
}

//SetTitle ...
func (e *Embed) SetTitle(name string) *Embed {
	e.Title = name
	return e
}

//SetDescription [desc]
func (e *Embed) SetDescription(description string) *Embed {
	if len(description) > 2048 {
		description = description[:2048]
	}
	e.Description = description
	return e
}

//AddField [name] [value]
func (e *Embed) AddField(name, value string) *Embed {
	if len(value) > 1024 {
		value = value[:1024]
	}

	if len(name) > 1024 {
		name = name[:1024]
	}

	e.Fields = append(e.Fields, &discordgo.MessageEmbedField{
		Name:  name,
		Value: value,
	})

	return e

}

//SetFooter [Text] [iconURL]
func (e *Embed) SetFooter(args ...string) *Embed {
	iconURL := ""
	text := ""
	proxyURL := ""

	switch {
	case len(args) > 2:
		proxyURL = args[2]
		fallthrough
	case len(args) > 1:
		iconURL = args[1]
		fallthrough
	case len(args) > 0:
		text = args[0]
	case len(args) == 0:
		return e
	}

	e.Footer = &discordgo.MessageEmbedFooter{
		IconURL:      iconURL,
		Text:         text,
		ProxyIconURL: proxyURL,
	}

	return e
}

//SetImage ...
func (e *Embed) SetImage(args ...string) *Embed {
	var URL string
	var proxyURL string

	if len(args) == 0 {
		return e
	}
	if len(args) > 0 {
		URL = args[0]
	}
	if len(args) > 1 {
		proxyURL = args[1]
	}
	e.Image = &discordgo.MessageEmbedImage{
		URL:      URL,
		ProxyURL: proxyURL,
	}
	return e
}

//SetThumbnail ...
func (e *Embed) SetThumbnail(args ...string) *Embed {
	var URL string
	var proxyURL string

	if len(args) == 0 {
		return e
	}
	if len(args) > 0 {
		URL = args[0]
	}
	if len(args) > 1 {
		proxyURL = args[1]
	}
	e.Thumbnail = &discordgo.MessageEmbedThumbnail{
		URL:      URL,
		ProxyURL: proxyURL,
	}
	return e
}

//SetAuthor ...
func (e *Embed) SetAuthor(args ...string) *Embed {
	var (
		name     string
		iconURL  string
		URL      string
		proxyURL string
	)

	if len(args) == 0 {
		return e
	}
	if len(args) > 0 {
		name = args[0]
	}
	if len(args) > 1 {
		iconURL = args[1]
	}
	if len(args) > 2 {
		URL = args[2]
	}
	if len(args) > 3 {
		proxyURL = args[3]
	}

	e.Author = &discordgo.MessageEmbedAuthor{
		Name:         name,
		IconURL:      iconURL,
		URL:          URL,
		ProxyIconURL: proxyURL,
	}

	return e
}

//SetURL ...
func (e *Embed) SetURL(URL string) *Embed {
	e.URL = URL
	return e
}

//SetColor ...
func (e *Embed) SetColor(clr int) *Embed {
	e.Color = clr
	return e
}

// InlineAllFields sets all fields in the embed to be inline
func (e *Embed) InlineAllFields() *Embed {
	for _, v := range e.Fields {
		v.Inline = true
	}
	return e
}

// Truncate truncates any embed value over the character limit.
func (e *Embed) Truncate() *Embed {
	e.TruncateDescription()
	e.TruncateFields()
	e.TruncateFooter()
	e.TruncateTitle()
	return e
}

// TruncateFields truncates fields that are too long
func (e *Embed) TruncateFields() *Embed {
	if len(e.Fields) > 25 {
		e.Fields = e.Fields[:EmbedLimitField]
	}

	for _, v := range e.Fields {

		if len(v.Name) > EmbedLimitFieldName {
			v.Name = v.Name[:EmbedLimitFieldName]
		}

		if len(v.Value) > EmbedLimitFieldValue {
			v.Value = v.Value[:EmbedLimitFieldValue]
		}

	}
	return e
}

// TruncateDescription ...
func (e *Embed) TruncateDescription() *Embed {
	if len(e.Description) > EmbedLimitDescription {
		e.Description = e.Description[:EmbedLimitDescription]
	}
	return e
}

// TruncateTitle ...
func (e *Embed) TruncateTitle() *Embed {
	if len(e.Title) > EmbedLimitTitle {
		e.Title = e.Title[:EmbedLimitTitle]
	}
	return e
}

// TruncateFooter ...
func (e *Embed) TruncateFooter() *Embed {
	if e.Footer != nil && len(e.Footer.Text) > EmbedLimitFooter {
		e.Footer.Text = e.Footer.Text[:EmbedLimitFooter]
	}
	return e
}