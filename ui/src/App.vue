<template>
    <v-app>
        <v-app-bar app>
            <v-tabs v-model="tab">
                <v-tab v-for="filename in filenames" 
                       :key="filename"
                       style="text-transform: none !important">
                    {{filename}}
                </v-tab>
            </v-tabs>
        </v-app-bar>
        <v-main>
            <v-alert dense 
                     v-if="!connected" 
                     :type="connected === null ? 'info' : 'error'"
                     :style="connected ? '' : 'cursor: pointer;'"
                     @click="connect">
                {{connected === null ? 'Connecting...' : 'No connection - click to retry'}}
            </v-alert>
            <v-tabs-items v-model="tab">
                <v-tab-item v-for="filename in filenames"
                    :transition="false" :reverse-transition="false"
                    style="font-family: monospace; white-space: pre; font-size:12px; margin: 1em"
                    :key="filename"><pre><code>{{files[filename]}}</code></pre></v-tab-item>
            </v-tabs-items>
        </v-main>
    </v-app>
</template>

<script>
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
        }
    },
    methods: {
        connect() {
            this.connected = null
            let qParams = new URLSearchParams(window.location.search)
            let roomCode = qParams.get("room")
            if (!roomCode) {
                //this.status = "No room specified" // TODO: Use?
                this.connected = false
                return
            }
            let url = `ws://${document.location.host}/view/${roomCode}`
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
        }
    },
    mounted() {
        this.connect()
    }
}
</script>

<style>

</style>
