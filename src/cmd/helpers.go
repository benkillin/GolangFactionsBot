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
		config.Guilds[GuildID] = &GuildConfig{
			GuildName: guild.Name,
			Reminders: &ReminderConfig{
				CheckChannelID:   "channelid",
				CheckReminder:    30 * time.Minute,
				CheckTimeout:     45 * time.Minute,
				Enabled:          false,
				LastChecked:      1 * time.Minute,
				LastReminder:     1 * time.Minute,
				ReminderMessages: [""],
				ReminderName:     "DEFAULTREMINDER",
				Reminders:        0,
				RoleMention:      "",
				WeewooMessage:    "WEEEEEOOOOO",
				WeewoosAllowed:   false,
			},
			CommandPrefix: ".",
			Players:       players}
	} else {
		if guild.Name != config.Guilds[GuildID].GuildName {
			config.Guilds[GuildID].GuildName = guild.Name
		}
	}

	ConfigHelper.SaveConfig(configFile, config)

	return guild, nil
}

func checkPlayer(d *discordgo.Session, channelID string, GuildID string, authorID string) (*discordgo.User, error) {
	checkGuild(d, channelID, GuildID)
	player, err := d.User(authorID)
	if err != nil {
		log.Errorf("Error obtaining user information: %s", err)
		sendMsg(d, channelID, fmt.Sprintf("Error obtaining user information: %s", err))
		return nil, err
	}

	if _, ok := config.Guilds[GuildID].Players[player.ID]; !ok {
		config.Guilds[GuildID].Players[player.ID] = &PlayerConfig{
			PlayerString:   player.String(),
			PlayerUsername: player.Username,
			PlayerMention:  player.Mention(),
			WallChecks:     0,
			LastWallCheck:  time.Time{}}
	} else {
		if player.Username != config.Guilds[GuildID].Players[authorID].PlayerString {
			config.Guilds[GuildID].Players[authorID].PlayerString = player.String()
			config.Guilds[GuildID].Players[authorID].PlayerUsername = player.Username
			config.Guilds[GuildID].Players[authorID].PlayerMention = player.Mention()
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

	log.Errorf("User %s <%s (%s)> does not have the correct role.", msg.Author.Username, member.Nick, msg.Author.Mention())
	return fmt.Errorf("user %s (%s) does not have the necessary role %s", msg.Author.Mention(), msg.Author.ID, config.Guilds[msg.GuildID].WallsRoleMention)
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
func sendCurrentWallsSettings(d *discordgo.Session, channelID string, msg *discordgo.MessageCreate) {
	embed := EmbedHelper.NewEmbed().
		SetTitle("Walls settings").
		SetDescription("Current walls settings").
		AddField("Guild Name", config.Guilds[msg.GuildID].GuildName).
		AddField("Checks enabled", fmt.Sprintf("%t", config.Guilds[msg.GuildID].WallsEnabled)).
		AddField("Role to mention", "<@&"+config.Guilds[msg.GuildID].WallsRoleMention+">").
		AddField("Bot admin role", "<@&"+config.Guilds[msg.GuildID].WallsRoleAdmin+">").
		AddField("Check channel", "<#"+config.Guilds[msg.GuildID].WallsCheckChannelID+">").
		AddField("Walls check reminder", fmt.Sprintf("%s", config.Guilds[msg.GuildID].WallsCheckReminder)).
		AddField("Walls check interval", fmt.Sprintf("%s", config.Guilds[msg.GuildID].WallsCheckTimeout)).
		AddField("Walls last checked", fmt.Sprintf("%s", config.Guilds[msg.GuildID].WallsLastChecked)).
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

// clear the reminder messages that the bot has sent out for wall checks.
func clearReminderMessages(d *discordgo.Session, GuildID string) {
	for i := 0; i < len(config.Guilds[GuildID].ReminderMessages); i++ {
		messageID := config.Guilds[GuildID].ReminderMessages[i]
		deleteMsg(d, config.Guilds[GuildID].WallsCheckChannelID, messageID)
		time.Sleep(1500 * time.Millisecond)
	}
	config.Guilds[GuildID].ReminderMessages = config.Guilds[GuildID].ReminderMessages[:0]
	ConfigHelper.SaveConfig(configFile, config)
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
