import { DocRef } from 'generated/stoserver/stoservertypes_types';
import * as React from 'react';

interface DocLinkProps {
	doc: DocRef;
}

export class DocLink extends React.Component<DocLinkProps, {}> {
	render() {
		return (
			<a
				href={'https://github.com/function61/varasto/blob/master/' + this.props.doc}
				title="View documentation"
				target="_blank">
				<span className="glyphicon glyphicon-question-sign" />
			</a>
		);
	}
}
