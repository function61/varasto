import { DocsLink } from 'f61ui/component/docslink';
import { DocRef } from 'generated/stoserver/stoservertypes_types';
import * as React from 'react';

interface DocLinkProps {
	doc: DocRef;
	title?: string;
}
export class DocLink extends React.Component<DocLinkProps, {}> {
	render() {
		return <DocsLink url={DocGitHubMaster(this.props.doc)} title={this.props.title} />;
	}
}

export function DocGitHubMaster(doc: DocRef): string {
	return 'https://github.com/function61/varasto/blob/master/' + doc;
}
