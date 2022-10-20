import pino from "pino"
import prompts from "prompts"
import { TelegramClient, Api } from "telegram"
import { LogLevel } from "telegram/extensions/Logger"
import { StoreSession } from "telegram/sessions"
import { setRandomInterval, T_MINS } from "./utils"

const LOG_LEVEL = process.env.LOG_LEVEL ?? "info"
const log = pino({
  level: LOG_LEVEL,
  ...(process.env.NODE_ENV === "development" && {
    transport: {
      target: "pino-pretty",
      options: { colorize: true },
      level: LOG_LEVEL,
    },
  }),
})
const telegramLog = log.child({ from: "telegram" })
const debugDumpLog = pino({
  level: "trace",
  transport: {
    target: "pino/file",
    options: { destination: "./debug.log" },
    level: "trace",
  },
})

const TELEGRAM_API_ID = parseInt(process.env.API_ID ?? "")
const TELEGRAM_API_HASH = process.env.API_HASH ?? ""

const TOMAS_USERNAME = "tomas_hv"

;(async () => {
  const client = new TelegramClient(
    new StoreSession(process.env.SESSION_PATH ?? "bot.session"),
    TELEGRAM_API_ID,
    TELEGRAM_API_HASH,
    {
      connectionRetries: 5,
    }
  )
  client.logger.setLevel((process.env.LOG_LEVEL ?? "info") as LogLevel)
  client.logger.log = (level, message, color) => {
    telegramLog.debug({ level }, message, color)
  }
  await client.start({
    async phoneNumber() {
      const { value } = await prompts({
        type: "text",
        name: "value",
        message: "Phone number with country code",
        validate: (value: string) => {
          if (!value.startsWith("+")) {
            return "Please provide country code"
          } else if (value.includes(" ")) {
            return "No space allowed between country code and number"
          } else {
            return true
          }
        },
      })

      return value
    },
    async password() {
      const { value } = await prompts({
        type: "text",
        name: "value",
        message: "Password",
      })

      return value
    },
    async phoneCode() {
      const { value } = await prompts({
        type: "text",
        name: "value",
        message: "Code you received",
      })

      return value
    },
    onError(err) {
      log.error(err)
    },
  })

  log.info("Connection established")
  client.session.save()

  // client.addEventHandler(async (event) => {
  //   const message = event.message
  //   const chat = message.chat
  //   if (chat && "title" in chat && chat.title === "Dev chat") {
  //     log.debug(message)
  //     log.debug(chat)
  //   }
  // }, new NewMessage())

  async function checkForTomas() {
    const chatsResponse = await client.invoke(
      new Api.messages.GetAllChats({ exceptIds: [] })
    )
    const { chats } = chatsResponse

    if (chats.length > 0) {
      for (const chat of chats) {
        if ("title" in chat && chat.title === "Dev chat") {
          log.debug(chat)
          const participantsResponse = await client.invoke(
            new Api.channels.GetParticipants({
              channel: chat,
              filter: new Api.ChannelParticipantsRecent(),
            })
          )
          if (
            "users" in participantsResponse &&
            participantsResponse.users.length > 0
          ) {
            const { users } = participantsResponse

            let foundTomas = false
            for (const user of users) {
              if (
                "username" in user &&
                user.username &&
                user.username.length > 0 &&
                user.username === TOMAS_USERNAME
              ) {
                foundTomas = true
                break
              }
            }

            if (!foundTomas) {
              // add bozo back
              const inviteResponse = await client.invoke(
                new Api.channels.InviteToChannel({
                  channel: chat,
                  users: [TOMAS_USERNAME],
                })
              )
              log.debug(inviteResponse)
            }
          }
        }
      }
    }
  }

  setRandomInterval(
    () => checkForTomas(),
    (delay) => log.info(`Check completed. Next run will be in ${delay}ms.`),
    5 * T_MINS,
    30 * T_MINS
  )

  client.addEventHandler(async (event) => {
    if (
      event.className === "UpdateNewChannelMessage" &&
      event.message &&
      event.message.action &&
      event.message.action.className === "MessageActionChatDeleteUser"
    ) {
      await checkForTomas()
    }
  })
})()
