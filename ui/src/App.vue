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
            <v-badge v-show="status"/>
        </v-app-bar>
        <v-main>
            <v-tabs-items v-model="tab">
                <v-tab-item v-for="filename in filenames"
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
            status: "",
            ws: null,
            follow: true
        }
    },
    methods: {
        connect(roomCode) {
            let url = `ws://${document.location.host}/view/${roomCode}`
            this.ws = new WebSocket(url)
            this.files = {}
            this.ws.onclose = () => {
                this.status = "No connection"
            }
            this.ws.onopen = () => {
                this.status = "Connected"
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
                        this.tab = fileReceived.name
                    }
                }
                //hljs.highlightBlock(document.getElementById("viewer"))
            }
        }
    },
    mounted() {
        let qParams = new URLSearchParams(window.location.search)
        let roomCode = qParams.get("room")
        if (!roomCode) {
            this.status = "No room specified"
            return
        }
        this.connect(roomCode)
    }
}
</script>

<style>

</style>
