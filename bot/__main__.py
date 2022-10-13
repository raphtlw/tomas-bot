from dotenv import load_dotenv

load_dotenv()

import os
import asyncio
from random import randrange
from typing import Union, Optional
from telethon import TelegramClient
from telethon.tl.functions.messages import AddChatUserRequest
from telethon.tl.custom.dialog import Dialog
from telethon.tl.types import User

TELEGRAM_API_ID = int(os.environ.get("API_ID", "12345"))
TELEGRAM_API_HASH = os.environ.get("API_HASH", "0123456789abcdef0123456789abcdef")

client = TelegramClient("tomas", TELEGRAM_API_ID, TELEGRAM_API_HASH)

# Time to wait between API calls to avoid rate limit (in milliseconds)
TELEGRAM_RATE_LIMIT_MIN = 1000
TELEGRAM_RATE_LIMIT_MAX = 3000


async def wait(seconds: float = 0):
    if seconds == 0:
        seconds = randrange(TELEGRAM_RATE_LIMIT_MIN, TELEGRAM_RATE_LIMIT_MAX) / 1000

    await asyncio.sleep(seconds)


async def get_dialog_by_name(name: str) -> Optional[Dialog]:
    async for dialog in client.iter_dialogs():
        # print(dialog.name, "has ID", dialog.id)

        if dialog.name == name:
            return dialog


async def dev_chat_loop():
    while True:
        dev_chat = await get_dialog_by_name("Dev chat")
        if not dev_chat:
            break

        tomas_hv = await client.get_entity("tomas_hv")

        print("Checking if tomas is in the group...")

        found_user: Optional[User] = None
        for participant in await client.get_participants(dev_chat):
            if participant.id == tomas_hv.id:
                found_user = participant
                break

        if found_user:
            print("User @tomas_hv found!")
        else:
            print("User @tomas_hv not found, adding him...")
            await client(AddChatUserRequest(dev_chat.id, tomas_hv, fwd_limit=10))  # type: ignore
            print("User @tomas_hv added!")

        await wait(4)


async def bot_runnable():
    me = await client.get_me()

    print(me.stringify())

    dev_chat: Union[Dialog, None] = await get_dialog_by_name("Dev chat")
    if not dev_chat:
        return

    # await dev_chat.send_message("sent from tomas-bot")
    # await wait()


def main():
    with client:
        client.loop.run_until_complete(bot_runnable())
        client.loop.run_until_complete(dev_chat_loop())


if __name__ == "__main__":
    main()
