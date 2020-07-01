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

func checkGuild(d *discordgo.Session, channelID string, GuildID string) (*discordgo.Guild, error) {
	guild, err := d.Guild(GuildID)
	if err != nil {
		log.Errorf("Error obtaining guild: %s", err)
		sendMsg(d, channelID, fmt.Sprintf("Error obtaining guild: %s", err))
		return nil, err
	}

	if _, ok := config.Guilds[GuildID]; !ok {
		players := make(map[string]*PlayerConfig)
		reminders := make(map[string]*ReminderConfig)
		config.Guilds[GuildID] = &GuildConfig{
			GuildName:           guild.Name,
			Reminders:           reminders,
			CommandPrefix:       ".",
			Players:             players,
			SecretAdmin:         "123456789asdfghjkl",
			MinimumClearTimeout: 1 * time.Minute,
		}
	} else {
		if guild.Name != config.Guilds[GuildID].GuildName {
			config.Guilds[GuildID].GuildName = guild.Name
		}
	}

	ConfigHelper.SaveConfig(configFile, config)

	return guild, nil
}

func checkPlayer(d *discordgo.Session, channelID string, GuildID string, authorID string, reminderID string, reminderAvail bool) (*discordgo.User, error) {
	checkGuild(d, channelID, GuildID)
	player, err := d.User(authorID)
	if err != nil {
		log.Errorf("Error obtaining user information: %s", err)
		sendMsg(d, channelID, fmt.Sprintf("Error obtaining user information: %s", err))
		return nil, err
	}

	if _, ok := config.Guilds[GuildID].Players[player.ID]; !ok {

		blankReminderStats := make(map[string]*PlayerReminderStats)

		config.Guilds[GuildID].Players[player.ID] = &PlayerConfig{
			PlayerString:   player.String(),
			PlayerUsername: player.Username,
			PlayerMention:  player.Mention(),
			ReminderStats:  blankReminderStats,
		}
	} else {
		if player.Username != config.Guilds[GuildID].Players[authorID].PlayerString {
			config.Guilds[GuildID].Players[authorID].PlayerString = player.String()
			config.Guilds[GuildID].Players[authorID].PlayerUsername = player.Username
			config.Guilds[GuildID].Players[authorID].PlayerMention = player.Mention()
		}
	}

	// at this point we now know the player exists...
	if _, ok := config.Guilds[GuildID].Players[player.ID].ReminderStats[reminderID]; !ok {
		// create the reminder if there is not an entry in the map for it
		if reminderAvail {
			config.Guilds[GuildID].Players[player.ID].ReminderStats[reminderID] = &PlayerReminderStats{
				Checks:    0,
				LastCheck: time.Now().AddDate(0, 0, -1),
				Weewoos:   0,
			}
		}
	}

	return player, nil
}

// check to see if the user is in the specified role, or is an administrator.
func checkRole(d *discordgo.Session, msg *discordgo.MessageCreate, requiredRole string) error {
	member, err := d.GuildMember(msg.GuildID, msg.Author.ID)
	if err != nil {
		log.Errorf("Error obtaining user information: %s", err)
		sendMsg(d, msg.ChannelID, fmt.Sprintf("Error obtaining user information: %s", err))
		return err
	}

	for _, role := range member.Roles {
		if role == requiredRole {
			log.Debugf("User passed role check.")
			return nil
		}
	}

	isAdmin, err := MemberHasPermission(d, msg.GuildID, msg.Author.ID, discordgo.PermissionAdministrator)
	if err != nil {
		log.Debugf("Unable to determine if user is admin: %s", err)
		return err
	}

	if isAdmin {
		log.Debugf("User passed role check (user is administrator).")
		return nil
	}

	log.Errorf("User %s <%s (%s)> does not have the correct role (%s).", msg.Author.Username, member.Nick, msg.Author.Mention(), requiredRole)
	return fmt.Errorf("user %s (%s) does not have the necessary role %s", msg.Author.Mention(), msg.Author.ID, requiredRole)
}

// support func for setting the walls timeout and reminder duration.
func checkHourMinuteDuration(userInputDuration string, handler func(userDuration time.Duration), d *discordgo.Session, channelID string, msg *discordgo.MessageCreate) {
	if strings.HasSuffix(userInputDuration, "m") || strings.HasSuffix(userInputDuration, "h") {
		userDuration, err := time.ParseDuration(userInputDuration)
		if err != nil {
			log.Errorf("User specified invalid duration.")
			sendTempMsg(d, channelID, fmt.Sprintf("Error - invalid duration: %s", err), 30*time.Second)
			return
		}
		handler(userDuration)
	} else {
		log.Errorf("User specified invalid suffix for time duration.")
		sendTempMsg(d, channelID, "Error - invalid time units. You must specify 'm' or 'h'.", 30*time.Second)
		return
	}
}

// send the current walls settings to the specified channel.
func sendCurrentReminderSettings(d *discordgo.Session, channelID string, msg *discordgo.MessageCreate, reminderID string) {
	if _, ok := config.Guilds[msg.GuildID].Reminders[reminderID]; !ok {
		log.Errorf("sendCurrentReminderSettings: Reminder settings requested to be sent for reminderID '%s' but it does not exist.", reminderID)
		return
	}

	embed := EmbedHelper.NewEmbed().
		SetTitle(fmt.Sprintf("%s settings", config.Guilds[msg.GuildID].Reminders[reminderID].ReminderName)).
		SetDescription(fmt.Sprintf("Current %s settings", reminderID)).
		AddField("Guild Name", config.Guilds[msg.GuildID].GuildName).
		AddField("Bot admin role", "<@&"+config.Guilds[msg.GuildID].BotRoleAdmin+">").
		AddField("Reminder ID", reminderID).
		AddField("Reminder Name", config.Guilds[msg.GuildID].Reminders[reminderID].ReminderName).
		AddField("Reminder enabled", fmt.Sprintf("%t", config.Guilds[msg.GuildID].Reminders[reminderID].Enabled)).
		AddField("Role to mention", "<@&"+config.Guilds[msg.GuildID].Reminders[reminderID].RoleMention+">").
		AddField("Check channel", "<#"+config.Guilds[msg.GuildID].Reminders[reminderID].CheckChannelID+">").
		AddField("Reminder check reminder", fmt.Sprintf("%s", config.Guilds[msg.GuildID].Reminders[reminderID].CheckReminder)).
		AddField("Reminder check interval", fmt.Sprintf("%s", config.Guilds[msg.GuildID].Reminders[reminderID].CheckTimeout)).
		AddField("Reminder last checked", fmt.Sprintf("%s", config.Guilds[msg.GuildID].Reminders[reminderID].LastChecked)).
		AddField("Reminder alert message", fmt.Sprintf("%s", config.Guilds[msg.GuildID].Reminders[reminderID].WeewooMessage)).
		AddField("Reminder alert command", fmt.Sprintf("%s", config.Guilds[msg.GuildID].Reminders[reminderID].WeewooCommand)).
		AddField("Reminder alert enabled?", fmt.Sprintf("%t", config.Guilds[msg.GuildID].Reminders[reminderID].WeewoosAllowed)).
		MessageEmbed

	sendTempEmbed(d, channelID, embed, 60*time.Second)
}

// helper func to send an embed message, aka a message that has a bunch of key value pairs and other things like images and stuff.
func sendEmbed(d *discordgo.Session, channelID string, embed *discordgo.MessageEmbed) (*discordgo.Message, error) {
	msg, err := d.ChannelMessageSendEmbed(channelID, embed)

	if err != nil {
		log.Errorf("Error sending embed message: %s", err)
		return nil, err
	}

	return msg, nil
}

// sends an embed message and waits the specified duration in a separate goroutine prior to deleting the message.
func sendTempEmbed(d *discordgo.Session, channelID string, embed *discordgo.MessageEmbed, duration time.Duration) (*discordgo.Message, error) {
	msg, err := d.ChannelMessageSendEmbed(channelID, embed)

	if err != nil {
		log.Errorf("Error sending temp embed message: %s", err)
		return nil, err
	}

	go func() {
		time.Sleep(duration)
		deleteMsg(d, channelID, msg.ID)
	}()

	return msg, nil
}

func getCurChannelWeewooCmdAndReminderID(msg *discordgo.MessageCreate) (string, string, bool) {
	// loop through each reminder and check to see if it is the current channel, and then grab the assigned weewoo command and put that in the switch for executing a weewoo action.
	currentChannelWeewooCmd := "DOES NOT EXIZZZZTTTTTTTLOLOL"
	currentChannelReminderID := "DOES NOT EXIST"
	foundChannel := false
	for rid, reminder := range config.Guilds[msg.GuildID].Reminders {
		if reminder.CheckChannelID == msg.ChannelID {
			currentChannelWeewooCmd = reminder.WeewooCommand
			currentChannelReminderID = rid
			foundChannel = true
		}
	}

	return currentChannelWeewooCmd, currentChannelReminderID, foundChannel
}

// clear the reminder messages that the bot has sent out for wall checks.
func clearReminderMessages(d *discordgo.Session, GuildID string, reminderID string) {
	//for i := 0; i < len(config.Guilds[GuildID].Reminders[reminderID].ReminderMessages); i++ {
	for _, element := range config.Guilds[GuildID].Reminders[reminderID].ReminderMessages {
		//messageID := config.Guilds[GuildID].ReminderMessages[i]
		messageID := element
		deleteMsg(d, config.Guilds[GuildID].Reminders[reminderID].CheckChannelID, messageID)
		time.Sleep(1500 * time.Millisecond)
	}
	config.Guilds[GuildID].Reminders[reminderID].ReminderMessages = config.Guilds[GuildID].Reminders[reminderID].ReminderMessages[:0]
	ConfigHelper.SaveConfig(configFile, config)
}

// extract the channel ID
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

// test func for the unit tests - can be removed if we can figure out how to do unit testing with the discord api mocked somehow.
func hello() string {
	return "Hello, world!"
}

// send a message including a typing notification.
func sendMsg(d *discordgo.Session, channelID string, msg string) string {
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
func deleteMsg(d *discordgo.Session, channelID string, messageID string) error {
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
		log.SetOutput(os.Stdout) // by default the package outputs to stderr
	} else if config.Logging.Output == "stderr" {
		// do nothing
	} else {
		log.Warn("Warning: log output option not recognized. Valid options are 'file' 'stdout' 'stderr' for config.Logging.output")
	}
}

// remove an element from a string array.
func remove(s []string, i int) []string {
	s[len(s)-1], s[i] = s[i], s[len(s)-1]
	return s[:len(s)-1]
}

// insert an element at index in an array
// https://stackoverflow.com/questions/46128016/insert-a-value-in-a-slice-at-a-given-index
func insertStats(a []TopStatInfo, c TopStatInfo, i int) []TopStatInfo {
	return append(a[:i], append([]TopStatInfo{c}, a[i:]...)...)
}

// MemberHasPermission checks if a member has the given permission
// for example, If you would like to check if user has the administrator
// permission you would use
// --- MemberHasPermission(s, guildID, userID, discordgo.PermissionAdministrator)
// If you want to check for multiple permissions you would use the bitwise OR
// operator to pack more bits in. (e.g): PermissionAdministrator|PermissionAddReactions
// =================================================================================
//     s          :  discordgo session
//     guildID    :  guildID of the member you wish to check the roles of
//     userID     :  userID of the member you wish to retrieve
//     permission :  the permission you wish to check for
// from https://github.com/bwmarrin/discordgo/wiki/FAQ#permissions-and-roles
func MemberHasPermission(s *discordgo.Session, guildID string, userID string, permission int) (bool, error) {
	member, err := s.State.Member(guildID, userID)
	if err != nil {
		if member, err = s.GuildMember(guildID, userID); err != nil {
			return false, err
		}
	}

	// Iterate through the role IDs stored in member.Roles
	// to check permissions
	for _, roleID := range member.Roles {
		role, err := s.State.Role(guildID, roleID)
		if err != nil {
			return false, err
		}
		if role.Permissions&permission != 0 {
			return true, nil
		}
	}

	return false, nil
}
