package notificationhandler

import (
	"context"
	"fmt"
	"time"

	"github.com/kubescape/go-logger"
	"github.com/kubescape/go-logger/helpers"
	"github.com/kubescape/operator/utils"

	"github.com/armosec/cluster-notifier-api-go/notificationserver"
	"github.com/gorilla/websocket"
)

type NotificationHandler struct {
	connector  IWebsocketActions
	sessionObj *chan utils.SessionObj
}

func NewNotificationHandler(sessionObj *chan utils.SessionObj) *NotificationHandler {
	urlStr := initNotificationServerURL()

	return &NotificationHandler{
		connector:  NewWebsocketActions(urlStr),
		sessionObj: sessionObj,
	}
}

func (notification *NotificationHandler) WebsocketConnection(ctx context.Context) error {
	if utils.ClusterConfig.GatewayWebsocketURL == "" {
		return nil
	}
	retries := 0
	for {
		if err := notification.setupWebsocket(ctx); err != nil {
			retries += 1
			time.Sleep(time.Duration(retries*2) * time.Second)
			logger.L().Ctx(ctx).Warning("In WebsocketConnection", helpers.Error(err), helpers.Int("retry", retries))
		} else {
			retries = 0
		}
	}
}

// Websocket main function
func (notification *NotificationHandler) setupWebsocket(ctx context.Context) error {
	errs := make(chan error)
	_, err := notification.connector.DefaultDialer(nil)
	if err != nil {
		logger.L().Ctx(ctx).Error("In SetupWebsocket", helpers.Error(err))
		return err
	}
	defer notification.connector.Close()
	go func() {
		if err := notification.websocketPingMessage(ctx); err != nil {
			logger.L().Ctx(ctx).Error(err.Error(), helpers.Error(err))
			errs <- err
		}
	}()
	go func() {
		logger.L().Info("Waiting for websocket to receive notifications")
		if err := notification.websocketReceiveNotification(ctx); err != nil {
			logger.L().Ctx(ctx).Error(err.Error(), helpers.Error(err))
			errs <- err
		}
	}()

	return <-errs
}
func (notification *NotificationHandler) websocketReceiveNotification(ctx context.Context) error {
	for {
		messageType, messageBytes, err := notification.connector.ReadMessage()
		if err != nil {
			return fmt.Errorf("error receiving data from notificationServer. message: %s", err.Error())
		}

		switch messageType {
		case websocket.TextMessage, websocket.BinaryMessage:
			var notif *notificationserver.Notification
			switch messageBytes[0] {
			case '{', '[', '"':
				notif, err = decodeJsonNotification(messageBytes)
				if err != nil {
					notif, err = decodeBsonNotification(messageBytes)
					if err != nil {
						logger.L().Ctx(ctx).Error("failed to handle notification", helpers.String("messageBytes", string(messageBytes)), helpers.Error(err))
						continue
					}
				}
			default:
				notif, err = decodeBsonNotification(messageBytes)
				if err != nil {
					logger.L().Ctx(ctx).Error("failed to handle notification as BSON", helpers.String("messageBytes", string(messageBytes)), helpers.Error(err))
					continue
				}
			}

			err := notification.handleNotification(ctx, notif)
			if err != nil {
				logger.L().Ctx(ctx).Error("failed to handle notification", helpers.String("messageBytes", string(messageBytes)), helpers.Error(err))
			}
		case websocket.CloseMessage:
			return fmt.Errorf("websocket closed by server, message: %s", string(messageBytes))
		default:
			logger.L().Info(fmt.Sprintf("Unrecognized message received. received: %d", messageType))
		}
	}
}
