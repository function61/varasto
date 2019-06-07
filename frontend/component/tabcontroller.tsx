import { getCurrentHash } from 'f61ui/browserutils';
import { jsxChildType } from 'f61ui/types';
import * as React from 'react';

export interface Tab {
	url: string;
	title: string;
	content: jsxChildType;
}

interface TabControllerProps {
	tabs: Tab[];
}

export class TabController extends React.Component<TabControllerProps, {}> {
	render() {
		const currentTabUrl = getCurrentHash();
		const activeTabs = this.props.tabs.filter((tab) => tab.url === currentTabUrl);

		return (
			<div>
				<ul className="nav nav-tabs" role="tablist">
					{this.props.tabs.map((tab) => (
						<li
							role="presentation"
							className={
								activeTabs[0] && activeTabs[0].url === tab.url ? 'active' : ''
							}>
							<a href={tab.url}>{tab.title}</a>
						</li>
					))}
				</ul>
				<div className="tab-content">{activeTabs[0] ? activeTabs[0].content : ''}</div>
			</div>
		);
	}
}
