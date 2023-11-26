<template>
    <v-app>
        <v-app-bar app v-show="roomCode">
            <v-tabs v-model="tab">
                <v-tab v-for="filename in filenames" 
                       :key="filename"
                       style="text-transform: none !important">
                    {{filename}}
                </v-tab>
            </v-tabs>
            <v-spacer/>
            <v-icon title="Download ZIP"
                    @click="download">mdi-zip-box</v-icon>
        </v-app-bar>
        <v-main>
            <v-alert dense 
                     v-if="!connected && roomCode"
                     :type="connected === null ? 'info' : 'error'"
                     :style="connected ? '' : 'cursor: pointer;'"
                     @click="connect">
                {{connected === null ? 'Connecting...' : 'No connection - click to retry'}}
            </v-alert>
            <v-tabs-items
                v-if="roomCode"
                v-model="tab"
            >
                <v-tab-item v-for="filename in filenames"
                    :transition="false" :reverse-transition="false"
                    style="font-family: monospace; margin: 1em; line-height: 1.2" :key="filename">
                        <pre><code style="background-color: white; padding: 0">{{files[filename]}}</code></pre>
                </v-tab-item>
            </v-tabs-items>
            <v-container v-else>
                <div>
                No room specified.
                Run the following in your terminal to open a new room:
                </div>
                <code>curl -s {{ hostName }} | python3</code>
            </v-container>
        </v-main>
    </v-app>
</template>

<script>
import JSZip from "jszip"
import { saveAs } from "file-saver"
export default {
    name: "App",
    data() {
        return {
            tab: null,
            files: {},
            filenames: [],
            connected: null,
            ws: null,
            follow: true,
            zip: new JSZip()
        }
    },
    methods: {
        connect() {
            this.connected = null
            if (!this.roomCode) {
                //this.status = "No room specified" // TODO: Use?
                this.connected = false
                return
            }
            // Quick fix to determine whether to use WSS or not
            let proto = window.location.href.startsWith("https://") ? "wss" : "ws"
            let url = `${proto}://${document.location.host}/ws/view/${this.roomCode}`
            this.ws = new WebSocket(url)
            this.ws.onclose = () => {
                this.connected = false
            }
            this.ws.onopen = () => {
                this.files = {}
                this.filenames = []
                this.connected = true
            }
            this.ws.onmessage = (msg) => {
                let fileReceived = JSON.parse(msg.data)
                if (!fileReceived.contents || !fileReceived.contents.length) {
                    // null file contents: Remove existing file
                    if (this.filenames.includes(fileReceived.name)) {
                        this.filenames = this.filenames.filter(fn => fn !== fileReceived.name)
                        delete this.files[fileReceived.name]
                    }
                } else {
                    this.$set(this.files, fileReceived.name, fileReceived.contents)
                    if (!this.filenames.includes(fileReceived.name)) {
                        this.filenames.push(fileReceived.name)
                        hljs.initHighlightingOnLoad()
                    }
                    if (this.tab === null || this.follow) {
                        this.tab = this.filenames.findIndex(fn => fn === fileReceived.name)
                    }
                }
                //hljs.highlightBlock(document.getElementById("viewer"))
            }
        },
        download() {
            for (let fn in this.files) {
                this.zip.file(fn, this.files[fn])
            }
            this.zip.generateAsync({type:"blob"}).then(content => {
                saveAs(content, this.roomCode + ".zip")
            })
        }
    },
    computed: {
        roomCode() {
            let qParams = new URLSearchParams(window.location.search)
            return qParams.get("r")
        },
        hostName() {
            return window.location.href
        }
    },
    mounted() {
        this.connect()
    }
}
</script>

<style>

</style>
