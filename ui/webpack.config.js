const path = require('path');
const VueLoaderPlugin = require('vue-loader/lib/plugin');
const VuetifyLoaderPlugin = require('vuetify-loader/lib/plugin');

module.exports = {
    entry: "./src/app.js",
    output: {
        path: __dirname, 
        filename: "lipre.js"
    },
    module: {
        rules: [
            {test: /\.js$/, use: 'babel-loader'},
            {test: /\.vue$/, use: 'vue-loader'},
            {
                test: /\.s(c|a)ss$/,
                use: [
                    'vue-style-loader',
                    'css-loader',
                    {
                        loader: 'sass-loader',
                        options: {
                            implementation: require('sass'),
                            sassOptions: {
                                indentedSyntax: true
                            },
                        },
                    }
                ],
            },
        ],
    },
    plugins:[
        new VueLoaderPlugin(),
        new VuetifyLoaderPlugin()
    ]
}
