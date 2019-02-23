import * as React from 'react';

interface ProgressBarProps {
	progress: number; // 0-100 [%]
}

export class ProgressBar extends React.Component<ProgressBarProps, {}> {
	render() {
		const pct = Math.round(this.props.progress) + '%';

		return (
			<div className="progress" title={pct}>
				<div
					className="progress-bar progress-bar-info progress-bar-striped"
					style={{ width: pct }}
				/>
			</div>
		);
	}
}
