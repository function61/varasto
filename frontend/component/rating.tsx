import { Glyphicon } from 'f61ui/component/bootstrap';
import { Result } from 'f61ui/component/result';
import { CommandExecutor } from 'f61ui/component/commandpagelet';
import { CollectionRate } from 'generated/stoserver/stoservertypes_commands';
import * as React from 'react';

const fontSizeStyle = { fontSize: '1.3em' };

interface RatingEditorProps {
	collId: string;
	rating: number | null;
}

interface RatingEditorState {
	hovered?: number;
	saveProgress: Result<void>;
}

export class RatingEditor extends React.Component<RatingEditorProps, RatingEditorState> {
	state: RatingEditorState = {
		saveProgress: new Result<void>((x) => {
			this.setState({ saveProgress: x });
		}),
	};

	render() {
		const [, loadingOrError] = this.state.saveProgress.unwrap();

		return (
			<span style={fontSizeStyle}>
				{this.makeStar(1)}
				{this.makeStar(2)}
				{this.makeStar(3)}
				{this.makeStar(4)}
				{this.makeStar(5)}

				{loadingOrError}
			</span>
		);
	}

	private makeStar(n: number) {
		// either consider what's hovered or what's the given rating.
		// if no rating default to 0 (= visually less than the worst rating).
		const rating = this.state.hovered || this.props.rating || 0;

		const lit = rating >= n;

		return (
			<span
				style={{ color: lit ? 'gold' : '#c0c0c0', cursor: 'pointer' }}
				onClick={() => {
					this.state.saveProgress.load(() =>
						new CommandExecutor(
							CollectionRate(this.props.collId, n),
						).executeAndRedirectOnSuccess(),
					);
				}}
				onMouseOver={() => {
					this.setState({ hovered: n });
				}}
				onMouseOut={() => {
					this.setState({ hovered: undefined });
				}}>
				<Glyphicon icon={lit ? 'star' : 'star-empty'} />
			</span>
		);
	}
}

interface RatingViewerProps {
	rating: number | null;
}

export class RatingViewer extends React.Component<RatingViewerProps, {}> {
	render() {
		return (
			<span style={fontSizeStyle}>
				{this.makeStar(1)}
				{this.makeStar(2)}
				{this.makeStar(3)}
				{this.makeStar(4)}
				{this.makeStar(5)}
			</span>
		);
	}

	private makeStar(n: number) {
		const rating = this.props.rating || 0;

		const lit = rating >= n;

		return (
			<span style={{ color: lit ? 'gold' : '#c0c0c0' }}>
				<Glyphicon icon={lit ? 'star' : 'star-empty'} />
			</span>
		);
	}
}
