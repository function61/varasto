const path = require('path');
const TsconfigPathsPlugin = require('tsconfig-paths-webpack-plugin');
const CircularDependencyPlugin = require('circular-dependency-plugin')
const webpack = require('webpack');

module.exports = {
	mode: 'development', // overridden by CLI flag on prod build
	entry: './main.tsx',
	plugins: [
		new CircularDependencyPlugin({
			exclude: /node_modules/,
			failOnError: true,
		}),
		new webpack.ProvidePlugin({
			jQuery: 'jquery/dist/jquery.slim.js', // for stupid Bootstrap
			u2f: 'u2f-api/dist/lib/generated-google-u2f-api.js',
		}),
	],
	module: {
		rules: [
			{
				test: /\.tsx?$/,
				use: 'ts-loader',
				exclude: /node_modules/
			}
		]
	},
	optimization: {
		// defaults: with prod -> true, with dev -> false
		// minify: false
	},
	performance: {
		hints: false
	},
	resolve: {
		extensions: [ '.tsx', '.ts', '.js' ],
		plugins: [new TsconfigPathsPlugin({ /*configFile: "./path/to/tsconfig.json" */ })]
	},
	output: {
		filename: 'build.js',
		library: 'main',
		path: path.resolve(__dirname, '../public')
	}
};
