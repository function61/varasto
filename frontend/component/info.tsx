import * as React from 'react';

interface InfoProps {
	text: string;
}

export class Info extends React.Component<InfoProps, {}> {
	render() {
		return <span className="glyphicon glyphicon-info-sign" title={this.props.text} />;
	}
}
