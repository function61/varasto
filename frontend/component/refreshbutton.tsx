import * as React from 'react';

interface RefreshButtonProps {
	refresh: () => void;
}

export class RefreshButton extends React.Component<RefreshButtonProps, {}> {
	render() {
		return (
			<span
				onClick={this.props.refresh}
				className="glyphicon glyphicon-refresh"
				style={{ cursor: 'pointer' }}
			/>
		);
	}
}
