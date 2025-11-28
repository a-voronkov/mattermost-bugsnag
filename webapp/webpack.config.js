const path = require('path');

module.exports = {
    entry: './src/index.tsx',
    output: {
        path: path.resolve(__dirname, 'dist'),
        filename: 'main.js',
        library: {
            type: 'umd',
        },
    },
    resolve: {
        extensions: ['.ts', '.tsx', '.js', '.jsx'],
        modules: [path.resolve(__dirname, 'src'), 'node_modules'],
        alias: {
            // Mattermost webapp components - use stubs for build
            'components/admin_console/plugin_settings$': path.resolve(__dirname, 'src/stubs/plugin_settings.ts'),
            'components/admin_console/settings/text_setting$': path.resolve(__dirname, 'src/stubs/text_setting.ts'),
            'components/widgets/buttons$': path.resolve(__dirname, 'src/stubs/buttons.ts'),
        },
    },
    externals: {
        react: 'React',
        'react-dom': 'ReactDOM',
    },
    module: {
        rules: [
            {
                test: /\.tsx?$/,
                use: 'ts-loader',
                exclude: /node_modules/,
            },
        ],
    },
    mode: 'production',
    devtool: 'source-map',
};

