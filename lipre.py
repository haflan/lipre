#!/usr/bin/python3

from os import listdir, getenv
from os.path import isfile, basename
import pyinotify
import json
import re
import sys
import websocket
import secrets
import string

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

def generate_room_code(length=8):
    # Define the characters to use for the URL safe string
    characters = string.ascii_letters + string.digits + '-_'
    secure_string = ''.join(secrets.choice(characters) for _ in range(length))
    return secure_string

host='{{.Host}}'
linger={{.Linger}}
room_code = generate_room_code()
wsUrlBase = f'ws://{host}' if 'localhost' in host else f'wss://{host}'
httpUrlBase = f'http://{host}' if 'localhost' in host else f'https://{host}'
url = f'{wsUrlBase}/ws/pres/{room_code}?linger={linger}'

if isfile(IGNOREFILE):
    ignorefilelist = [fn for fn in open(IGNOREFILE).read().split('\n') if fn]

print('Starting room: ' + f'{httpUrlBase}/?r={room_code}')

if linger:
    print(f'Room will linger for {linger} seconds after you disconnect')

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
