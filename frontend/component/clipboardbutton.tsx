import { elToClipboard } from 'f61ui/clipboard';
import * as React from 'react';

interface ClipboardButtonProps {
	text: string;
}

export class ClipboardButton extends React.Component<ClipboardButtonProps, {}> {
	render() {
		return (
			<span
				data-to-clipboard={this.props.text}
				onClick={(e) => {
					elToClipboard(e);
				}}
				className="fauxlink margin-left">
				📋
			</span>
		);
	}
}
