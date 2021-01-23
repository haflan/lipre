#!/usr/bin/python3

from os import listdir
from os.path import isfile, basename
import pyinotify
import json
import sys
import websocket

def send_file(filename, contents):
    fileobj = {
            "name": filename,
            "contents": contents
    }
    print(f'Sending {filename}')
    ws.send(json.dumps(fileobj))


program = sys.argv[0]
if len(sys.argv) <= 1:
    print(f"Use: {program} <room code>")
    exit(1)
room_code = sys.argv[1]

HOST="localhost:8080"

ws = websocket.WebSocket()
ws.connect(f"ws://{HOST}/pres/{room_code}")

# Initial file upload
filenames = [fn for fn in listdir() if isfile(fn)]
for fn in filenames:
    send_file(fn, open(fn).read())

# Listen for changes
class EventHandler(pyinotify.ProcessEvent):
    def process_IN_CREATE(self, event):
        fn = basename(event.pathname)
        send_file(fn, open(fn).read())
    def process_IN_DELETE(self, event):
        fn = basename(event.pathname)
        send_file(fn, None)
    def process_IN_MODIFY(self, event):
        self.process_IN_CREATE(event)

wm = pyinotify.WatchManager()
handler = EventHandler()
notifier = pyinotify.Notifier(wm, handler)
mask = pyinotify.IN_DELETE | pyinotify.IN_CREATE | pyinotify.IN_MODIFY
wm.add_watch('.', mask)
notifier.loop() 