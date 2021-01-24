<template>
    <v-app>
        <v-tabs>
            <v-tab v-for="(filename, i) in filenames" 
                   :key="i" @click="openFile(filename)"
                   style="text-transform: none !important">
                {{filename}}
            </v-tab>
        </v-tabs>
        <pre><code style="height:100%; width:100%; white-space: pre">{{fileContents}}</code></pre>
        <div v-show="status && status.length">{{status}}</div>
    </v-app>
</template>

<script>
export default {
    name: "App",
    data() {
        return {
            files: {},
            filenames: [],
            fileContents: "",
            status: "",
            ws: null
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
                if (!fileReceived.contents || this.fileContents) {
                    // null file contents: Remove existing file
                    if (this.filenames.includes(fileReceived.name)) {
                        this.filenames = this.filenames.filter(fn => fn !== fileReceived.name)
                        delete this.files[fileReceived.name]
                    }
                } else {
                    this.$set(this.files, fileReceived.name, fileReceived.contents)
                    if (!this.filenames.includes(fileReceived.name)) {
                        this.filenames.push(fileReceived.name)
                    }
                    if (!this.fileContents || !this.fileContents.length) {
                        this.openFile(fileReceived.name)
                    }
                }
                //hljs.highlightBlock(document.getElementById("viewer"))
            }
        },
        openFile(name) {
            this.fileContents = this.files[name].contents
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