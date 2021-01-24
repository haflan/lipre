import Vue from 'vue'
import vuetify from './plugins/vuetify'
import App from './App.vue'

export const app = new Vue({
    el: "#app",
    vuetify,
    render: h => h(App)
});
