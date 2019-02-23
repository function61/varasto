import { globalConfig } from 'f61ui/globalconfig';
import * as React from 'react';

interface AssetImgProps {
	src: '/collection.png' | '/directory.png' | '/file.png';
}

export class AssetImg extends React.Component<AssetImgProps, {}> {
	render() {
		// FIXME: path descension is a hack
		return <img src={globalConfig().assetsDir + '/..' + this.props.src} alt="" />;
	}
}
