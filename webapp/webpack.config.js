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
        alias: {
            // Mattermost webapp components are provided at runtime
            'components/admin_console/plugin_settings': path.resolve(__dirname, 'src/types/modules.d.ts'),
            'components/admin_console/settings/text_setting': path.resolve(__dirname, 'src/types/modules.d.ts'),
            'components/widgets/buttons': path.resolve(__dirname, 'src/types/modules.d.ts'),
        },
    },
    externals: {
        react: 'React',
        'react-dom': 'ReactDOM',
        // Mattermost webapp provides these at runtime
        'components/admin_console/plugin_settings': 'PluginSettings',
        'components/admin_console/settings/text_setting': 'TextSetting',
        'components/widgets/buttons': 'Button',
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

