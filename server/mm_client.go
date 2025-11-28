package main

import (
	"encoding/json"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// MMClient wraps the plugin.API with small, typed helpers so the plugin code can
// evolve without scattering low-level API calls everywhere.
type MMClient struct {
	api         plugin.API
	debug       bool
	kvNamespace string
	botUserID   string
}

func newMMClient(api plugin.API, debug bool, namespace, botUserID string) *MMClient {
	return &MMClient{api: api, debug: debug, kvNamespace: namespace, botUserID: botUserID}
}

func (c *MMClient) CreatePost(channelID, message string, attachments []*model.SlackAttachment) (*model.Post, *model.AppError) {
	post := &model.Post{
		ChannelId: channelID,
		Message:   message,
		UserId:    c.botUserID,
	}

	if len(attachments) > 0 {
		post.Props = map[string]interface{}{"attachments": attachments}
	}

	return c.api.CreatePost(post)
}

func (c *MMClient) CreateReply(channelID, rootPostID, message string) (*model.Post, *model.AppError) {
	post := &model.Post{ChannelId: channelID, Message: message, RootId: rootPostID}
	return c.api.CreatePost(post)
}

func (c *MMClient) UpdatePost(post *model.Post) (*model.Post, *model.AppError) {
	return c.api.UpdatePost(post)
}

func (c *MMClient) GetPost(postID string) (*model.Post, *model.AppError) {
	return c.api.GetPost(postID)
}

func (c *MMClient) GetChannel(channelID string) (*model.Channel, *model.AppError) {
	return c.api.GetChannel(channelID)
}

func (c *MMClient) GetUser(userID string) (*model.User, *model.AppError) {
	return c.api.GetUser(userID)
}

func (c *MMClient) StoreJSON(key string, value any) *model.AppError {
	data, err := json.Marshal(value)
	if err != nil {
		return model.NewAppError("KVSet", "app.plugin.json_marshal.app_error", nil, err.Error(), 0)
	}

	return c.api.KVSet(c.namespaced(key), data)
}

func (c *MMClient) LoadJSON(key string, dest any) (bool, *model.AppError) {
	data, appErr := c.api.KVGet(c.namespaced(key))
	if appErr != nil {
		return false, appErr
	}

	if data == nil {
		return false, nil
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return false, model.NewAppError("KVGet", "app.plugin.json_unmarshal.app_error", nil, err.Error(), 0)
	}

	return true, nil
}

func (c *MMClient) LogDebug(msg string, keyValuePairs ...interface{}) {
	if c.debug {
		c.api.LogDebug(msg, keyValuePairs...)
	}
}

func (c *MMClient) namespaced(key string) string {
	if strings.TrimSpace(c.kvNamespace) == "" {
		return key
	}
	return c.kvNamespace + ":" + key
}
