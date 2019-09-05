package main

import (
    "os"
    "fmt"
    "github.com/bwmarrin/discordgo"
    "github.com/benkillin/ConfigHelper"
    log "github.com/sirupsen/logrus"
    "time"
    "strings"
    "github.com/benkillin/GolangFactionsBot/src/EmbedHelper"
)

var (
    configFile = "factionsBotConfig.json"
    defaultConfigFile = "factionsBotConfig.default.json" // this file gets overwritten every run with the current default config
    botID string // Bot ID
    config *Config
)

// our main function
func main() {
    defaultConfig := &Config{
        Token: "",
        TimerLoopTimeout: 5 * time.Second,
        Logging: LoggingConfig {
            Level: "trace",
            Format: "text",
            Output: "stderr",
            Logfile: ""},
        Guilds: map[string]*GuildConfig{
            "123456789012345678": &GuildConfig{
                GuildName: "DerpGuild",
                CommandPrefix: ".",
                BotRoleAdmin: "523075010089189378",
                
                Reminders: map[string]*ReminderConfig{
                    "walls": &ReminderConfig{
                        ReminderName: "walls",
                        Enabled: true,
                        CheckTimeout: 45*time.Minute,
                        CheckReminder: 30*time.Minute,
                        LastChecked: time.Now(),
                        CheckChannelID: "523136929995161611",
                        RoleMention: "523130951644217363",
                        Reminders: 0,
                        LastReminder: time.Now(),
                },
                    "buffers": &ReminderConfig{
                        ReminderName: "buffers",
                        Enabled: true,
                        CheckTimeout: 45*time.Minute,
                        CheckReminder: 30*time.Minute,
                        LastChecked: time.Now(),
                        CheckChannelID: "523136929995161611",
                        RoleMention: "523130951644217363",
                        Reminders: 0,
                        LastReminder: time.Now(),
                    },
                },
                Players: map[string]*PlayerConfig{
                    "123456789012345678": &PlayerConfig{
                        PlayerString: "Derp#1234",
                        PlayerUsername: "asdfasdfasdf",
                        PlayerMention: "@123456789012345678",
                        ReminderStats: map[string]*PlayerReminderStats{
                            "walls": &PlayerReminderStats{
                                Weewoos: 0,
                                Checks: 0,
                                LastCheck: time.Now(),
                            },
                            "buffers": &PlayerReminderStats{
                                Weewoos: 0,
                                Checks: 0,
                                LastCheck: time.Now(),
                            },
                        },
                    },
                },
            },
        },
    } // the default config
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
    go doTimerChecks(d)

    <-make(chan struct{})
}


func doTimerChecks(d *discordgo.Session) {
    for {
        for guildID := range config.Guilds {
            for reminderID := range config.Guilds[guildID].Reminders {
                if config.Guilds[guildID].Reminders[reminderID].Enabled {
                    lastCheckedPlusTimeout := config.Guilds[guildID].Reminders[reminderID].LastChecked.Add(config.Guilds[guildID].Reminders[reminderID].CheckTimeout)

                    if time.Now().After(lastCheckedPlusTimeout) {
                        if config.Guilds[guildID].Reminders[reminderID].Reminders == 0 {
                            config.Guilds[guildID].Reminders[reminderID].Reminders = 1

                            reminderMsgID := sendMsg(d, config.Guilds[guildID].Reminders[reminderID].CheckChannelID, 
                                fmt.Sprintf("It's time to check %s! Time last checked %s", config.Guilds[guildID].Reminders[reminderID].ReminderName, config.Guilds[guildID].Reminders[reminderID].LastChecked.Round(time.Second)))
                            config.Guilds[guildID].Reminders[reminderID].ReminderMessages = append(config.Guilds[guildID].Reminders[reminderID].ReminderMessages, reminderMsgID)
                            config.Guilds[guildID].Reminders[reminderID].LastReminder = time.Now()
                        } else {
                            lastReminderPlusReminderInterval := config.Guilds[guildID].Reminders[reminderID].LastReminder.Add(config.Guilds[guildID].Reminders[reminderID].CheckReminder)

                            if time.Now().After(lastReminderPlusReminderInterval) {
                                config.Guilds[guildID].Reminders[reminderID].Reminders++
                                durationSinceLastChecked := time.Now().Sub(config.Guilds[guildID].Reminders[reminderID].LastChecked)
                                msg := fmt.Sprintf("<@&%s>, reminder to check %s! They have still not been checked! It has been %s since the last check!", 
                                    config.Guilds[guildID].Reminders[reminderID].RoleMention, 
                                    config.Guilds[guildID].Reminders[reminderID].ReminderName,
                                    durationSinceLastChecked.Round(time.Second))
                                reminderMsgID := sendMsg(d, config.Guilds[guildID].Reminders[reminderID].CheckChannelID, msg)
                                clearReminderMessages(d, guildID)
                                config.Guilds[guildID].Reminders[reminderID].ReminderMessages = append(config.Guilds[guildID].Reminders[reminderID].ReminderMessages, reminderMsgID)
                                config.Guilds[guildID].Reminders[reminderID].LastReminder = time.Now()
                            }
                        } // end checking to see if this is an initial timeout or if a reminder that the timeout has elapsed is required.

                        ConfigHelper.SaveConfig(configFile, config)
                    } // end checking to see if we are over the timeout since the last check.
                } // end checking to see if the current reminder type is enabled.
            } // end looping over reminder types
        } // end looping over guilds

        time.Sleep(config.TimerLoopTimeout)
    }
}


// our command handler function
func messageHandler(d *discordgo.Session, msg *discordgo.MessageCreate) {
    user := msg.Author
    if user.ID == botID || user.Bot || msg.GuildID == "" {
        return
    }
    
    checkGuild(d, msg.ChannelID, msg.GuildID)
    content := msg.Content
    splitContent := strings.Split(content, " ")
    prefix := config.Guilds[msg.GuildID].CommandPrefix
    switch splitContent[0]{
    case prefix + "test":
        testCmd(d, msg.ChannelID, msg, splitContent)
    case prefix + "set":
        setCmd(d, msg.ChannelID, msg, splitContent)
    case prefix + "clear":
        clearCmd(d, msg.ChannelID, msg, splitContent)
    case prefix + "weewoo":
        weewooCmd(d, msg.ChannelID, msg, splitContent)
    case prefix + "help":
        helpCmd(d, msg.ChannelID, msg, splitContent, availableCommands)
    case prefix + "invite":
        deleteMsg(d, msg.ChannelID, msg.ID)
        ch, err := d.UserChannelCreate(msg.Author.ID)
        if err != nil {
            errmsg := fmt.Sprintf("Error creating user channel for private message with invite link: %s", err)
            log.Error(errmsg)
            sendTempMsg(d, msg.ChannelID, errmsg, 30*time.Second)
            break
        }
        sendMsg(d, ch.ID, fmt.Sprintf("Here is a link to invite this bot to your own server: https://discordapp.com/api/oauth2/authorize?client_id=%s&permissions=8&scope=bot", botID))
    case prefix + "lennyface":
        deleteMsg(d, msg.ChannelID, msg.ID)
        sendMsg(d, msg.ChannelID, "( ͡° ͜ʖ ͡°)")
    case prefix + "tableflip":
        fallthrough
    case prefix + "fliptable":
        deleteMsg(d, msg.ChannelID, msg.ID)
        sendMsg(d, msg.ChannelID, "(╯ ͠° ͟ʖ ͡°)╯┻━┻")
    case prefix + "grr":
        deleteMsg(d, msg.ChannelID, msg.ID)
        sendMsg(d, msg.ChannelID, "ಠ_ಠ")
    case prefix + "manylenny":
        fallthrough
    case prefix + "manyface":
        deleteMsg(d, msg.ChannelID, msg.ID)
        sendMsg(d, msg.ChannelID, "( ͡°( ͡° ͜ʖ( ͡° ͜ʖ ͡°)ʖ ͡°) ͡°)")
    case prefix + "finger":
        deleteMsg(d, msg.ChannelID, msg.ID)
        sendMsg(d, msg.ChannelID, "凸-_-凸")
    case prefix + "gimme":
        deleteMsg(d, msg.ChannelID, msg.ID)
        sendMsg(d, msg.ChannelID, "ლ(´ڡ`ლ)")
    case prefix + "shrug":
        deleteMsg(d, msg.ChannelID, msg.ID)
        sendMsg(d, msg.ChannelID, "¯\\_(ツ)_/¯")
    }
}

// Settings command - set the various settings that make the bot operate on a particular guild aka server.
func setCmd(d *discordgo.Session, channelID string, msg *discordgo.MessageCreate, splitMessage []string) {
    if len(splitMessage) > 1 {
        //deleteMsg(d, msg.ChannelID, msg.ID) // let's not delete settings commands in case someone does something nefarious.
        log.Debugf("Incoming settings message: %+v", msg.Message)

        checkGuild(d, channelID, msg.GuildID)
        err := checkRole(d, msg, config.Guilds[msg.GuildID].BotRoleAdmin)
        if err != nil {
            sendMsg(d, config.Guilds[msg.GuildID].WallsCheckChannelID, fmt.Sprintf("User %s tried to update bot settings, but does not have the correct role.", msg.Author.Mention()))
            sendMsg(d, msg.ChannelID, fmt.Sprintf("Role check failed. Contact someone who can assign you the correct role for wall settings."))
            return
        }

        subcommand := splitMessage[1]

        switch subcommand {
        case "walls":
            if len(splitMessage) > 2 {
                changed := false

                switch splitMessage[2] {
                case "on":
                    config.Guilds[msg.GuildID].WallsEnabled = true
                    changed = true
                    sendTempMsg(d, channelID, fmt.Sprintf("Wall checks are now enabled!"), 45 * time.Second)

                case "off":
                    config.Guilds[msg.GuildID].WallsEnabled = false
                    changed = true
                    sendTempMsg(d, channelID, fmt.Sprintf("Wall checks are now disabled."), 45 * time.Second)

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
                        sendTempMsg(d, channelID, "usage: " + config.Guilds[msg.GuildID].CommandPrefix + "set walls channel #channelNameForWallChecks", 10*time.Second)
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
                        sendTempMsg(d, channelID, "usage: " + config.Guilds[msg.GuildID].CommandPrefix + "set walls role @roleForWallCheckRemidners", 10*time.Second)
                    }

                case "timeout":
                    if len(splitMessage) > 3 {
                        changed = true
                        checkHourMinuteDuration(splitMessage[3], func(userDuration time.Duration){
                            config.Guilds[msg.GuildID].WallsCheckTimeout = userDuration}, d, channelID, msg)
                    }

                case "reminder": 
                    if len(splitMessage) > 3 {
                        changed = true
                        checkHourMinuteDuration(splitMessage[3], func(userDuration time.Duration){
                            config.Guilds[msg.GuildID].WallsCheckReminder = userDuration}, d, channelID, msg)
                    }

                default:
                    sendCurrentWallsSettings(d, channelID, msg)
                }

                if changed {
                    ConfigHelper.SaveConfig(configFile, config)
                    sendCurrentWallsSettings(d, channelID, msg)
                }
            } else {
                sendCurrentWallsSettings(d, channelID, msg)
            }

        case "prefix":
            if len(splitMessage) > 2 {
                prefix := splitMessage[2]
                config.Guilds[msg.GuildID].CommandPrefix = prefix
                ConfigHelper.SaveConfig(configFile, config)
            } else {
                sendTempMsg(d, channelID, "usage: " + config.Guilds[msg.GuildID].CommandPrefix + "set prefix {command prefix here. example: . or !! or ! or ¡ or ¿}", 10*time.Second)
            }

        case "admin":
            isAdmin, err := MemberHasPermission(d, msg.GuildID, msg.Author.ID, discordgo.PermissionAdministrator)
            if err != nil {
                log.Debugf("Unable to determine if user is admin: %s", err)
                sendTempMsg(d, channelID, fmt.Sprintf("Error: Unable to determine user permissions: %s", err), 45*time.Second)
            }

            if isAdmin {
                if len(msg.MentionRoles) > 0 {
                    admin := msg.MentionRoles[0]
                    config.Guilds[msg.GuildID].BotRoleAdmin = admin
                    ConfigHelper.SaveConfig(configFile, config)
                } else {
                    sendTempMsg(d, channelID, "Error - invalid/no role specified", 60*time.Second)
                }
            } else {
                sendMsg(d, channelID, "Error - only server/guild administrators may change this setting.")
            }

        case "adminChannel":
            isAdmin, err := MemberHasPermission(d, msg.GuildID, msg.Author.ID, discordgo.PermissionAdministrator)
            if err != nil {
                log.Debugf("Unable to determine if user is admin: %s", err)
                sendTempMsg(d, channelID, fmt.Sprintf("Error: Unable to determine user permissions: %s", err), 45*time.Second)
            }

            if isAdmin {
                if len(splitMessage) > 2 {
                    adminChannel, err := extractChannel(d, splitMessage[2])
                    if err != nil {
                        log.Errorf("Invalid channel specified while setting bot admin channel: %s", err)
                        sendTempMsg(d, channelID, fmt.Sprintf("Invalid channel specified: %s", err), 10*time.Second)
                    } else {
                        config.Guilds[msg.GuildID].BotAdminChannel = adminChannel.ID
                        sendTempMsg(d, channelID, fmt.Sprintf("Set channel to send bot admin messages to <#%s>", adminChannel.ID), 5*time.Second)
                        ConfigHelper.SaveConfig(configFile, config)
                    }
                } else {
                    sendTempMsg(d, channelID, "usage: " + config.Guilds[msg.GuildID].CommandPrefix + "set adminChannel #channelNameForWallChecks", 10*time.Second)
                }
            } else {
                sendMsg(d, channelID, "Error - only server/guild administrators may change this setting.")
            }
        case "addReminder":
            // TODO: handle adding a new reminder type.
        default: 
            helpCmd(d, channelID, msg, splitMessage, setCommands)
        }
    } else {
        helpCmd(d, channelID, msg, splitMessage, setCommands)
    }
}

func extractChannel(d *discordgo.Session, input string) (*discordgo.Channel, error) {
    channelID := strings.Replace(input, "<", "", -1)
    channelID = strings.Replace(channelID, ">", "", -1)
    channelID = strings.Replace(channelID, "#", "", -1)

    channel, err := d.Channel(channelID)
    if err != nil {
        return nil, err
    }

    return channel, nil
}

// Help command - explains the different commands the bot offers.
func helpCmd(d *discordgo.Session, channelID string, msg *discordgo.MessageCreate, splitMessage []string, commands []CmdHelp) {
    deleteMsg(d, msg.ChannelID, msg.ID)

    embed := EmbedHelper.NewEmbed().SetTitle("Available commands").SetDescription("Below are the available commands")

    for _, command := range commands {
        embed = embed.AddField(config.Guilds[msg.GuildID].CommandPrefix + command.command, command.description)
    }

    sendEmbed(d, channelID, embed.MessageEmbed)
}

// Clear command handler - marks walls clear and thanks the wall checker.
func clearCmd(d *discordgo.Session, channelID string, msg *discordgo.MessageCreate, splitMessage []string) {
    deleteMsg(d, msg.ChannelID, msg.ID)
    log.Debugf("Incoming clear message: %+v", msg.Message)
    checkGuild(d, channelID, msg.GuildID)
    player, err := checkPlayer(d, channelID, msg.GuildID, msg.Author.ID)
    if err != nil {
        log.Errorf("Unable to check the player. %s", err)
        return
    }
    err = checkRole(d, msg, config.Guilds[msg.GuildID].WallsRoleMention)
    if err != nil {
        sendMsg(d, config.Guilds[msg.GuildID].WallsCheckChannelID, fmt.Sprintf("User %s tried to mark walls clear, but does not have the correct role.", msg.Author.Mention()))
        sendMsg(d, msg.ChannelID, fmt.Sprintf("Role check failed. Contact someone who can assign you the correct role for wall checks."))
        return
    }
    
    timeTookSinceLastWallCheck := time.Now().Sub(config.Guilds[msg.GuildID].WallsLastChecked).Round(time.Second)
    playerLastWallCheck := time.Now().Sub(config.Guilds[msg.GuildID].Players[msg.Author.ID].LastWallCheck).Round(time.Second)

    config.Guilds[msg.GuildID].WallsLastChecked = time.Now()
    config.Guilds[msg.GuildID].WallReminders = 0
    config.Guilds[msg.GuildID].Players[msg.Author.ID].WallChecks++
    config.Guilds[msg.GuildID].Players[msg.Author.ID].LastWallCheck = time.Now()
    
    thankyouMessage := EmbedHelper.NewEmbed().
        SetTitle("Walls clear!").
        SetDescription(fmt.Sprintf(":white_check_mark: **%s** cleared the walls using command `%sclear`!",
            config.Guilds[msg.GuildID].Players[msg.Author.ID].PlayerMention, 
            config.Guilds[msg.GuildID].CommandPrefix)).
        AddField("Score", fmt.Sprintf("%d", config.Guilds[msg.GuildID].Players[msg.Author.ID].WallChecks)).
        AddField("Time taken to clear", fmt.Sprintf("%s", timeTookSinceLastWallCheck)).
        AddField("Time since last check", fmt.Sprintf("%s", playerLastWallCheck)).
        AddField("Time Checked", config.Guilds[msg.GuildID].WallsLastChecked.Format("Jan 2, 2006 at 3:04pm (MST)")).
        SetFooter(fmt.Sprintf("Thank you, %s! You rock!",
            config.Guilds[msg.GuildID].Players[msg.Author.ID].PlayerUsername), "https://i.imgur.com/cCNP4qR.png").
        SetThumbnail(player.AvatarURL("4096")).
        MessageEmbed

    sendEmbed(d, config.Guilds[msg.GuildID].WallsCheckChannelID, thankyouMessage)

    go func() {
        clearReminderMessages(d, msg.GuildID)
    } ()

    ConfigHelper.SaveConfig(configFile, config)
}

// WEE WOO!!! handler. Sends an alert message indicating that a raid is in progress.
func weewooCmd(d *discordgo.Session, channelID string, msg *discordgo.MessageCreate, splitMessage []string) {
    deleteMsg(d, msg.ChannelID, msg.ID)
    log.Debugf("Incoming WEEWOO! message: %+v", msg.Message)
    checkGuild(d, channelID, msg.GuildID)
    err := checkRole(d, msg, config.Guilds[msg.GuildID].WallsRoleMention)
    if err != nil {
        sendMsg(d, config.Guilds[msg.GuildID].WallsCheckChannelID, fmt.Sprintf("User %s tried to weewoo, but does not have the correct role.", msg.Author.Mention()))
        sendMsg(d, msg.ChannelID, fmt.Sprintf("Role check failed. Contact someone who can assign you the correct role for wall checks."))
        return
    }

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
