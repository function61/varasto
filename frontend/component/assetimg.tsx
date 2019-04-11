import { globalConfig } from 'f61ui/globalconfig';
import * as React from 'react';

interface AssetImgProps {
	// TODO: have this backed by enum like:
	//   src: '/collection.png' | '/directory.png' | '/file.png';
	src: string;
	width?: number;
	height?: number;
}

export class AssetImg extends React.Component<AssetImgProps, {}> {
	render() {
		// FIXME: path descension is a hack
		return (
			<img
				src={globalConfig().assetsDir + '/..' + this.props.src}
				alt=""
				width={this.props.width}
				height={this.props.height}
			/>
		);
	}
}
