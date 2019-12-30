import { Glyphicon } from 'f61ui/component/bootstrap';
import * as React from 'react';

interface RefreshButtonProps {
	refresh: () => void;
}

export class RefreshButton extends React.Component<RefreshButtonProps, {}> {
	render() {
		return (
			<a onClick={this.props.refresh} style={{ cursor: 'pointer' }}>
				<Glyphicon icon="refresh" title="Refresh" />
			</a>
		);
	}
}
