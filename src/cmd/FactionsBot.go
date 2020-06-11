package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/benkillin/ConfigHelper"
	"github.com/benkillin/GolangFactionsBot/src/EmbedHelper"
	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

var (
	configFile        = "factionsBotConfig.json"
	defaultConfigFile = "factionsBotConfig.default.json" // this file gets overwritten every run with the current default config
	botID             string                             // Bot ID
	config            *Config
)

// our main function
func main() {
	defaultConfig := &Config{
		Token:            "",
		TimerLoopTimeout: 5 * time.Second,
		Logging: LoggingConfig{
			Level:   "trace",
			Format:  "text",
			Output:  "stderr",
			Logfile: ""},
		Guilds: map[string]*GuildConfig{
			"123456789012345678": &GuildConfig{
				GuildName:     "DerpGuild",
				CommandPrefix: ".",
				BotRoleAdmin:  "523075010089189378",

				Reminders: map[string]*ReminderConfig{
					"walls": &ReminderConfig{
						ReminderName:   "walls",
						Enabled:        true,
						CheckTimeout:   45 * time.Minute,
						CheckReminder:  30 * time.Minute,
						LastChecked:    time.Now(),
						CheckChannelID: "523136929995161611",
						RoleMention:    "523130951644217363",
						Reminders:      0,
						LastReminder:   time.Now(),
					},
					"buffers": &ReminderConfig{
						ReminderName:   "buffers",
						Enabled:        true,
						CheckTimeout:   45 * time.Minute,
						CheckReminder:  30 * time.Minute,
						LastChecked:    time.Now(),
						CheckChannelID: "523136929995161611",
						RoleMention:    "523130951644217363",
						Reminders:      0,
						LastReminder:   time.Now(),
					},
				},
				Players: map[string]*PlayerConfig{
					"123456789012345678": &PlayerConfig{
						PlayerString:   "Derp#1234",
						PlayerUsername: "asdfasdfasdf",
						PlayerMention:  "@123456789012345678",
						ReminderStats: map[string]*PlayerReminderStats{
							"walls": &PlayerReminderStats{
								Weewoos:   0,
								Checks:    0,
								LastCheck: time.Now(),
							},
							"buffers": &PlayerReminderStats{
								Weewoos:   0,
								Checks:    0,
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
								clearReminderMessages(d, guildID, reminderID) // TODO: verify adding reminderID here was correct?????
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

	// loop through each reminder and check to see if it is the current channel, and then grab the assigned weewoo command and put that in the switch for executing a weewoo action.
	// TODO: Remove the generic weewoo command.
	currentChannelWeewooCmd := "DOES NOT EXIZZZZTTTTTTTLOLOL"
	for _, reminder := range config.Guilds[msg.GuildID].Reminders {
		if reminder.CheckChannelID == msg.ChannelID {
			currentChannelWeewooCmd = reminder.WeewooCommand
		}
	}

	checkGuild(d, msg.ChannelID, msg.GuildID)
	content := msg.Content
	splitContent := strings.Split(content, " ")
	prefix := config.Guilds[msg.GuildID].CommandPrefix
	switch splitContent[0] {
	case prefix + "test":
		testCmd(d, msg.ChannelID, msg, splitContent)
	case prefix + "set":
	case prefix + "clear":
		clearCmd(d, msg.ChannelID, msg, splitContent)
	case prefix + "weewoo": // TODO: update this to grab active reminder channels and check to see if there is a particular command for weewooing for this channel.
		weewooCmd(d, msg.ChannelID, msg, splitContent)
	case prefix + currentChannelWeewooCmd:
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
			// admin role check failed, check to see if user is a server administrator:
			isAdmin, err := MemberHasPermission(d, msg.GuildID, msg.Author.ID, discordgo.PermissionAdministrator)
			if err != nil {
				log.Debugf("Unable to determine if user is admin: %s", err)
				sendTempMsg(d, channelID, fmt.Sprintf("Error: Unable to determine user permissions: %s", err), 45*time.Second)
			}

			// if the user is a SA but has no role assigned that has the admin box checked, the above check will fail - this checks for server owners:
			hasAllPerms, err := MemberHasPermission(d, msg.GuildID, msg.Author.ID, discordgo.PermissionAll)
			if err != nil {
				log.Debugf("Unable to determine if user has ALL permissions: %s", err)
				sendTempMsg(d, channelID, fmt.Sprintf("Error: Unable to determine user permissions: %s", err), 45*time.Second)
			}

			if !isAdmin && !hasAllPerms {
				// user is not a server administrator, deny the set command.
				sendMsg(d, config.Guilds[msg.GuildID].BotAdminChannel, fmt.Sprintf("User %s tried to update bot settings, but does not have the correct role and is not an administrator/owner.", msg.Author.Mention()))
				sendMsg(d, msg.ChannelID, fmt.Sprintf("Role check failed. Contact someone who can assign you the correct role for reminder settings."))
				return
			}
		}

		subcommand := splitMessage[1]

		switch subcommand {
		case "reminder":

			if len(splitMessage) > 2 {
				changed := false
				reminderID := splitMessage[2]
				log.Debugf("trying to update reminder %s", reminderID)

				// does the reminder exist?
				if _, ok := config.Guilds[msg.GuildID].Reminders[reminderID]; !ok {
					log.Debugf("User tried to update settings for %s but that reminder id has not been crated.", reminderID)
					sendMsg(d, channelID, fmt.Sprintf("Error: the reminder id '%s' has not been added. Add the new reminder with the command '%sset addReminder {REMINDER_ID}'", reminderID, config.Guilds[msg.GuildID].CommandPrefix))
					return
				}

				if len(splitMessage) > 3 {
					reminderCmd := splitMessage[3]
					log.Debugf("current set command for reminder %s: %s", reminderID, reminderCmd)

					switch reminderCmd {
					case "reminderName":
						if len(splitMessage) >= 4 {
							name := strings.Join(splitMessage[4:], " ")
							config.Guilds[msg.GuildID].Reminders[reminderID].ReminderName = name
							changed = true
							sendTempMsg(d, channelID, fmt.Sprintf("Set reminder name to '%s'.", name), 45*time.Second)
						} else {
							sendTempMsg(d, channelID, "(reminderName) YOU DONE FUCKED UP, A-A-RON!", 10*time.Second)
						}
					case "weewooMsg":
						if len(splitMessage) >= 4 {
							messageForAlert := strings.Join(splitMessage[4:], " ")
							config.Guilds[msg.GuildID].Reminders[reminderID].WeewooMessage = messageForAlert
							changed = true
							sendTempMsg(d, channelID, fmt.Sprintf("Set the reminder alert message to '%s'", messageForAlert), 45*time.Second)
						} else {
							sendTempMsg(d, channelID, "(weewooMsg) YOU DONE FUCKED UP, A-A-RON!", 10*time.Second)
						}
					case "weewooEnabled":
						if len(splitMessage) > 4 {
							onOrOff := strings.ToLower(splitMessage[4])
							if onOrOff == "on" {
								config.Guilds[msg.GuildID].Reminders[reminderID].WeewoosAllowed = true
								changed = true
							} else {
								config.Guilds[msg.GuildID].Reminders[reminderID].WeewoosAllowed = false
								changed = true
							}
							sendTempMsg(d, channelID, fmt.Sprintf("Set Alert commands allowed to '%t'", config.Guilds[msg.GuildID].Reminders[reminderID].WeewoosAllowed), 45*time.Second)
						} else {
							sendTempMsg(d, channelID, "(weewooEnabled) YOU DONE FUCKED UP, A-A-RON!", 10*time.Second)
						}
					case "weewooCmd":
						if len(splitMessage) >= 4 {
							alertCommand := splitMessage[4]
							config.Guilds[msg.GuildID].Reminders[reminderID].WeewooCommand = alertCommand
							changed = true
							sendTempMsg(d, channelID, fmt.Sprintf("Set the alert commands '%s'", config.Guilds[msg.GuildID].Reminders[reminderID].WeewooCommand), 45*time.Second)
						} else {
							sendTempMsg(d, channelID, "(weewooCmd) YOU DONE FUCKED UP, A-A-RON!", 10*time.Second)
						}
					case "on":
						config.Guilds[msg.GuildID].Reminders[reminderID].Enabled = true
						changed = true
						sendTempMsg(d, channelID, fmt.Sprintf("Wall checks are now enabled."), 45*time.Second)

					case "off":
						config.Guilds[msg.GuildID].Reminders[reminderID].Enabled = false
						changed = true
						sendTempMsg(d, channelID, fmt.Sprintf("Wall checks are now disabled!!!!"), 45*time.Second)

					case "channel":
						if len(splitMessage) > 4 {
							wallsChannel := splitMessage[4]
							wallsChannelID := strings.Replace(wallsChannel, "<", "", -1)
							wallsChannelID = strings.Replace(wallsChannelID, ">", "", -1)
							wallsChannelID = strings.Replace(wallsChannelID, "#", "", -1)

							_, err := d.Channel(wallsChannelID)
							if err != nil {
								log.Errorf("Invalid channel specified while setting reminder '%s' checks channel: %s", reminderID, err)
								sendTempMsg(d, channelID, fmt.Sprintf("Invalid channel specified: %s", err), 10*time.Second)
							} else {
								config.Guilds[msg.GuildID].Reminders[reminderID].CheckChannelID = wallsChannelID
								sendTempMsg(d, channelID, fmt.Sprintf("Set channel to send reminders to <#%s>", wallsChannelID), 5*time.Second)
								changed = true
							}
						} else {
							sendTempMsg(d, channelID, "usage: "+config.Guilds[msg.GuildID].CommandPrefix+"set walls channel #channelNameForWallChecks", 10*time.Second)
						}

					case "role":
						if len(splitMessage) > 4 {
							if len(msg.MentionRoles) > 0 {
								mentionRole := msg.MentionRoles[0]
								config.Guilds[msg.GuildID].Reminders[reminderID].RoleMention = mentionRole
								changed = true
							} else {
								sendTempMsg(d, channelID, "Error - invalid/no role specified", 10*time.Second)
							}
						} else {
							sendTempMsg(d, channelID, "usage: "+config.Guilds[msg.GuildID].CommandPrefix+"set walls role @roleForWallCheckRemidners", 10*time.Second)
						}

					case "timeout":
						if len(splitMessage) > 4 {
							changed = true
							checkHourMinuteDuration(splitMessage[4], func(userDuration time.Duration) {
								config.Guilds[msg.GuildID].Reminders[reminderID].CheckTimeout = userDuration
							}, d, channelID, msg)
						}

					case "reminder":
						if len(splitMessage) > 4 {
							changed = true
							checkHourMinuteDuration(splitMessage[4], func(userDuration time.Duration) {
								config.Guilds[msg.GuildID].Reminders[reminderID].CheckReminder = userDuration
							}, d, channelID, msg)
						}

					default:
						sendCurrentReminderSettings(d, channelID, msg, reminderID)
					}

				} else {
					//user did not enter a command for the reminder id
					sendMsg(d, channelID, fmt.Sprintf("Error: no command specified to update a reminder '%s' settings. Current reminder settings:", reminderID))
					sendCurrentReminderSettings(d, channelID, msg, reminderID)
				}

				if changed {
					ConfigHelper.SaveConfig(configFile, config)
					sendCurrentReminderSettings(d, channelID, msg, reminderID)
				}
			} else {
				embed := EmbedHelper.NewEmbed().
					SetTitle("Reminder ID list").
					SetDescription("Please specify the reminder you wish to configure. Available reminders:")

				for rID, el := range config.Guilds[msg.GuildID].Reminders {
					embed.AddField(rID, el.ReminderName)
				}

				sendTempEmbed(d, channelID, embed.MessageEmbed, 60*time.Second)
			}

		case "prefix":
			if len(splitMessage) > 2 {
				prefix := splitMessage[2]
				config.Guilds[msg.GuildID].CommandPrefix = prefix
				ConfigHelper.SaveConfig(configFile, config)
			} else {
				sendTempMsg(d, channelID, "usage: "+config.Guilds[msg.GuildID].CommandPrefix+"set prefix {command prefix here. example: . or !! or ! or ¡ or ¿}", 10*time.Second)
			}

		case "admin":
			isAdmin, err := MemberHasPermission(d, msg.GuildID, msg.Author.ID, discordgo.PermissionAll)
			if err != nil {
				log.Debugf("Unable to determine if user is admin: %s", err)
				sendTempMsg(d, channelID, fmt.Sprintf("Error: Unable to determine user permissions: %s", err), 45*time.Second)
			}

			if isAdmin {
				if len(msg.MentionRoles) > 0 {
					admin := msg.MentionRoles[0]
					config.Guilds[msg.GuildID].BotRoleAdmin = admin
					ConfigHelper.SaveConfig(configFile, config)
					sendTempMsg(d, channelID, "Successfully updated bot admin role.", 60*time.Second)
				} else {
					sendTempMsg(d, channelID, "Error - invalid/no role specified", 60*time.Second)
				}
			} else {
				sendMsg(d, channelID, "Error - only server/guild administrators/owners may change this setting.")
			}

		case "adminChannel":
			isAdmin, err := MemberHasPermission(d, msg.GuildID, msg.Author.ID, discordgo.PermissionAll)
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
					sendTempMsg(d, channelID, "usage: "+config.Guilds[msg.GuildID].CommandPrefix+"set adminChannel #channelNameForWallChecks", 10*time.Second)
				}
			} else {
				sendMsg(d, channelID, "Error - only server/guild administrators/owners may change this setting.")
			}

		case "addReminder":
			if len(splitMessage) > 2 {
				newReminderID := splitMessage[2]

				// check to see if the requested reminder ID exists:
				if _, ok := config.Guilds[msg.GuildID].Reminders[newReminderID]; !ok {
					config.Guilds[msg.GuildID].Reminders[newReminderID] = &ReminderConfig{
						CheckChannelID:   "TODO: SET CHECK CHANNEL ID",
						CheckReminder:    30 * time.Minute,
						CheckTimeout:     45 * time.Minute,
						Enabled:          false,
						LastChecked:      time.Now(),
						LastReminder:     time.Now(),
						ReminderMessages: []string{},
						ReminderName:     "TODO: SET REMINDER NAME",
						Reminders:        0,
						RoleMention:      "TODO: SET ROLE",
						WeewooMessage:    "This is the default weewoo message indicating an alert for this reminder has been confirmed as in progress. You can update this message using the bot set reminder commands.",
						WeewooCommand:    "TODO: SET THE COMMAND TO TRIGGER THE ALERT FOR THIS REMINDER",
						WeewoosAllowed:   false,
					}
					ConfigHelper.SaveConfig(configFile, config)
					sendMsg(d, channelID, fmt.Sprintf("Added new reminder with ID '%s'. You must now use the '%sset reminder %s' commands for setting the reminder name, weewoo message, turning on or off weewoos, enabling the reminder, and setting the timeout, reminder interval, mention role, and channel for the reminder.", newReminderID, config.Guilds[msg.GuildID].CommandPrefix, newReminderID))
				} else {
					sendMsg(d, channelID, "Requested reminderID already exists.")
				}
			} else {
				sendTempMsg(d, channelID, "usage: "+config.Guilds[msg.GuildID].CommandPrefix+"set addReminder REMINDER_ID", 10*time.Second)
			}

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
		embed = embed.AddField(config.Guilds[msg.GuildID].CommandPrefix+command.command, command.description)
	}

	sendEmbed(d, channelID, embed.MessageEmbed)
}

// Clear command handler - marks walls clear and thanks the wall checker.
func clearCmd(d *discordgo.Session, channelID string, msg *discordgo.MessageCreate, splitMessage []string) {
	// TODO: update and uncomment
	// deleteMsg(d, msg.ChannelID, msg.ID)
	// log.Debugf("Incoming clear message: %+v", msg.Message)
	// checkGuild(d, channelID, msg.GuildID)
	// player, err := checkPlayer(d, channelID, msg.GuildID, msg.Author.ID)
	// if err != nil {
	// 	log.Errorf("Unable to check the player. %s", err)
	// 	return
	// }
	// err = checkRole(d, msg, config.Guilds[msg.GuildID].WallsRoleMention)
	// if err != nil {
	// 	sendMsg(d, config.Guilds[msg.GuildID].WallsCheckChannelID, fmt.Sprintf("User %s tried to mark walls clear, but does not have the correct role.", msg.Author.Mention()))
	// 	sendMsg(d, msg.ChannelID, fmt.Sprintf("Role check failed. Contact someone who can assign you the correct role for wall checks."))
	// 	return
	// }

	// timeTookSinceLastWallCheck := time.Now().Sub(config.Guilds[msg.GuildID].WallsLastChecked).Round(time.Second)
	// playerLastWallCheck := time.Now().Sub(config.Guilds[msg.GuildID].Players[msg.Author.ID].LastWallCheck).Round(time.Second)

	// config.Guilds[msg.GuildID].WallsLastChecked = time.Now()
	// config.Guilds[msg.GuildID].WallReminders = 0
	// config.Guilds[msg.GuildID].Players[msg.Author.ID].WallChecks++
	// config.Guilds[msg.GuildID].Players[msg.Author.ID].LastWallCheck = time.Now()

	// thankyouMessage := EmbedHelper.NewEmbed().
	// 	SetTitle("Walls clear!").
	// 	SetDescription(fmt.Sprintf(":white_check_mark: **%s** cleared the walls using command `%sclear`!",
	// 		config.Guilds[msg.GuildID].Players[msg.Author.ID].PlayerMention,
	// 		config.Guilds[msg.GuildID].CommandPrefix)).
	// 	AddField("Score", fmt.Sprintf("%d", config.Guilds[msg.GuildID].Players[msg.Author.ID].WallChecks)).
	// 	AddField("Time taken to clear", fmt.Sprintf("%s", timeTookSinceLastWallCheck)).
	// 	AddField("Time since last check", fmt.Sprintf("%s", playerLastWallCheck)).
	// 	AddField("Time Checked", config.Guilds[msg.GuildID].WallsLastChecked.Format("Jan 2, 2006 at 3:04pm (MST)")).
	// 	SetFooter(fmt.Sprintf("Thank you, %s! You rock!",
	// 		config.Guilds[msg.GuildID].Players[msg.Author.ID].PlayerUsername), "https://i.imgur.com/cCNP4qR.png").
	// 	SetThumbnail(player.AvatarURL("4096")).
	// 	MessageEmbed

	// sendEmbed(d, config.Guilds[msg.GuildID].WallsCheckChannelID, thankyouMessage)

	// go func() {
	// 	clearReminderMessages(d, msg.GuildID)
	// }()

	// ConfigHelper.SaveConfig(configFile, config)
}

// WEE WOO!!! handler. Sends an alert message indicating that a raid is in progress.
func weewooCmd(d *discordgo.Session, channelID string, msg *discordgo.MessageCreate, splitMessage []string) {
	// TODO: update and uncomment
	// deleteMsg(d, msg.ChannelID, msg.ID)
	// log.Debugf("Incoming WEEWOO! message: %+v", msg.Message)
	// checkGuild(d, channelID, msg.GuildID)
	// err := checkRole(d, msg, config.Guilds[msg.GuildID].WallsRoleMention)
	// if err != nil {
	// 	sendMsg(d, config.Guilds[msg.GuildID].WallsCheckChannelID, fmt.Sprintf("User %s tried to weewoo, but does not have the correct role.", msg.Author.Mention()))
	// 	sendMsg(d, msg.ChannelID, fmt.Sprintf("Role check failed. Contact someone who can assign you the correct role for wall checks."))
	// 	return
	// }

	// sendMsg(d, config.Guilds[msg.GuildID].WallsCheckChannelID,
	// 	fmt.Sprintf("WEEWOO!!! WEEWOO!!!! WE ARE BEING RAIDED!!!! PLEASE GET ON AND HELP DEFEND THE BASE!!!"))

	// go func() {
	// 	for i := 0; i < 3; i++ {
	// 		sendTempMsg(d, config.Guilds[msg.GuildID].WallsCheckChannelID,
	// 			fmt.Sprintf("<@&%s> WE ARE BEING RAIDED!", config.Guilds[msg.GuildID].WallsRoleMention),
	// 			30*time.Second)
	// 		time.Sleep(500 * time.Millisecond)
	// 	}
	// }()
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
