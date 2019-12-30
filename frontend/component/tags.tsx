import { Glyphicon } from 'f61ui/component/bootstrap';
import { CommandIcon } from 'f61ui/component/CommandButton';
import { CollectionTag, CollectionUntag } from 'generated/stoserver/stoservertypes_commands';
import { CollectionSubset } from 'generated/stoserver/stoservertypes_types';
import * as React from 'react';

interface CollectionTagViewProps {
	collection: CollectionSubset;
}

export class CollectionTagView extends React.Component<CollectionTagViewProps, {}> {
	render() {
		const coll = this.props.collection; // shorthand

		return (
			<span>
				{coll.Tags.map((tag) => (
					<span className="label label-default margin-left">
						<Glyphicon icon="tag" />
						&nbsp;
						{tag}
					</span>
				))}
			</span>
		);
	}
}

interface CollectionTagEditorProps {
	collection: CollectionSubset;
}

export class CollectionTagEditor extends React.Component<CollectionTagEditorProps, {}> {
	render() {
		const coll = this.props.collection; // shorthand

		return (
			<span>
				{coll.Tags.map((tag) => (
					<span className="label label-default margin-left">
						<Glyphicon icon="tag" />
						&nbsp;
						{tag}
						&nbsp;
						<CommandIcon
							command={CollectionUntag(coll.Id, tag, { disambiguation: tag })}
						/>
					</span>
				))}
				<span className="label label-default margin-left">
					<Glyphicon icon="tag" />
					<span className="margin-left">
						<CommandIcon command={CollectionTag(coll.Id)} />
					</span>
				</span>
			</span>
		);
	}
}
