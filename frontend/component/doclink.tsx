import { DocsLink } from 'f61ui/component/docslink';
import { DocRef } from 'generated/stoserver/stoservertypes_types';
import * as React from 'react';

interface DocLinkProps {
	doc: DocRef;
	title?: string;
}
export class DocLink extends React.Component<DocLinkProps, {}> {
	render() {
		return <DocsLink url={DocUrlLatest(this.props.doc)} title={this.props.title} />;
	}
}

export function DocUrlLatest(doc: DocRef): string {
	return 'https://function61.com/varasto/' + doc.replace('index.md', '').replace('.md', '/');
}
