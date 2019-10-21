import * as React from 'react';

interface LabelProps {
	title?: string;
	children: React.ReactNode;
}

export class SuccessLabel extends React.Component<LabelProps, {}> {
	render() {
		return (
			<span className="label label-success" title={this.props.title}>
				{this.props.children}
			</span>
		);
	}
}

export class WarningLabel extends React.Component<LabelProps, {}> {
	render() {
		return (
			<span className="label label-warning" title={this.props.title}>
				{this.props.children}
			</span>
		);
	}
}

export class DangerLabel extends React.Component<LabelProps, {}> {
	render() {
		return (
			<span className="label label-danger" title={this.props.title}>
				{this.props.children}
			</span>
		);
	}
}
