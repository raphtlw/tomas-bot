package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/updates"
	updhook "github.com/gotd/td/telegram/updates/hook"
	"github.com/gotd/td/tg"
	"github.com/joho/godotenv"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := run(ctx); err != nil {
		panic(err)
	}
}

// noSignUp can be embedded to prevent signing up.
type noSignUp struct{}

func (c noSignUp) SignUp(ctx context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, errors.New("not implemented")
}

func (c noSignUp) AcceptTermsOfService(ctx context.Context, tos tg.HelpTermsOfService) error {
	return &auth.SignUpRequired{TermsOfService: tos}
}

// termAuth implements authentication via terminal.
type termAuth struct {
	noSignUp

	phone string
}

func (a termAuth) Phone(_ context.Context) (string, error) {
	return a.phone, nil
}

func (a termAuth) Password(_ context.Context) (string, error) {
	fmt.Print("Enter 2FA password: ")
	bytePwd, err := terminal.ReadPassword(0)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytePwd)), nil
}

func (a termAuth) Code(_ context.Context, _ *tg.AuthSentCode) (string, error) {
	fmt.Print("Enter code: ")
	code, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(code), nil
}

func run(ctx context.Context) error {
	log, _ := zap.NewDevelopment(zap.IncreaseLevel(zapcore.DebugLevel), zap.AddStacktrace(zapcore.FatalLevel))
	defer func() { _ = log.Sync() }()

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	appID := os.Getenv("APP_ID")
	appHash := os.Getenv("APP_HASH")

	if len(appID) == 0 || len(appHash) == 0 {
		log.Fatal("APP_ID or APP_HASH not set properly!")
	}

	// appIDInt, err := strconv.Atoi(appID)

	phone := flag.String("phone", "", "phone number to authenticate")
	flag.Parse()

	flow := auth.NewFlow(
		termAuth{phone: *phone},
		auth.SendCodeOptions{},
	)
	dispatcher := tg.NewUpdateDispatcher()
	gaps := updates.New(updates.Config{
		Handler: dispatcher,
		Logger:  log.Named("gaps"),
	})
	client, err := telegram.ClientFromEnvironment(telegram.Options{
		Logger:        log,
		UpdateHandler: gaps,
		Middlewares: []telegram.Middleware{
			updhook.UpdateHook(gaps.Handle),
		},
	})
	if err != nil {
		return err
	}

	dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {
		log.Info("Message", zap.Any("message", update.Message))
		return nil
	})

	return client.Run(ctx, func(ctx context.Context) error {
		if err := client.Auth().IfNecessary(ctx, flow); err != nil {
			return err
		}
		log.Info("Authentication successful")

		api := client.API()
		sender := message.NewSender(api)
		chats, err := api.MessagesGetAllChats(ctx, []int64{})
		chatClasses := chats.GetChats()
		foundChatIndex := 0
		for i := range chatClasses {
			// log.Debug("Found chat", zap.Any("chat", chatClasses[i]))
			fullChat, ok := chatClasses[i].AsFull()
			if ok {
				if fullChat.GetTitle() == "Dev chat" {
					foundChatIndex = i
				}
			}
		}
		// target := sender.Resolve("Dev chat") // FIXME: Resolve this
		// target := sender.Resolve(chatClasses[foundChatIndex].GetID())
		foundChat, _ := chatClasses[foundChatIndex].AsFull()
		log.Debug("chatID", zap.Any("value", foundChat))
		target := sender.To(&tg.InputPeerChat{ChatID: foundChat.GetID()})
		if _, err := target.Text(ctx, "User @tomas_hv is in the chat"); err != nil {
			log.Fatal("Send error", zap.Error(err))
		}

		// Fetch user info.
		user, err := client.Self(ctx)
		if err != nil {
			return err
		}

		// Notify update manager about authentication.
		if err := gaps.Auth(ctx, client.API(), user.ID, user.Bot, true); err != nil {
			return err
		}
		defer func() { _ = gaps.Logout() }()

		<-ctx.Done()
		return ctx.Err()
	})
}
