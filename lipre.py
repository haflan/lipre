#!/usr/bin/python3

from os import listdir, getenv
from os.path import isfile, basename
import pyinotify
import json
import re
import sys
import websocket

IGNOREFILE='.lpignore'
ignorefilelist = []

def should_ignore(filename):
    for to_ignore in ignorefilelist:
        # Make a regex from the asterix
        match = re.search(to_ignore.replace("*", ".*"), filename)
        if match:
            return True
    return False

def send_file(filename):
    if should_ignore(filename):
        return
    if isfile(filename):
        contents = open(filename).read()
    else:
        contents = None
    fileobj = {
            'name': filename,
            'contents': contents
    }
    print(f'Sending {filename}')
    ws.send(json.dumps(fileobj))

def closed():
    print('Connection closed')
    exit()

program = sys.argv[0]
if len(sys.argv) <= 1:
    print(f'Use: {program} <room code> [host]')
    exit(1)
room_code = sys.argv[1]

if getenv('LPHOST'):
    HOST = getenv('LPHOST')
elif len(sys.argv) >= 3:
    HOST = sys.argv[2]
else:
    HOST = 'ws://localhost:8080'

if ':' in room_code:
    code, linger = room_code = room_code.split(':')
    url = f'{HOST}/ws/pres/{code}?linger={linger}'
else:
    url = f'{HOST}/ws/pres/{room_code}'

if isfile(IGNOREFILE):
    ignorefilelist = [fn for fn in open(IGNOREFILE).read().split('\n') if fn]


# Listen for changes
class EventHandler(pyinotify.ProcessEvent):
    def process_IN_CREATE(self, event):
        fn = basename(event.pathname)
        send_file(fn)
    def process_IN_DELETE(self, event):
        fn = basename(event.pathname)
        send_file(fn)
    def process_IN_MODIFY(self, event):
        self.process_IN_CREATE(event)

def present():
    # Initial file upload
    filenames = [fn for fn in listdir() if isfile(fn)]
    for fn in filenames:
        send_file(fn)
    # Continously watch for changes
    wm = pyinotify.WatchManager()
    handler = EventHandler()
    notifier = pyinotify.Notifier(wm, handler)
    mask = pyinotify.IN_DELETE | pyinotify.IN_CREATE | pyinotify.IN_MODIFY
    wm.add_watch('.', mask)
    notifier.loop()

ws = websocket.WebSocket()
ws.connect(url)
ws.on_close = closed
present()
