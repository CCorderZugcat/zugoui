const path = require('path');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const MiniCssExtractPlugin = require('mini-css-extract-plugin');
const CompressionPlugin = require('compression-webpack-plugin');

module.exports = {
    mode:  'production',
    entry: './client.js',
    output: {
        path: path.resolve(__dirname, 'dist'),
        filename: 'bundle.js',
        publicPath: '',
        clean: true,
    },
    experiments: {
        topLevelAwait: true,
    },
    module: {
        rules: [
            {
                test: /wasm_exec\.js$/,
                use: ['script-loader'],
            },
            {
                test: /\.css$/i,
                use: [MiniCssExtractPlugin.loader, 'css-loader'],
            },
            {
                test: /\.wasm$/,
                type: 'asset/resource',
                generator: {
                    filename: '[name][ext]',
                },
            },
        ],
    },
    plugins: [
        new MiniCssExtractPlugin({
            filename: 'style.[contenthash:8].css',
        }),
        new HtmlWebpackPlugin({
            template: 'index.html',
            inject: 'head',
            scriptLoading: 'blocking',
        }),
        new CompressionPlugin({
            filename: '[path][base].gz',
            algorithm: 'gzip',
            test: /\.wasm$/,
        }),
        new CompressionPlugin({
            filename: '[path][base].br',
            algorithm: 'brotliCompress',
            test: /\.wasm$/,
        }),
    ],
};
